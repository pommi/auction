package auctionrunner

import (
	"os"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/pivotal-golang/lager"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry/gunk/workpool"
)

type auctionRunner struct {
	delegate     auctiontypes.AuctionRunnerDelegate
	batch        *Batch
	timeProvider timeprovider.TimeProvider
	workPool     *workpool.WorkPool
	maxRetries   int
	logger       lager.Logger
}

func New(delegate auctiontypes.AuctionRunnerDelegate, timeProvider timeprovider.TimeProvider, maxRetries int, workPool *workpool.WorkPool, logger lager.Logger) *auctionRunner {
	return &auctionRunner{
		delegate:     delegate,
		batch:        NewBatch(timeProvider),
		timeProvider: timeProvider,
		workPool:     workPool,
		maxRetries:   maxRetries,
		logger:       logger,
	}
}

func (a *auctionRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	var hasWork chan struct{}
	hasWork = a.batch.HasWork

	for {
		select {
		case <-hasWork:
			logger := a.logger.Session("auction")

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

			logger.Info("fetching-cell-state")
			cells := FetchStateAndBuildCells(a.workPool, clients)
			logger.Info("fetched-cell-state", lager.Data{"cell-state-count": len(cells), "num-failed-requests": len(clients) - len(cells)})

			logger.Info("fetching-auctions")
			lrpStartAuctions, taskAuctions := a.batch.DedupeAndDrain()
			logger.Info("fetched-auctions", lager.Data{
				"lrp-start-auctions": len(lrpStartAuctions),
				"task-auctions":      len(taskAuctions),
			})
			if len(lrpStartAuctions) == 0 && len(taskAuctions) == 0 {
				logger.Info("nothing-to-auction")
				break
			}

			logger.Info("scheduling")
			auctionRequest := auctiontypes.AuctionRequest{
				LRPStarts: lrpStartAuctions,
				Tasks:     taskAuctions,
			}
			auctionResults := Schedule(a.workPool, cells, a.timeProvider, auctionRequest)
			logger.Info("scheduled", lager.Data{
				"successful-lrp-start-auctions": len(auctionResults.SuccessfulLRPStarts),
				"successful-task-auctions":      len(auctionResults.SuccessfulTasks),
				"failed-lrp-start-auctions":     len(auctionResults.FailedLRPStarts),
				"failed-task-auctions":          len(auctionResults.FailedTasks),
			})
			numStartsFailed := len(auctionResults.FailedLRPStarts)
			numTasksFailed := len(auctionResults.FailedTasks)

			logger.Info("resubmitting-failures")
			auctionResults = ResubmitFailedAuctions(a.batch, auctionResults, a.maxRetries)
			logger.Info("resubmitted-failures", lager.Data{
				"successful-lrp-start-auctions":     len(auctionResults.SuccessfulLRPStarts),
				"successful-task-auctions":          len(auctionResults.SuccessfulTasks),
				"will-not-retry-lrp-start-auctions": len(auctionResults.FailedLRPStarts),
				"will-not-retry-task-auctions":      len(auctionResults.FailedTasks),
				"will-retry-lrp-start-auctions":     numStartsFailed - len(auctionResults.FailedLRPStarts),
				"will-retry-task-auctions":          numTasksFailed - len(auctionResults.FailedTasks),
			})

			go a.delegate.DistributedBatch(auctionResults)
		case <-signals:
			return nil
		}
	}
}

func (a *auctionRunner) ScheduleLRPStartsForAuctions(lrpStarts []models.LRPStart) {
	a.batch.AddLRPStarts(lrpStarts)
}

func (a *auctionRunner) ScheduleTasksForAuctions(tasks []models.Task) {
	a.batch.AddTasks(tasks)
}
