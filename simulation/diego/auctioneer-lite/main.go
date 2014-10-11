package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider"

	"github.com/cloudfoundry/storeadapter/workerpool"

	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"

	"github.com/cloudfoundry-incubator/runtime-schema/bbs"

	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/cf-lager"
)

var timeout = flag.Duration("timeout", time.Second, "timeout for nats responses")
var etcdCluster = flag.String("etcdCluster", "", "etcd cluster")

var lookupTable map[string]string
var lookupTableLock *sync.RWMutex

func FetchLookupTable() {
	store := etcdstoreadapter.NewETCDStoreAdapter(strings.Split(*etcdCluster, ","), workerpool.NewWorkerPool(10))
	store.Connect()
	BBS := bbs.NewBBS(store, timeprovider.NewTimeProvider(), cf_lager.New("auctioneer-bbs"))

	actuals, err := BBS.GetAllActualLRPs()
	if err != nil {
		log.Fatalln("failed to fetch reps from etcd", err)
	}

	lookupTableLock.Lock()
	lookupTable = map[string]string{}
	for _, actual := range actuals {
		if strings.HasPrefix(actual.ProcessGuid, "rep-lite") && len(actual.Ports) == 1 {
			lookupTable[actual.ProcessGuid] = fmt.Sprintf("http://%s:%d", actual.Host, actual.Ports[0].HostPort)
		}
	}
	lookupTableLock.Unlock()
}

func AddressLookup(repGuid string) (string, error) {
	lookupTableLock.RLock()
	defer lookupTableLock.RUnlock()

	if lookupTable == nil {
		return "", errors.New("lookupTable uninitialized")
	}

	address, ok := lookupTable[repGuid]
	if !ok {
		return "", errors.New("unkown rep-guid: " + repGuid)
	}

	return address, nil
}

func main() {
	flag.Parse()
	lookupTableLock = &sync.RWMutex{}

	if *etcdCluster == "" {
		log.Fatalln("you must provide an etcd cluster")
	}

	FetchLookupTable()

	var repClient auctiontypes.RepPoolClient
	repClient = auction_http_client.New(&http.Client{
		Timeout: *timeout,
	}, cf_lager.New("auctioneer-http"), AddressLookup)

	http.HandleFunc("/start-auction", func(w http.ResponseWriter, r *http.Request) {
		var auctionRequest auctiontypes.StartAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStartAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	http.HandleFunc("/stop-auction", func(w http.ResponseWriter, r *http.Request) {
		var auctionRequest auctiontypes.StopAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStopAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
	})

	http.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		FetchLookupTable()
		lookupTableLock.RLock()
		defer lookupTableLock.RUnlock()
		json.NewEncoder(w).Encode(lookupTable)
	})

	fmt.Println("auctioneering")

	panic(http.ListenAndServe("0.0.0.0:8080", nil))
}
