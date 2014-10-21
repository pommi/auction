package auction_http_client_test

import (
	"errors"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReleaseReservation", func() {
	var startAuctionInfo auctiontypes.StartAuctionInfo

	BeforeEach(func() {
		startAuctionInfo = auctiontypes.StartAuctionInfo{
			ProcessGuid:  "process-guid",
			InstanceGuid: "instance-guid",
			Index:        1,
			DiskMB:       1024,
			MemoryMB:     256,
		}

		auctionRepA.ReleaseReservationReturns(nil)
		auctionRepB.ReleaseReservationReturns(errors.New("oops"))
	})

	It("should tell all the reps to release the reservation", func() {
		client.ReleaseReservation(RepAddressesFor("A", "B", "RepThat500s", "RepThatErrors"), startAuctionInfo)

		Ω(auctionRepA.ReleaseReservationCallCount()).Should(Equal(1))
		Ω(auctionRepA.ReleaseReservationArgsForCall(0)).Should(Equal(startAuctionInfo))

		Ω(auctionRepB.ReleaseReservationCallCount()).Should(Equal(1))
		Ω(auctionRepB.ReleaseReservationArgsForCall(0)).Should(Equal(startAuctionInfo))
	})

	PIt("what about errors?", func() {

	})
})
