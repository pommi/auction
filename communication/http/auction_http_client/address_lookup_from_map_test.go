package auction_http_client_test

import (
	. "github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddressLookupFromMap", func() {
	var lookup AddressLookup
	Describe("the returning AddressLookup function", func() {
		BeforeEach(func() {
			lookup = AddressLookupFromMap(map[string]string{
				"rep-guid-A": "http://a",
				"rep-guid-B": "http://b",
			})
		})

		It("returns the correct address when one is found", func() {
			Ω(lookup("rep-guid-A")).Should(Equal("http://a"))
			Ω(lookup("rep-guid-B")).Should(Equal("http://b"))
		})

		It("retuns an error when no address is found", func() {
			addr, err := lookup("nope")
			Ω(addr).Should(BeEmpty())
			Ω(err).Should(HaveOccurred())
		})
	})
})
