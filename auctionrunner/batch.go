package auctionrunner

import (
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auctioneer"
	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/rep"

	"github.com/pivotal-golang/clock"
)

type Batch struct {
	containerAuctions auctiontypes.ContainerAuctions
	lrpAuctions       []auctiontypes.LRPAuction
	taskAuctions      []auctiontypes.TaskAuction
	lock              *sync.Mutex
	HasWork           chan struct{}
	clock             clock.Clock
}

func NewBatch(clock clock.Clock) *Batch {
	return &Batch{
		lrpAuctions: []auctiontypes.LRPAuction{},
		lock:        &sync.Mutex{},
		clock:       clock,
		HasWork:     make(chan struct{}, 1),
	}
}

func (b *Batch) AddContainerStarts(starts []auctioneer.ContainerStartRequest) {
	auctions := make([]auctiontypes.ContainerAuction, 0, len(starts))
	now := b.clock.Now()
	for i := range starts {
		start := &starts[i]
		for _, index := range start.Indices {
			containerKey := rep.NewContainerKey(start.ProcessGuid, start.Domain, int32(index))
			auction := auctiontypes.NewContainerAuction(rep.NewContainer(containerKey, start.Resource), now)
			auctions = append(auctions, auction)
		}
	}

	b.lock.Lock()
	b.containerAuctions = append(b.containerAuctions, auctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) AddLRPStarts(starts []auctioneer.LRPStartRequest) {
	auctions := make([]auctiontypes.LRPAuction, 0, len(starts))
	now := b.clock.Now()
	for i := range starts {
		start := &starts[i]
		for _, index := range start.Indices {
			lrpKey := models.NewActualLRPKey(start.ProcessGuid, int32(index), start.Domain)
			auction := auctiontypes.NewLRPAuction(rep.NewLRP(lrpKey, start.Resource), now)
			auctions = append(auctions, auction)
		}
	}

	b.lock.Lock()
	b.lrpAuctions = append(b.lrpAuctions, auctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) AddTasks(tasks []auctioneer.TaskStartRequest) {
	auctions := make([]auctiontypes.TaskAuction, 0, len(tasks))
	now := b.clock.Now()
	for i := range tasks {
		auctions = append(auctions, auctiontypes.NewTaskAuction(tasks[i].Task, now))
	}

	b.lock.Lock()
	b.taskAuctions = append(b.taskAuctions, auctions...)
	b.claimToHaveWork()
	b.lock.Unlock()
}

func (b *Batch) DedupeAndDrain() (auctiontypes.ContainerAuctions, []auctiontypes.LRPAuction, []auctiontypes.TaskAuction) {
	b.lock.Lock()
	containerAuctions := b.containerAuctions
	lrpAuctions := b.lrpAuctions
	taskAuctions := b.taskAuctions
	b.lrpAuctions = []auctiontypes.LRPAuction{}
	b.taskAuctions = []auctiontypes.TaskAuction{}
	select {
	case <-b.HasWork:
	default:
	}
	b.lock.Unlock()

	dedupedContainerAuctions := auctiontypes.NewContainerAuctions()
	presentContainerAuctions := map[string]bool{}
	for _, startAuction := range containerAuctions {
		id := startAuction.Identifier()
		if presentContainerAuctions[id] {
			continue
		}
		presentContainerAuctions[id] = true
		dedupedContainerAuctions = append(dedupedContainerAuctions, startAuction)
	}

	dedupedLRPAuctions := []auctiontypes.LRPAuction{}
	presentLRPAuctions := map[string]bool{}
	for _, startAuction := range lrpAuctions {
		id := startAuction.Identifier()
		if presentLRPAuctions[id] {
			continue
		}
		presentLRPAuctions[id] = true
		dedupedLRPAuctions = append(dedupedLRPAuctions, startAuction)
	}

	dedupedTaskAuctions := []auctiontypes.TaskAuction{}
	presentTaskAuctions := map[string]bool{}
	for _, taskAuction := range taskAuctions {
		id := taskAuction.Identifier()
		if presentTaskAuctions[id] {
			continue
		}
		presentTaskAuctions[id] = true
		dedupedTaskAuctions = append(dedupedTaskAuctions, taskAuction)
	}

	return dedupedContainerAuctions, dedupedLRPAuctions, dedupedTaskAuctions
}

func (b *Batch) claimToHaveWork() {
	select {
	case b.HasWork <- struct{}{}:
	default:
	}
}
