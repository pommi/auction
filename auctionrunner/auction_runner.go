package auctionrunner

import (
	"os"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/auctioneer"
	"code.cloudfoundry.org/workpool"
)

const (
	maxRetries = 5
)

type auctionRunner struct {
	logger lager.Logger

	delegate                auctiontypes.AuctionRunnerDelegate
	metricEmitter           auctiontypes.AuctionMetricEmitterDelegate
	batch                   *Batch
	clock                   clock.Clock
	workPool                *workpool.WorkPool
	startingContainerWeight float64
	identifier              string
}

func New(
	logger lager.Logger,
	delegate auctiontypes.AuctionRunnerDelegate,
	metricEmitter auctiontypes.AuctionMetricEmitterDelegate,
	clock clock.Clock,
	workPool *workpool.WorkPool,
	startingContainerWeight float64,
	identifier string,
) *auctionRunner {
	return &auctionRunner{
		logger: logger,

		delegate:                delegate,
		metricEmitter:           metricEmitter,
		batch:                   NewBatch(clock),
		clock:                   clock,
		workPool:                workPool,
		startingContainerWeight: startingContainerWeight,
		identifier:              identifier,
	}
}

func (a *auctionRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	var hasWork chan struct{}
	hasWork = a.batch.HasWork

	for {
		select {
		case <-hasWork:
			logger := a.logger.Session("auction", lager.Data{"id": a.identifier})

			logger.Info("fetching-cell-reps")
			clients, err := a.delegate.FetchCellReps()
			if err != nil {
				logger.Error("failed-to-fetch-reps", err)
				time.Sleep(time.Second)
				hasWork = make(chan struct{}, 1)
				hasWork <- struct{}{}
				break
			}
			logger.Info("fetched-cell-reps", lager.Data{"cell-reps-count": len(clients)})

			hasWork = a.batch.HasWork

			logger.Info("fetching-zone-state")
			fetchStatesStartTime := time.Now()
			zones := FetchStateAndBuildZones(logger, a.workPool, clients, a.metricEmitter)
			fetchStateDuration := time.Since(fetchStatesStartTime)
			err = a.metricEmitter.FetchStatesCompleted(fetchStateDuration)
			if err != nil {
				logger.Error("failed-sending-fetch-states-completed-metric", err)
			}

			cellCount := 0
			for zone, cells := range zones {
				logger.Info("zone-state", lager.Data{"zone": zone, "cell-count": len(cells)})
				cellCount += len(cells)
			}
			logger.Info("fetched-zone-state", lager.Data{
				"cell-state-count":    cellCount,
				"num-failed-requests": len(clients) - cellCount,
				"duration":            fetchStateDuration.String(),
			})

			logger.Info("fetching-auctions")
			lrpAuctions, taskAuctions := a.batch.DedupeAndDrain()
			logger.Info("fetched-auctions", lager.Data{
				"lrp-start-auctions": len(lrpAuctions),
				"task-auctions":      len(taskAuctions),
			})
			if len(lrpAuctions) == 0 && len(taskAuctions) == 0 {
				logger.Info("nothing-to-auction")
				break
			}

			logger.Info("scheduling")
			auctionRequest := auctiontypes.AuctionRequest{
				LRPs:  lrpAuctions,
				Tasks: taskAuctions,
			}

			scheduler := NewScheduler(a.workPool, zones, a.clock, logger, a.startingContainerWeight)
			auctionResults := scheduler.Schedule(auctionRequest)
			logger.Info("scheduled", lager.Data{
				"successful-lrp-start-auctions": len(auctionResults.SuccessfulLRPs),
				"successful-task-auctions":      len(auctionResults.SuccessfulTasks),
				"failed-lrp-start-auctions":     len(auctionResults.FailedLRPs),
				"failed-task-auctions":          len(auctionResults.FailedTasks),
			})

			numStartsFailed := len(auctionResults.FailedLRPs)
			numTasksFailed := len(auctionResults.FailedTasks)

			logger.Info("resubmitting-failures")
			auctionResults = ResubmitFailedAuctions(a.batch, auctionResults, maxRetries)
			logger.Info("resubmitted-failures", lager.Data{
				"successful-lrp-start-auctions":     len(auctionResults.SuccessfulLRPs),
				"successful-task-auctions":          len(auctionResults.SuccessfulTasks),
				"will-not-retry-lrp-start-auctions": len(auctionResults.FailedLRPs),
				"will-not-retry-task-auctions":      len(auctionResults.FailedTasks),
				"will-retry-lrp-start-auctions":     numStartsFailed - len(auctionResults.FailedLRPs),
				"will-retry-task-auctions":          numTasksFailed - len(auctionResults.FailedTasks),
			})

			a.metricEmitter.AuctionCompleted(auctionResults)
			a.delegate.AuctionCompleted(auctionResults)
		case <-signals:
			return nil
		}
	}
}

func (a *auctionRunner) ScheduleLRPsForAuctions(lrpStarts []auctioneer.LRPStartRequest) {
	a.batch.AddLRPStarts(lrpStarts)
}

func (a *auctionRunner) ScheduleTasksForAuctions(tasks []auctioneer.TaskStartRequest) {
	a.batch.AddTasks(tasks)
}

func (a *auctionRunner) Identifier() string {
	return a.identifier
}
