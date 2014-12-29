// This file was generated by counterfeiter
package fakes

import (
	"os"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type FakeAuctionRunner struct {
	RunStub        func(signals <-chan os.Signal, ready chan<- struct{}) error
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		signals <-chan os.Signal
		ready   chan<- struct{}
	}
	runReturns struct {
		result1 error
	}
	ScheduleLRPStartsForAuctionsStub        func([]models.LRPStart)
	scheduleLRPStartsForAuctionsMutex       sync.RWMutex
	scheduleLRPStartsForAuctionsArgsForCall []struct {
		arg1 []models.LRPStart
	}
	ScheduleTasksForAuctionsStub        func([]models.Task)
	scheduleTasksForAuctionsMutex       sync.RWMutex
	scheduleTasksForAuctionsArgsForCall []struct {
		arg1 []models.Task
	}
}

func (fake *FakeAuctionRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	fake.runMutex.Lock()
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		signals <-chan os.Signal
		ready   chan<- struct{}
	}{signals, ready})
	fake.runMutex.Unlock()
	if fake.RunStub != nil {
		return fake.RunStub(signals, ready)
	} else {
		return fake.runReturns.result1
	}
}

func (fake *FakeAuctionRunner) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeAuctionRunner) RunArgsForCall(i int) (<-chan os.Signal, chan<- struct{}) {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return fake.runArgsForCall[i].signals, fake.runArgsForCall[i].ready
}

func (fake *FakeAuctionRunner) RunReturns(result1 error) {
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeAuctionRunner) ScheduleLRPStartsForAuctions(arg1 []models.LRPStart) {
	fake.scheduleLRPStartsForAuctionsMutex.Lock()
	fake.scheduleLRPStartsForAuctionsArgsForCall = append(fake.scheduleLRPStartsForAuctionsArgsForCall, struct {
		arg1 []models.LRPStart
	}{arg1})
	fake.scheduleLRPStartsForAuctionsMutex.Unlock()
	if fake.ScheduleLRPStartsForAuctionsStub != nil {
		fake.ScheduleLRPStartsForAuctionsStub(arg1)
	}
}

func (fake *FakeAuctionRunner) ScheduleLRPStartsForAuctionsCallCount() int {
	fake.scheduleLRPStartsForAuctionsMutex.RLock()
	defer fake.scheduleLRPStartsForAuctionsMutex.RUnlock()
	return len(fake.scheduleLRPStartsForAuctionsArgsForCall)
}

func (fake *FakeAuctionRunner) ScheduleLRPStartsForAuctionsArgsForCall(i int) []models.LRPStart {
	fake.scheduleLRPStartsForAuctionsMutex.RLock()
	defer fake.scheduleLRPStartsForAuctionsMutex.RUnlock()
	return fake.scheduleLRPStartsForAuctionsArgsForCall[i].arg1
}

func (fake *FakeAuctionRunner) ScheduleTasksForAuctions(arg1 []models.Task) {
	fake.scheduleTasksForAuctionsMutex.Lock()
	fake.scheduleTasksForAuctionsArgsForCall = append(fake.scheduleTasksForAuctionsArgsForCall, struct {
		arg1 []models.Task
	}{arg1})
	fake.scheduleTasksForAuctionsMutex.Unlock()
	if fake.ScheduleTasksForAuctionsStub != nil {
		fake.ScheduleTasksForAuctionsStub(arg1)
	}
}

func (fake *FakeAuctionRunner) ScheduleTasksForAuctionsCallCount() int {
	fake.scheduleTasksForAuctionsMutex.RLock()
	defer fake.scheduleTasksForAuctionsMutex.RUnlock()
	return len(fake.scheduleTasksForAuctionsArgsForCall)
}

func (fake *FakeAuctionRunner) ScheduleTasksForAuctionsArgsForCall(i int) []models.Task {
	fake.scheduleTasksForAuctionsMutex.RLock()
	defer fake.scheduleTasksForAuctionsMutex.RUnlock()
	return fake.scheduleTasksForAuctionsArgsForCall[i].arg1
}

var _ auctiontypes.AuctionRunner = new(FakeAuctionRunner)
