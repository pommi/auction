package auction_http_client_test

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/auctiontypes/fakes"
	. "github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	"github.com/cloudfoundry-incubator/auction/communication/http/routes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"

	"testing"
)

func TestAuctionHttpClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AuctionHttpClient Suite")
}

var auctionRepA, auctionRepB *fakes.FakeSimulationAuctionRep
var serverA, serverB *httptest.Server
var serverThat500s *ghttp.Server
var serverThatErrors *ghttp.Server
var client auctiontypes.SimulationRepPoolClient
var addressMap map[string]string

func addressLookup(repGuid string) (string, error) {
	address, ok := addressMap[repGuid]
	if !ok {
		return "", errors.New("unknown repguid")
	}

	return address, nil
}

var _ = BeforeEach(func() {
	logger := lagertest.NewTestLogger("auction_http_client")
	client = New(&http.Client{}, logger, addressLookup)

	auctionRepA = &fakes.FakeSimulationAuctionRep{}
	auctionRepA.GuidReturns("A")

	auctionRepB = &fakes.FakeSimulationAuctionRep{}
	auctionRepB.GuidReturns("B")

	//an auction http server backed by a fake auction rep
	handler, err := rata.NewRouter(routes.Routes, auction_http_handlers.New(auctionRepA, logger))
	Ω(err).ShouldNot(HaveOccurred())
	serverA = httptest.NewServer(handler)

	//another auction http server backed by a fake auction rep
	handler, err = rata.NewRouter(routes.Routes, auction_http_handlers.New(auctionRepB, logger))
	Ω(err).ShouldNot(HaveOccurred())
	serverB = httptest.NewServer(handler)

	//an auction http server that always 500s
	serverThat500s = ghttp.NewServer()
	serverThat500s.AllowUnhandledRequests = true
	serverThat500s.UnhandledRequestStatusCode = http.StatusInternalServerError

	//an auction http server that always errors (by disconnecting)
	serverThatErrors = ghttp.NewServer()
	erroringHandler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		serverThatErrors.CloseClientConnections()
	})
	//5 erroringHandlers should be more than enough: none of the individual tests should make more than 5 requests to this server
	serverThatErrors.AppendHandlers(erroringHandler, erroringHandler, erroringHandler, erroringHandler, erroringHandler)

	addressMap = map[string]string{
		"A":             serverA.URL,
		"B":             serverB.URL,
		"RepThat500s":   serverThat500s.URL(),
		"RepThatErrors": serverThatErrors.URL(),
	}
})

var _ = AfterEach(func() {
	serverA.Close()
	serverB.Close()
	serverThat500s.Close()
	serverThatErrors.Close()
})
