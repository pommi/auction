// This file was generated by counterfeiter
package fakes

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type FakeSimulationAuctionRep struct {
	GuidStub        func() string
	guidMutex       sync.RWMutex
	guidArgsForCall []struct{}
	guidReturns struct {
		result1 string
	}
	BidForStartAuctionStub        func(startAuctionInfo auctiontypes.StartAuctionInfo) (float64, error)
	bidForStartAuctionMutex       sync.RWMutex
	bidForStartAuctionArgsForCall []struct {
		startAuctionInfo auctiontypes.StartAuctionInfo
	}
	bidForStartAuctionReturns struct {
		result1 float64
		result2 error
	}
	BidForStopAuctionStub        func(stopAuctionInfo auctiontypes.StopAuctionInfo) (float64, []string, error)
	bidForStopAuctionMutex       sync.RWMutex
	bidForStopAuctionArgsForCall []struct {
		stopAuctionInfo auctiontypes.StopAuctionInfo
	}
	bidForStopAuctionReturns struct {
		result1 float64
		result2 []string
		result3 error
	}
	RebidThenTentativelyReserveStub        func(startAuctionInfo models.LRPStartAuction) (float64, error)
	rebidThenTentativelyReserveMutex       sync.RWMutex
	rebidThenTentativelyReserveArgsForCall []struct {
		startAuctionInfo models.LRPStartAuction
	}
	rebidThenTentativelyReserveReturns struct {
		result1 float64
		result2 error
	}
	ReleaseReservationStub        func(startAuctionInfo models.LRPStartAuction) error
	releaseReservationMutex       sync.RWMutex
	releaseReservationArgsForCall []struct {
		startAuctionInfo models.LRPStartAuction
	}
	releaseReservationReturns struct {
		result1 error
	}
	RunStub        func(startAuction models.LRPStartAuction) error
	runMutex       sync.RWMutex
	runArgsForCall []struct {
		startAuction models.LRPStartAuction
	}
	runReturns struct {
		result1 error
	}
	StopStub        func(stopInstance models.StopLRPInstance) error
	stopMutex       sync.RWMutex
	stopArgsForCall []struct {
		stopInstance models.StopLRPInstance
	}
	stopReturns struct {
		result1 error
	}
	ResetStub        func()
	resetMutex       sync.RWMutex
	resetArgsForCall []struct{}
	SetSimulatedInstancesStub        func(instances []auctiontypes.SimulatedInstance)
	setSimulatedInstancesMutex       sync.RWMutex
	setSimulatedInstancesArgsForCall []struct {
		instances []auctiontypes.SimulatedInstance
	}
	SimulatedInstancesStub        func() []auctiontypes.SimulatedInstance
	simulatedInstancesMutex       sync.RWMutex
	simulatedInstancesArgsForCall []struct{}
	simulatedInstancesReturns struct {
		result1 []auctiontypes.SimulatedInstance
	}
	TotalResourcesStub        func() auctiontypes.Resources
	totalResourcesMutex       sync.RWMutex
	totalResourcesArgsForCall []struct{}
	totalResourcesReturns struct {
		result1 auctiontypes.Resources
	}
}

func (fake *FakeSimulationAuctionRep) Guid() string {
	fake.guidMutex.Lock()
	fake.guidArgsForCall = append(fake.guidArgsForCall, struct{}{})
	fake.guidMutex.Unlock()
	if fake.GuidStub != nil {
		return fake.GuidStub()
	} else {
		return fake.guidReturns.result1
	}
}

func (fake *FakeSimulationAuctionRep) GuidCallCount() int {
	fake.guidMutex.RLock()
	defer fake.guidMutex.RUnlock()
	return len(fake.guidArgsForCall)
}

func (fake *FakeSimulationAuctionRep) GuidReturns(result1 string) {
	fake.GuidStub = nil
	fake.guidReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeSimulationAuctionRep) BidForStartAuction(startAuctionInfo auctiontypes.StartAuctionInfo) (float64, error) {
	fake.bidForStartAuctionMutex.Lock()
	fake.bidForStartAuctionArgsForCall = append(fake.bidForStartAuctionArgsForCall, struct {
		startAuctionInfo auctiontypes.StartAuctionInfo
	}{startAuctionInfo})
	fake.bidForStartAuctionMutex.Unlock()
	if fake.BidForStartAuctionStub != nil {
		return fake.BidForStartAuctionStub(startAuctionInfo)
	} else {
		return fake.bidForStartAuctionReturns.result1, fake.bidForStartAuctionReturns.result2
	}
}

func (fake *FakeSimulationAuctionRep) BidForStartAuctionCallCount() int {
	fake.bidForStartAuctionMutex.RLock()
	defer fake.bidForStartAuctionMutex.RUnlock()
	return len(fake.bidForStartAuctionArgsForCall)
}

func (fake *FakeSimulationAuctionRep) BidForStartAuctionArgsForCall(i int) auctiontypes.StartAuctionInfo {
	fake.bidForStartAuctionMutex.RLock()
	defer fake.bidForStartAuctionMutex.RUnlock()
	return fake.bidForStartAuctionArgsForCall[i].startAuctionInfo
}

func (fake *FakeSimulationAuctionRep) BidForStartAuctionReturns(result1 float64, result2 error) {
	fake.BidForStartAuctionStub = nil
	fake.bidForStartAuctionReturns = struct {
		result1 float64
		result2 error
	}{result1, result2}
}

func (fake *FakeSimulationAuctionRep) BidForStopAuction(stopAuctionInfo auctiontypes.StopAuctionInfo) (float64, []string, error) {
	fake.bidForStopAuctionMutex.Lock()
	fake.bidForStopAuctionArgsForCall = append(fake.bidForStopAuctionArgsForCall, struct {
		stopAuctionInfo auctiontypes.StopAuctionInfo
	}{stopAuctionInfo})
	fake.bidForStopAuctionMutex.Unlock()
	if fake.BidForStopAuctionStub != nil {
		return fake.BidForStopAuctionStub(stopAuctionInfo)
	} else {
		return fake.bidForStopAuctionReturns.result1, fake.bidForStopAuctionReturns.result2, fake.bidForStopAuctionReturns.result3
	}
}

func (fake *FakeSimulationAuctionRep) BidForStopAuctionCallCount() int {
	fake.bidForStopAuctionMutex.RLock()
	defer fake.bidForStopAuctionMutex.RUnlock()
	return len(fake.bidForStopAuctionArgsForCall)
}

func (fake *FakeSimulationAuctionRep) BidForStopAuctionArgsForCall(i int) auctiontypes.StopAuctionInfo {
	fake.bidForStopAuctionMutex.RLock()
	defer fake.bidForStopAuctionMutex.RUnlock()
	return fake.bidForStopAuctionArgsForCall[i].stopAuctionInfo
}

func (fake *FakeSimulationAuctionRep) BidForStopAuctionReturns(result1 float64, result2 []string, result3 error) {
	fake.BidForStopAuctionStub = nil
	fake.bidForStopAuctionReturns = struct {
		result1 float64
		result2 []string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeSimulationAuctionRep) RebidThenTentativelyReserve(startAuctionInfo models.LRPStartAuction) (float64, error) {
	fake.rebidThenTentativelyReserveMutex.Lock()
	fake.rebidThenTentativelyReserveArgsForCall = append(fake.rebidThenTentativelyReserveArgsForCall, struct {
		startAuctionInfo models.LRPStartAuction
	}{startAuctionInfo})
	fake.rebidThenTentativelyReserveMutex.Unlock()
	if fake.RebidThenTentativelyReserveStub != nil {
		return fake.RebidThenTentativelyReserveStub(startAuctionInfo)
	} else {
		return fake.rebidThenTentativelyReserveReturns.result1, fake.rebidThenTentativelyReserveReturns.result2
	}
}

func (fake *FakeSimulationAuctionRep) RebidThenTentativelyReserveCallCount() int {
	fake.rebidThenTentativelyReserveMutex.RLock()
	defer fake.rebidThenTentativelyReserveMutex.RUnlock()
	return len(fake.rebidThenTentativelyReserveArgsForCall)
}

func (fake *FakeSimulationAuctionRep) RebidThenTentativelyReserveArgsForCall(i int) models.LRPStartAuction {
	fake.rebidThenTentativelyReserveMutex.RLock()
	defer fake.rebidThenTentativelyReserveMutex.RUnlock()
	return fake.rebidThenTentativelyReserveArgsForCall[i].startAuctionInfo
}

func (fake *FakeSimulationAuctionRep) RebidThenTentativelyReserveReturns(result1 float64, result2 error) {
	fake.RebidThenTentativelyReserveStub = nil
	fake.rebidThenTentativelyReserveReturns = struct {
		result1 float64
		result2 error
	}{result1, result2}
}

func (fake *FakeSimulationAuctionRep) ReleaseReservation(startAuctionInfo models.LRPStartAuction) error {
	fake.releaseReservationMutex.Lock()
	fake.releaseReservationArgsForCall = append(fake.releaseReservationArgsForCall, struct {
		startAuctionInfo models.LRPStartAuction
	}{startAuctionInfo})
	fake.releaseReservationMutex.Unlock()
	if fake.ReleaseReservationStub != nil {
		return fake.ReleaseReservationStub(startAuctionInfo)
	} else {
		return fake.releaseReservationReturns.result1
	}
}

func (fake *FakeSimulationAuctionRep) ReleaseReservationCallCount() int {
	fake.releaseReservationMutex.RLock()
	defer fake.releaseReservationMutex.RUnlock()
	return len(fake.releaseReservationArgsForCall)
}

func (fake *FakeSimulationAuctionRep) ReleaseReservationArgsForCall(i int) models.LRPStartAuction {
	fake.releaseReservationMutex.RLock()
	defer fake.releaseReservationMutex.RUnlock()
	return fake.releaseReservationArgsForCall[i].startAuctionInfo
}

func (fake *FakeSimulationAuctionRep) ReleaseReservationReturns(result1 error) {
	fake.ReleaseReservationStub = nil
	fake.releaseReservationReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeSimulationAuctionRep) Run(startAuction models.LRPStartAuction) error {
	fake.runMutex.Lock()
	fake.runArgsForCall = append(fake.runArgsForCall, struct {
		startAuction models.LRPStartAuction
	}{startAuction})
	fake.runMutex.Unlock()
	if fake.RunStub != nil {
		return fake.RunStub(startAuction)
	} else {
		return fake.runReturns.result1
	}
}

func (fake *FakeSimulationAuctionRep) RunCallCount() int {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return len(fake.runArgsForCall)
}

func (fake *FakeSimulationAuctionRep) RunArgsForCall(i int) models.LRPStartAuction {
	fake.runMutex.RLock()
	defer fake.runMutex.RUnlock()
	return fake.runArgsForCall[i].startAuction
}

func (fake *FakeSimulationAuctionRep) RunReturns(result1 error) {
	fake.RunStub = nil
	fake.runReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeSimulationAuctionRep) Stop(stopInstance models.StopLRPInstance) error {
	fake.stopMutex.Lock()
	fake.stopArgsForCall = append(fake.stopArgsForCall, struct {
		stopInstance models.StopLRPInstance
	}{stopInstance})
	fake.stopMutex.Unlock()
	if fake.StopStub != nil {
		return fake.StopStub(stopInstance)
	} else {
		return fake.stopReturns.result1
	}
}

func (fake *FakeSimulationAuctionRep) StopCallCount() int {
	fake.stopMutex.RLock()
	defer fake.stopMutex.RUnlock()
	return len(fake.stopArgsForCall)
}

func (fake *FakeSimulationAuctionRep) StopArgsForCall(i int) models.StopLRPInstance {
	fake.stopMutex.RLock()
	defer fake.stopMutex.RUnlock()
	return fake.stopArgsForCall[i].stopInstance
}

func (fake *FakeSimulationAuctionRep) StopReturns(result1 error) {
	fake.StopStub = nil
	fake.stopReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeSimulationAuctionRep) Reset() {
	fake.resetMutex.Lock()
	fake.resetArgsForCall = append(fake.resetArgsForCall, struct{}{})
	fake.resetMutex.Unlock()
	if fake.ResetStub != nil {
		fake.ResetStub()
	}
}

func (fake *FakeSimulationAuctionRep) ResetCallCount() int {
	fake.resetMutex.RLock()
	defer fake.resetMutex.RUnlock()
	return len(fake.resetArgsForCall)
}

func (fake *FakeSimulationAuctionRep) SetSimulatedInstances(instances []auctiontypes.SimulatedInstance) {
	fake.setSimulatedInstancesMutex.Lock()
	fake.setSimulatedInstancesArgsForCall = append(fake.setSimulatedInstancesArgsForCall, struct {
		instances []auctiontypes.SimulatedInstance
	}{instances})
	fake.setSimulatedInstancesMutex.Unlock()
	if fake.SetSimulatedInstancesStub != nil {
		fake.SetSimulatedInstancesStub(instances)
	}
}

func (fake *FakeSimulationAuctionRep) SetSimulatedInstancesCallCount() int {
	fake.setSimulatedInstancesMutex.RLock()
	defer fake.setSimulatedInstancesMutex.RUnlock()
	return len(fake.setSimulatedInstancesArgsForCall)
}

func (fake *FakeSimulationAuctionRep) SetSimulatedInstancesArgsForCall(i int) []auctiontypes.SimulatedInstance {
	fake.setSimulatedInstancesMutex.RLock()
	defer fake.setSimulatedInstancesMutex.RUnlock()
	return fake.setSimulatedInstancesArgsForCall[i].instances
}

func (fake *FakeSimulationAuctionRep) SimulatedInstances() []auctiontypes.SimulatedInstance {
	fake.simulatedInstancesMutex.Lock()
	fake.simulatedInstancesArgsForCall = append(fake.simulatedInstancesArgsForCall, struct{}{})
	fake.simulatedInstancesMutex.Unlock()
	if fake.SimulatedInstancesStub != nil {
		return fake.SimulatedInstancesStub()
	} else {
		return fake.simulatedInstancesReturns.result1
	}
}

func (fake *FakeSimulationAuctionRep) SimulatedInstancesCallCount() int {
	fake.simulatedInstancesMutex.RLock()
	defer fake.simulatedInstancesMutex.RUnlock()
	return len(fake.simulatedInstancesArgsForCall)
}

func (fake *FakeSimulationAuctionRep) SimulatedInstancesReturns(result1 []auctiontypes.SimulatedInstance) {
	fake.SimulatedInstancesStub = nil
	fake.simulatedInstancesReturns = struct {
		result1 []auctiontypes.SimulatedInstance
	}{result1}
}

func (fake *FakeSimulationAuctionRep) TotalResources() auctiontypes.Resources {
	fake.totalResourcesMutex.Lock()
	fake.totalResourcesArgsForCall = append(fake.totalResourcesArgsForCall, struct{}{})
	fake.totalResourcesMutex.Unlock()
	if fake.TotalResourcesStub != nil {
		return fake.TotalResourcesStub()
	} else {
		return fake.totalResourcesReturns.result1
	}
}

func (fake *FakeSimulationAuctionRep) TotalResourcesCallCount() int {
	fake.totalResourcesMutex.RLock()
	defer fake.totalResourcesMutex.RUnlock()
	return len(fake.totalResourcesArgsForCall)
}

func (fake *FakeSimulationAuctionRep) TotalResourcesReturns(result1 auctiontypes.Resources) {
	fake.TotalResourcesStub = nil
	fake.totalResourcesReturns = struct {
		result1 auctiontypes.Resources
	}{result1}
}

var _ auctiontypes.SimulationAuctionRep = new(FakeSimulationAuctionRep)
