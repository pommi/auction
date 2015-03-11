package auctionrunner

import (
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type Cell struct {
	Guid   string
	client auctiontypes.CellRep
	state  auctiontypes.CellState

	workToCommit auctiontypes.Work
}

func NewCell(guid string, client auctiontypes.CellRep, state auctiontypes.CellState) *Cell {
	return &Cell{
		Guid:   guid,
		client: client,
		state:  state,
	}
}

func (c *Cell) Stack() string {
	return c.state.Stack
}

//writeup: here more than one container per volume breaks hard
func (c *Cell) subtractVolumeResources(vol auctiontypes.VolumeAuction) auctiontypes.Resources {
	remainingResources := c.state.AvailableResources
	remainingResources.MemoryMB -= vol.ReservedMemoryMB
	remainingResources.PersistentDiskMB -= vol.SizeMB
	remainingResources.Containers -= 1
	return remainingResources
}

func (c *Cell) subtractLRPResources(lrpAuction auctiontypes.LRPAuction) auctiontypes.Resources {
	remainingResources := c.state.AvailableResources
	var mem, container int
	vol, err := c.volumeForLRP(lrpAuction)
	switch err {
	case nil:
		mem = lrpAuction.DesiredLRP.MemoryMB - vol.ReservedMemoryMB
		if mem < 0 {
			mem = 0
		}
		container = 0

	case auctiontypes.ErrNoVolumeMount:
		mem = lrpAuction.DesiredLRP.MemoryMB
		container = 1

	default:
		panic("should be impossible")
	}

	remainingResources.MemoryMB -= mem
	remainingResources.DiskMB -= lrpAuction.DesiredLRP.DiskMB
	remainingResources.Containers -= container

	return remainingResources
}
func (c *Cell) ScoreForVolumeAuction(vol auctiontypes.VolumeAuction) (float64, error) {
	err := c.canHandleVolumeAuction(vol)
	if err != nil {
		return 0, err
	}

	numberOfVolumesWithMatchingVolumeSetGuid := 0
	for i := range c.state.Volumes {
		if vol.VolumeSetGuid == c.state.Volumes[i].VolumeSetGuid {
			numberOfVolumesWithMatchingVolumeSetGuid++
		}
	}

	remainingResources := c.subtractVolumeResources(vol)
	resourceScore := c.computeVolumeScore(remainingResources, numberOfVolumesWithMatchingVolumeSetGuid)

	return resourceScore, nil
}

func (c *Cell) ScoreForLRPAuction(lrpAuction auctiontypes.LRPAuction) (float64, error) {
	err := c.canHandleLRPAuction(lrpAuction)
	if err != nil {
		return 0, err
	}

	numberOfInstancesWithMatchingProcessGuid := 0
	for _, lrp := range c.state.LRPs {
		if lrp.ProcessGuid == lrpAuction.DesiredLRP.ProcessGuid {
			numberOfInstancesWithMatchingProcessGuid++
		}
	}

	remainingResources := c.subtractLRPResources(lrpAuction)
	resourceScore := c.computeScore(remainingResources, numberOfInstancesWithMatchingProcessGuid)

	return resourceScore, nil
}

func (c *Cell) ScoreForTask(task models.Task) (float64, error) {
	err := c.canHandleTask(task)
	if err != nil {
		return 0, err
	}

	remainingResources := c.state.AvailableResources
	remainingResources.MemoryMB -= task.MemoryMB
	remainingResources.DiskMB -= task.DiskMB
	remainingResources.Containers -= 1

	resourceScore := c.computeTaskScore(remainingResources)

	return resourceScore, nil
}

func (c *Cell) ReserveVolume(vol auctiontypes.VolumeAuction) error {
	err := c.canHandleVolumeAuction(vol)
	if err != nil {
		return err
	}

	c.state.Volumes = append(c.state.Volumes, models.Volume{
		VolumeSetGuid:    vol.VolumeSetGuid,
		CellID:           c.Guid,
		Index:            vol.Index,
		SizeMB:           vol.SizeMB,
		ReservedMemoryMB: vol.ReservedMemoryMB,
	})

	c.state.AvailableResources = c.subtractVolumeResources(vol)
	c.workToCommit.Volumes = append(c.workToCommit.Volumes, vol)
	return nil
}

func (c *Cell) ReserveLRP(lrpAuction auctiontypes.LRPAuction) error {
	err := c.canHandleLRPAuction(lrpAuction)
	if err != nil {
		return err
	}

	c.state.LRPs = append(c.state.LRPs, auctiontypes.LRP{
		ProcessGuid: lrpAuction.DesiredLRP.ProcessGuid,
		Index:       lrpAuction.Index,
		MemoryMB:    lrpAuction.DesiredLRP.MemoryMB,
		DiskMB:      lrpAuction.DesiredLRP.DiskMB,
	})

	c.state.AvailableResources = c.subtractLRPResources(lrpAuction)
	c.workToCommit.LRPs = append(c.workToCommit.LRPs, lrpAuction)

	return nil
}

func (c *Cell) ReserveTask(task models.Task) error {
	err := c.canHandleTask(task)
	if err != nil {
		return err
	}

	c.state.Tasks = append(c.state.Tasks, auctiontypes.Task{
		TaskGuid: task.TaskGuid,
		MemoryMB: task.MemoryMB,
		DiskMB:   task.DiskMB,
	})

	c.state.AvailableResources.MemoryMB -= task.MemoryMB
	c.state.AvailableResources.DiskMB -= task.DiskMB
	c.state.AvailableResources.Containers -= 1

	c.workToCommit.Tasks = append(c.workToCommit.Tasks, task)

	return nil
}

func (c *Cell) Commit() auctiontypes.Work {
	if len(c.workToCommit.LRPs) == 0 && len(c.workToCommit.Tasks) == 0 && len(c.workToCommit.Volumes) == 0 {
		return auctiontypes.Work{}
	}

	failedWork, err := c.client.Perform(c.workToCommit)
	if err != nil {
		//an error may indicate partial failure
		//in this case we don't reschedule work in order to make sure we don't
		//create duplicates of things -- we'll let the converger figure things out for us later
		return auctiontypes.Work{}
	}
	return failedWork
}

// writeup: if we place a volume where the contianer cannot attach, we've made a boo-boo.
// therefore, all container scheduling needs to be taken into account when scheduling the volume.
// Examples include stack, # of available containers, DiskMB, etc.
func (c *Cell) canHandleVolumeAuction(vol auctiontypes.VolumeAuction) error {
	if c.state.Stack != vol.Stack {
		return auctiontypes.ErrorCellMismatch
	}
	if c.state.AvailableResources.MemoryMB < vol.ReservedMemoryMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.PersistentDiskMB < vol.SizeMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return auctiontypes.ErrorInsufficientResources
	}

	return nil
}

func (c *Cell) canHandleLRPAuction(lrpAuction auctiontypes.LRPAuction) error {
	if c.state.Stack != lrpAuction.DesiredLRP.Stack {
		return auctiontypes.ErrorCellMismatch
	}
	if c.state.AvailableResources.MemoryMB < lrpAuction.DesiredLRP.MemoryMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.DiskMB < lrpAuction.DesiredLRP.DiskMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return auctiontypes.ErrorInsufficientResources
	}

	_, err := c.volumeForLRP(lrpAuction)
	if err != nil && err != auctiontypes.ErrNoVolumeMount {
		return err
	}
	return nil
}

func (c *Cell) canHandleTask(task models.Task) error {
	if c.state.Stack != task.Stack {
		return auctiontypes.ErrorCellMismatch
	}
	if c.state.AvailableResources.MemoryMB < task.MemoryMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.DiskMB < task.DiskMB {
		return auctiontypes.ErrorInsufficientResources
	}
	if c.state.AvailableResources.Containers < 1 {
		return auctiontypes.ErrorInsufficientResources
	}
	return nil
}

func (c *Cell) volumeForLRP(lrpAuction auctiontypes.LRPAuction) (models.Volume, error) {
	if lrpAuction.DesiredLRP.VolumeMount == nil {
		return models.Volume{}, auctiontypes.ErrNoVolumeMount
	}

	for i := range c.state.Volumes {
		if c.state.Volumes[i].VolumeSetGuid == lrpAuction.DesiredLRP.VolumeMount.VolumeSetGuid && c.state.Volumes[i].Index == lrpAuction.Index {
			return c.state.Volumes[i], nil
		}
	}

	return models.Volume{}, auctiontypes.ErrVolumeNotAvailable
}

func (c *Cell) computeVolumeScore(remainingResources auctiontypes.Resources, numInstances int) float64 {
	fractionUsedPersistentDisk := 1.0 - float64(remainingResources.PersistentDiskMB)/float64(c.state.TotalResources.PersistentDiskMB)
	fractionUsedContainers := 1.0 - float64(remainingResources.Containers)/float64(c.state.TotalResources.Containers)
	fractionUsedMemory := 1.0 - float64(remainingResources.MemoryMB)/float64(c.state.TotalResources.MemoryMB)
	fractionUsedDisk := 1.0 - float64(remainingResources.DiskMB)/float64(c.state.TotalResources.DiskMB)

	resourceScore := (fractionUsedPersistentDisk + fractionUsedMemory + fractionUsedDisk + fractionUsedContainers) / 4.0
	resourceScore += float64(numInstances)

	return resourceScore
}

func (c *Cell) computeScore(remainingResources auctiontypes.Resources, numInstances int) float64 {
	fractionUsedMemory := 1.0 - float64(remainingResources.MemoryMB)/float64(c.state.TotalResources.MemoryMB)
	fractionUsedDisk := 1.0 - float64(remainingResources.DiskMB)/float64(c.state.TotalResources.DiskMB)
	fractionUsedContainers := 1.0 - float64(remainingResources.Containers)/float64(c.state.TotalResources.Containers)

	resourceScore := (fractionUsedMemory + fractionUsedDisk + fractionUsedContainers) / 3.0
	resourceScore += float64(numInstances)

	return resourceScore
}

func (c *Cell) computeTaskScore(remainingResources auctiontypes.Resources) float64 {
	fractionUsedMemory := 1.0 - float64(remainingResources.MemoryMB)/float64(c.state.TotalResources.MemoryMB)
	fractionUsedDisk := 1.0 - float64(remainingResources.DiskMB)/float64(c.state.TotalResources.DiskMB)
	fractionUsedContainers := 1.0 - float64(remainingResources.Containers)/float64(c.state.TotalResources.Containers)

	resourceScore := (fractionUsedMemory + fractionUsedDisk + fractionUsedContainers) / 3.0

	return resourceScore
}
