package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/cloudfoundry-incubator/auction/communication/http/routes"

	"github.com/tedsuo/rata"

	"github.com/cloudfoundry-incubator/auction/auctionrep"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_handlers"
	"github.com/cloudfoundry-incubator/auction/simulation/simulationrepdelegate"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var memoryMB = flag.Int("memoryMB", 100, "total available memory in MB")
var diskMB = flag.Int("diskMB", 100, "total available disk in MB")
var containers = flag.Int("containers", 100, "total available containers")
var repGuid = flag.String("repGuid", "", "rep-guid")

func main() {
	flag.Parse()

	if *repGuid == "" {
		panic("need rep-guid")
	}

	repDelegate := simulationrepdelegate.New(auctiontypes.Resources{
		MemoryMB:   *memoryMB,
		DiskMB:     *diskMB,
		Containers: *containers,
	})
	rep := auctionrep.New(*repGuid, repDelegate)

	handlers := auction_http_handlers.New(rep, cf_lager.New("rep-lite").Session(*repGuid))
	router, err := rata.NewRouter(routes.Routes, handlers)
	if err != nil {
		log.Fatalln("failed to make router:", err)
	}
	httpServer := http_server.New("0.0.0.0:8080", router)

	monitor := ifrit.Envoke(sigmon.New(httpServer))

	fmt.Println("rep node listening")
	err = <-monitor.Wait()
	if err != nil {
		println("EXITED WITH ERROR: ", err.Error())
	}
}
