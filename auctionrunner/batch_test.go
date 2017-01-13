package auctionrunner_test

import (
	"time"

	"code.cloudfoundry.org/auction/auctionrunner"
	"code.cloudfoundry.org/auction/auctiontypes"
	"code.cloudfoundry.org/auctioneer"
	"code.cloudfoundry.org/clock/fakeclock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Batch", func() {
	var lrpStart auctioneer.LRPStartRequest
	var task auctioneer.TaskStartRequest
	var batch *auctionrunner.Batch
	var clock *fakeclock.FakeClock

	BeforeEach(func() {
		clock = fakeclock.NewFakeClock(time.Now())
		batch = auctionrunner.NewBatch(clock)
	})

	It("should start off empty", func() {
		Expect(batch.HasWork).NotTo(Receive())
		starts, tasks := batch.DedupeAndDrain()
		Expect(starts).To(BeEmpty())
		Expect(tasks).To(BeEmpty())
	})

	Describe("adding work", func() {
		Context("when adding start auctions", func() {
			BeforeEach(func() {
				lrpStart = BuildLRPStartRequest("pg-1", "domain", []int{1}, "linux", 10, 10, 10, []string{}, []string{})
				batch.AddLRPStarts([]auctioneer.LRPStartRequest{lrpStart})
			})

			It("makes the start auction available when drained", func() {
				lrpAuctions, _ := batch.DedupeAndDrain()
				Expect(lrpAuctions).To(ConsistOf(BuildLRPAuctions(lrpStart, clock.Now())))
			})

			It("should have work", func() {
				Expect(batch.HasWork).To(Receive())
			})
		})

		Context("when adding tasks", func() {
			BeforeEach(func() {
				task = BuildTaskStartRequest("tg-1", "domain", "linux", 10, 10, 10)
				batch.AddTasks([]auctioneer.TaskStartRequest{task})
			})

			It("makes the stop auction available when drained", func() {
				_, taskAuctions := batch.DedupeAndDrain()
				Expect(taskAuctions).To(ConsistOf(BuildTaskAuction(&task.Task, clock.Now())))
			})

			It("should have work", func() {
				Expect(batch.HasWork).To(Receive())
			})
		})
	})

	Describe("resubmitting work", func() {
		Context("resubmitting starts", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				lrpStartAuction1 := BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10)
				startAuction1 := BuildLRPAuctions(lrpStartAuction1, timeProvider.Now())

				lrpStartAuction2 := BuildLRPStartRequest("pg-2", []uint{1}, "lucid64", 10, 10)
				startAuction2 := BuildLRPAuctions(lrpStartAuction2, timeProvider.Now())

				batch.AddLRPStarts([]models.LRPStartRequest{lrpStartAuction1, lrpStartAuction2})

				lrpAuctions, _ := batch.DedupeAndDrain()
				Ω(lrpAuctions).Should(Equal(append(startAuction1, startAuction2...)))

				batch.AddLRPStarts([]models.LRPStartRequest{lrpStartAuction1, lrpStartAuction2})
				batch.ResubmitStartAuctions(startAuction2)

				lrpAuctions, _ = batch.DedupeAndDrain()
				Ω(lrpAuctions).Should(Equal(append(startAuction2, startAuction1...)))
			})

			It("should have work", func() {
				lrpStartAuction := BuildLRPStartRequest("pg-1", []uint{1}, "lucid64", 10, 10)
				startAuction := BuildLRPAuctions(lrpStartAuction, timeProvider.Now())
				batch.ResubmitStartAuctions(startAuction)

				Ω(batch.HasWork).Should(Receive())
			})
		})

		Context("resubmitting tasks", func() {
			It("adds the work, and ensures it has priority when deduping", func() {
				task1 := BuildTask("tg-1", "lucid64", 10, 10)
				taskAuction1 := BuildTaskAuction(task1, timeProvider.Now())

				task2 := BuildTask("tg-2", "lucid64", 10, 10)
				taskAuction2 := BuildTaskAuction(task2, timeProvider.Now())

				batch.AddTasks([]models.Task{task1, task2})

				_, taskAuctions := batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction1, taskAuction2}))

				batch.AddTasks([]models.Task{task1, task2})
				batch.ResubmitTaskAuctions([]auctiontypes.TaskAuction{taskAuction2})

				_, taskAuctions = batch.DedupeAndDrain()
				Ω(taskAuctions).Should(Equal([]auctiontypes.TaskAuction{taskAuction2, taskAuction1}))
			})

			It("should have work", func() {
				task := BuildTask("tg-1", "lucid64", 10, 10)
				taskAuction := BuildTaskAuction(task, timeProvider.Now())
				batch.ResubmitTaskAuctions([]auctiontypes.TaskAuction{taskAuction})

				Ω(batch.HasWork).Should(Receive())
			})
		})
	})

	Describe("DedupeAndDrain", func() {
		BeforeEach(func() {
			batch.AddLRPStarts([]auctioneer.LRPStartRequest{
				BuildLRPStartRequest("pg-1", "domain", []int{1}, "linux", 10, 10, 10, []string{"driver-1"}, []string{"tag-1"}),
				BuildLRPStartRequest("pg-1", "domain", []int{1}, "linux", 10, 10, 10, []string{"driver-1"}, []string{"tag-1"}),
				BuildLRPStartRequest("pg-2", "domain", []int{2}, "linux", 10, 10, 10, []string{"driver-2"}, []string{"tag-2"}),
			})

			batch.AddTasks([]auctioneer.TaskStartRequest{
				BuildTaskStartRequest("tg-1", "domain", "linux", 10, 10, 10),
				BuildTaskStartRequest("tg-1", "domain", "linux", 10, 10, 10),
				BuildTaskStartRequest("tg-2", "domain", "linux", 10, 10, 10)})
		})

		It("should dedupe any duplicate start auctions and stop auctions", func() {
			lrpAuctions, taskAuctions := batch.DedupeAndDrain()
			Expect(lrpAuctions).To(Equal([]auctiontypes.LRPAuction{
				BuildLRPAuction("pg-1", "domain", 1, "linux", 10, 10, 10, clock.Now(), []string{"driver-1"}, []string{"tag-1"}),
				BuildLRPAuction("pg-2", "domain", 2, "linux", 10, 10, 10, clock.Now(), []string{"driver-2"}, []string{"tag-2"}),
			}))

			Expect(taskAuctions).To(Equal([]auctiontypes.TaskAuction{
				BuildTaskAuction(
					BuildTask("tg-1", "domain", "linux", 10, 10, 10, []string{}, []string{}),
					clock.Now(),
				),
				BuildTaskAuction(
					BuildTask("tg-2", "domain", "linux", 10, 10, 10, []string{}, []string{}),
					clock.Now(),
				),
			}))
		})

		It("should clear out its cache, so a subsequent call shouldn't fetch anything", func() {
			batch.DedupeAndDrain()
			lrpAuctions, taskAuctions := batch.DedupeAndDrain()
			Expect(lrpAuctions).To(BeEmpty())
			Expect(taskAuctions).To(BeEmpty())
		})

		It("should no longer have work after draining", func() {
			batch.DedupeAndDrain()
			Expect(batch.HasWork).NotTo(Receive())
		})

		It("should not hang forever if the work channel was already drained", func() {
			Expect(batch.HasWork).To(Receive())
			batch.DedupeAndDrain()
			Expect(batch.HasWork).NotTo(Receive())
		})
	})
})
