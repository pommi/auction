package auctionrunner

import (
	"sort"
	"sync"

	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/rep"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/workpool"
)

const (
	MaxTaskRetries = 3
)

type Zone []*Cell

func (z *Zone) filterCells(pc rep.PlacementConstraint) ([]*Cell, error) {
	var cells = make([]*Cell, 0, len(*z))
	err := auctiontypes.ErrorCellMismatch

	for _, cell := range *z {
		if cell.MatchRootFS(pc.RootFs) {
			if err == auctiontypes.ErrorCellMismatch {
				err = auctiontypes.ErrorVolumeDriverMismatch
			}

			if cell.MatchVolumeDrivers(pc.VolumeDrivers) {
				if err == auctiontypes.ErrorVolumeDriverMismatch {
					err = auctiontypes.NewPlacementTagMismatchError(pc.PlacementTags)
				}

				if cell.MatchPlacementTags(pc.PlacementTags) {
					err = nil
					cells = append(cells, cell)
				}
			}
		}
	}

	return cells, err
}

type Scheduler struct {
	workPool                      *workpool.WorkPool
	zones                         map[string]Zone
	clock                         clock.Clock
	logger                        lager.Logger
	startingContainerWeight       float64
	startingContainerCountMaximum int // <=0 means no limit
	internalResults               auctiontypes.AuctionResults
}

func NewScheduler(
	workPool *workpool.WorkPool,
	zones map[string]Zone,
	clock clock.Clock,
	logger lager.Logger,
	startingContainerWeight float64,
	startingContainerCountMaximum int,
) *Scheduler {
	return &Scheduler{
		workPool:                      workPool,
		zones:                         zones,
		clock:                         clock,
		logger:                        logger,
		startingContainerWeight:       startingContainerWeight,
		startingContainerCountMaximum: startingContainerCountMaximum,
		internalResults:               auctiontypes.AuctionResults{},
	}
}

/*
Schedule takes in a set of job requests (LRP start auctions and task starts) and
assigns the work to available cells according to the diego scoring algorithm. The
scheduler is single-threaded.  It determines scheduling of jobs one at a time so
that each calculation reflects available resources correctly.  It commits the
work in batches at the end, for better network performance.  Schedule returns
AuctionResults, indicating the success or failure of each requested job.
*/
func (s *Scheduler) Schedule(auctionRequest auctiontypes.AuctionRequest) (auctiontypes.AuctionResults, auctiontypes.AuctionResults) {

	attempted := false
	if len(s.zones) == 0 {
		s.internalResults.FailedLRPs = auctionRequest.LRPs
		for i, _ := range s.internalResults.FailedLRPs {
			s.internalResults.FailedLRPs[i].PlacementError = auctiontypes.ErrorCellCommunication.Error()
		}
		s.internalResults.FailedTasks = auctionRequest.Tasks
		for i, _ := range s.internalResults.FailedTasks {
			s.internalResults.FailedTasks[i].PlacementError = auctiontypes.ErrorCellCommunication.Error()
		}
		attempted := true
		return s.markResults(attempted)
	}

	var successfulLRPs = map[string]*auctiontypes.LRPAuction{}
	var lrpStartAuctionLookup = map[string]*auctiontypes.LRPAuction{}
	var successfulTasks = map[string]*auctiontypes.TaskAuction{}
	var taskAuctionLookup = map[string]*auctiontypes.TaskAuction{}
	var currentInflightContainerStarts int

	for _, zone := range s.zones {
		for _, cell := range zone {
			currentInflightContainerStarts += cell.StartingContainerCount()
		}
	}

	sort.Sort(SortableLRPAuctions(auctionRequest.LRPs))
	sort.Sort(SortableTaskAuctions(auctionRequest.Tasks))

	lrpsBeforeTasks, lrpsAfterTasks := splitLRPS(auctionRequest.LRPs)

	auctionLRP := func(lrpsToAuction []auctiontypes.LRPAuction) {
		for i := range lrpsToAuction {
			lrpAuction := &lrpsToAuction[i]
			lrpStartAuctionLookup[lrpAuction.Identifier()] = lrpAuction

			if s.exceededInflightContainerCreation(currentInflightContainerStarts) {
				s.logger.Info(
					"exceeded-max-inflight-container-creation",
					lager.Data{
						"max-inflight": s.startingContainerCountMaximum,
						"lrp-guid":     lrpAuction.Identifier(),
					},
				)
				lrpAuction.PlacementError = auctiontypes.ErrorExceededInflightCreation.Error()
				s.internalResults.FailedLRPs = append(s.internalResults.FailedLRPs, *lrpAuction)
				continue
			}

			successfulStart, err := s.scheduleLRPAuction(lrpAuction)
			if err != nil {
				lrpAuction.PlacementError = err.Error()
				s.internalResults.FailedLRPs = append(s.internalResults.FailedLRPs, *lrpAuction)
			} else {
				successfulLRPs[successfulStart.Identifier()] = successfulStart
				currentInflightContainerStarts++
			}
		}
	}

	auctionLRP(lrpsBeforeTasks)

	for i := range auctionRequest.Tasks {
		taskAuction := &auctionRequest.Tasks[i]
		taskAuctionLookup[taskAuction.Identifier()] = taskAuction

		if s.exceededInflightContainerCreation(currentInflightContainerStarts) {
			s.logger.Info(
				"exceeded-max-inflight-container-creation",
				lager.Data{
					"max-inflight": s.startingContainerCountMaximum,
					"task-guid":    taskAuction.Identifier(),
				},
			)
			taskAuction.PlacementError = auctiontypes.ErrorExceededInflightCreation.Error()
			s.internalResults.FailedTasks = append(s.internalResults.FailedTasks, *taskAuction)
			continue
		}

		successfulTask, err := s.scheduleTaskAuction(taskAuction, s.startingContainerWeight)
		if err != nil {
			attempted = true
			taskAuction.PlacementError = err.Error()
			s.internalResults.FailedTasks = append(s.internalResults.FailedTasks, *taskAuction)
		} else {
			successfulTasks[successfulTask.Identifier()] = successfulTask
			currentInflightContainerStarts++
		}
	}

	auctionLRP(lrpsAfterTasks)

	failedWorks := s.commitCells()
	for _, failedWork := range failedWorks {
		for _, failedStart := range failedWork.LRPs {
			identifier := failedStart.Identifier()
			delete(successfulLRPs, identifier)

			s.logger.Info("lrp-failed-to-be-placed", lager.Data{"lrp-guid": failedStart.Identifier()})
			s.internalResults.FailedLRPs = append(s.internalResults.FailedLRPs, *lrpStartAuctionLookup[identifier])
		}

		for _, failedTask := range failedWork.Tasks {
			identifier := failedTask.Identifier()
			delete(successfulTasks, identifier)

			s.logger.Info("task-failed-to-be-placed", lager.Data{"task-guid": failedTask.Identifier()})
			s.internalResults.FailedTasks = append(s.internalResults.FailedTasks, *taskAuctionLookup[identifier])
		}
	}

	for _, successfulStart := range successfulLRPs {
		s.logger.Info("lrp-added-to-cell", lager.Data{"lrp-guid": successfulStart.Identifier(), "cell-guid": successfulStart.Winner})
		s.internalResults.SuccessfulLRPs = append(s.internalResults.SuccessfulLRPs, *successfulStart)
	}
	for _, successfulTask := range successfulTasks {
		s.logger.Info("task-added-to-cell", lager.Data{"task-guid": successfulTask.Identifier(), "cell-guid": successfulTask.Winner})
		s.internalResults.SuccessfulTasks = append(s.internalResults.SuccessfulTasks, *successfulTask)
	}
	return s.markResults(attempted)
}

func (s *Scheduler) markResults(attempted bool) (auctiontypes.AuctionResults, auctiontypes.AuctionResults) {
	results := auctiontypes.AuctionResults{}
	retries := auctiontypes.AuctionResults{}

	now := s.clock.Now()
	for _, lrpAuction := range s.internalResults.FailedLRPs {
		lrpAuction.Attempts++
		results.FailedLRPs = append(results.FailedLRPs, lrpAuction)
	}
	for _, taskAuction := range s.internalResults.FailedTasks {
		taskAuction.Attempts++
		logger := s.logger.Session("failed-task", lager.Data{"attempts": taskAuction.Attempts, "guid": taskAuction.TaskGuid})
		if !attempted || taskAuction.Attempts > MaxTaskRetries {
			logger.Info("gave-up-retry", lager.Data{"attempted?": attempted})
			results.FailedTasks = append(results.FailedTasks, taskAuction)
		} else {
			logger.Info("retrying-task")
			retries.FailedTasks = append(results.FailedTasks, taskAuction)
		}
	}
	for _, lrpAuction := range s.internalResults.SuccessfulLRPs {
		lrpAuction.Attempts++
		lrpAuction.WaitDuration = now.Sub(lrpAuction.QueueTime)
		results.SuccessfulLRPs = append(results.SuccessfulLRPs, lrpAuction)
	}
	for _, taskAuction := range s.internalResults.SuccessfulTasks {
		taskAuction.Attempts++
		taskAuction.WaitDuration = now.Sub(taskAuction.QueueTime)
		results.SuccessfulTasks = append(results.SuccessfulTasks, taskAuction)
	}

	return results, retries
}

func splitLRPS(lrps []auctiontypes.LRPAuction) ([]auctiontypes.LRPAuction, []auctiontypes.LRPAuction) {
	const pivot = 0

	for idx, lrp := range lrps {
		if lrp.Index > pivot {
			return lrps[:idx], lrps[idx:]
		}
	}

	return lrps[:0], lrps[0:]
}

func (s *Scheduler) commitCells() []rep.Work {
	wg := &sync.WaitGroup{}
	for _, cells := range s.zones {
		wg.Add(len(cells))
	}

	lock := &sync.Mutex{}
	failedWorks := []rep.Work{}

	for _, cells := range s.zones {
		for _, cell := range cells {
			cell := cell
			s.workPool.Submit(func() {
				defer wg.Done()
				failedWork := cell.Commit()

				lock.Lock()
				failedWorks = append(failedWorks, failedWork)
				lock.Unlock()
			})
		}
	}

	wg.Wait()
	return failedWorks
}

func (s *Scheduler) scheduleLRPAuction(lrpAuction *auctiontypes.LRPAuction) (*auctiontypes.LRPAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	zones := accumulateZonesByInstances(s.zones, lrpAuction.ProcessGuid)

	filteredZones, err := filterZones(zones, lrpAuction)
	if err != nil {
		return nil, err
	}

	sortedZones := sortZonesByInstances(filteredZones)
	problems := map[string]struct{}{"disk": struct{}{}, "memory": struct{}{}, "containers": struct{}{}}

	for zoneIndex, lrpByZone := range sortedZones {
		for _, cell := range lrpByZone.zone {
			score, err := cell.ScoreForLRP(&lrpAuction.LRP, s.startingContainerWeight)
			if err != nil {
				removeNonApplicableProblems(problems, err)
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}

		// if (not last zone) && (this zone has the same # of instances as the next sorted zone)
		// acts as a tie breaker
		if zoneIndex+1 < len(sortedZones) &&
			lrpByZone.instances == sortedZones[zoneIndex+1].instances {
			continue
		}

		if winnerCell != nil {
			break
		}
	}

	if winnerCell == nil {
		err := &rep.InsufficientResourcesError{Problems: problems}
		s.logger.Error("lrp-auction-failed", err, lager.Data{"lrp-guid": lrpAuction.Identifier()})
		return nil, err
	}

	err = winnerCell.ReserveLRP(&lrpAuction.LRP)
	if err != nil {
		s.logger.Error("lrp-failed-to-reserve-cell", err, lager.Data{"cell-guid": winnerCell.Guid, "lrp-guid": lrpAuction.Identifier()})
		return nil, err
	}

	winningAuction := lrpAuction.Copy()
	winningAuction.Winner = winnerCell.Guid
	return &winningAuction, nil
}

func (s *Scheduler) scheduleTaskAuction(taskAuction *auctiontypes.TaskAuction, startingContainerWeight float64) (*auctiontypes.TaskAuction, error) {
	var winnerCell *Cell
	winnerScore := 1e20

	filteredZones := []Zone{}
	var zoneError error

	for _, zone := range s.zones {
		cells, err := zone.filterCells(taskAuction.PlacementConstraint)
		if err != nil {
			_, isZoneErrorPlacementTagMismatchError := zoneError.(auctiontypes.PlacementTagMismatchError)
			_, isErrPlacementTagMismatchError := err.(auctiontypes.PlacementTagMismatchError)

			if isZoneErrorPlacementTagMismatchError ||
				(zoneError == auctiontypes.ErrorVolumeDriverMismatch && isErrPlacementTagMismatchError) ||
				zoneError == auctiontypes.ErrorCellMismatch || zoneError == nil {
				zoneError = err
			}
			continue
		}

		filteredZones = append(filteredZones, Zone(cells))
	}

	if len(filteredZones) == 0 {
		return nil, zoneError
	}

	problems := map[string]struct{}{"disk": struct{}{}, "memory": struct{}{}, "containers": struct{}{}}

	for _, zone := range filteredZones {
		for _, cell := range zone {
			score, err := cell.ScoreForTask(&taskAuction.Task, startingContainerWeight)
			if err != nil {
				removeNonApplicableProblems(problems, err)
				continue
			}

			if score < winnerScore {
				winnerScore = score
				winnerCell = cell
			}
		}
	}

	if winnerCell == nil {
		err := &rep.InsufficientResourcesError{Problems: problems}
		s.logger.Error("task-auction-failed", err, lager.Data{"task-guid": taskAuction.Identifier()})
		return nil, err
	}

	err := winnerCell.ReserveTask(&taskAuction.Task)
	if err != nil {
		s.logger.Error("task-failed-to-reserve-cell", err, lager.Data{"cell-guid": winnerCell.Guid, "task-guid": taskAuction.Identifier()})
		return nil, err
	}

	winningAuction := taskAuction.Copy()
	winningAuction.Winner = winnerCell.Guid
	return &winningAuction, nil
}

// removeNonApplicableProblems modifies the 'problems' map to remove any problems that didn't show up on err.
//
// The list of problems to report should only consist of the problems that exist on every cell
// For example, if there is not enough memory on one cell and not enough disk on another, we should
// not call out memory or disk as being a specific problem.
func removeNonApplicableProblems(problems map[string]struct{}, err error) {
	if ierr, ok := err.(rep.InsufficientResourcesError); ok {
		for problem, _ := range problems {
			if _, ok := ierr.Problems[problem]; !ok {
				delete(problems, problem)
			}
		}
	}
}

func (s *Scheduler) exceededInflightContainerCreation(currentInflight int) bool {
	return s.startingContainerCountMaximum > 0 && currentInflight >= s.startingContainerCountMaximum
}
