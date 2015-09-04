package auctionrunner_test

import (
	"time"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auctioneer"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/rep"
	. "github.com/onsi/gomega"
)

func BuildLRPStartRequest(processGuid string, indices []int, rootFS string, memoryMB, diskMB int32) auctioneer.LRPStartRequest {
	return auctioneer.NewLRPStartRequest(processGuid, indices, rep.NewResource(memoryMB, diskMB, rootFS))
}

func BuildTaskStartRequest(taskGuid, rootFS string, memoryMB, diskMB int32) auctioneer.TaskStartRequest {
	return auctioneer.NewTaskStartRequest(*BuildTask(taskGuid, rootFS, memoryMB, diskMB))
}

func BuildLRP(guid string, index int, rootFS string, memoryMB, diskMB int32) *rep.LRP {
	lrp := rep.NewLRP(guid, index, rep.NewResource(memoryMB, diskMB, rootFS))
	return &lrp
}

func BuildTask(taskGuid, rootFS string, memoryMB, diskMB int32) *rep.Task {
	task := rep.NewTask(taskGuid, rep.NewResource(memoryMB, diskMB, rootFS))
	return &task
}

func BuildLRPAuction(processGuid string, index int, rootFS string, memoryMB, diskMB int32, queueTime time.Time) auctiontypes.LRPAuction {
	return auctiontypes.NewLRPAuction(rep.NewLRP(processGuid, index, rep.NewResource(memoryMB, diskMB, rootFS)), queueTime)
}

func BuildLRPAuctionWithPlacementError(processGuid string, index int, rootFS string, memoryMB, diskMB int32, queueTime time.Time, placementError string) auctiontypes.LRPAuction {
	a := auctiontypes.NewLRPAuction(rep.NewLRP(processGuid, index, rep.NewResource(memoryMB, diskMB, rootFS)), queueTime)
	a.PlacementError = placementError
	return a
}

func BuildLRPAuctions(start auctioneer.LRPStartRequest, queueTime time.Time) []auctiontypes.LRPAuction {
	auctions := make([]auctiontypes.LRPAuction, 0, len(start.Indices))
	for _, index := range start.Indices {
		auctions = append(auctions, auctiontypes.NewLRPAuction(rep.NewLRP(start.ProcessGuid, int(index), start.Resource), queueTime))
	}

	return auctions
}

func BuildTaskAuction(task *rep.Task, queueTime time.Time) auctiontypes.TaskAuction {
	return auctiontypes.NewTaskAuction(*task, queueTime)
}

const linuxStack = "linux"

var linuxRootFSURL = models.PreloadedRootFS(linuxStack)

var linuxOnlyRootFSProviders = rep.RootFSProviders{models.PreloadedRootFSScheme: rep.NewFixedSetRootFSProvider(linuxStack)}

const windowsStack = "windows"

var windowsRootFSURL = models.PreloadedRootFS(windowsStack)

var windowsOnlyRootFSProviders = rep.RootFSProviders{models.PreloadedRootFSScheme: rep.NewFixedSetRootFSProvider(windowsStack)}

func BuildCellState(
	zone string,
	memoryMB int32,
	diskMB int32,
	containers int,
	evacuating bool,
	rootFSProviders rep.RootFSProviders,
	lrps []rep.LRP,
) rep.CellState {
	totalResources := rep.NewResources(memoryMB, diskMB, containers)

	availableResources := totalResources.Copy()
	for i := range lrps {
		availableResources.Subtract(&lrps[i].Resource)
	}

	Expect(availableResources.MemoryMB).To(BeNumerically(">=", 0), "Check your math!")
	Expect(availableResources.DiskMB).To(BeNumerically(">=", 0), "Check your math!")
	Expect(availableResources.Containers).To(BeNumerically(">=", 0), "Check your math!")

	return rep.NewCellState(rootFSProviders, availableResources, totalResources, lrps, nil, zone, evacuating)
}
