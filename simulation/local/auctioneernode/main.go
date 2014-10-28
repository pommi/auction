package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"
	"github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_client"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/cloudfoundry/yagnats"

	"github.com/cloudfoundry-incubator/auction/auctionrunner"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/cf-lager"
)

var natsAddrs = flag.String("natsAddrs", "", "nats server addresses")
var timeout = flag.Duration("timeout", time.Second, "timeout for nats responses")
var httpAddr = flag.String("httpAddr", "", "http address to listen on")

var errorResponse = []byte("error")

func main() {
	flag.Parse()

	if *httpAddr == "" {
		panic("need http addr")
	}

	repNATSClient := connectToNATS()
	var repHTTPClient auctiontypes.RepPoolClient

	repHTTPClient = auction_http_client.New(&http.Client{
		Timeout: *timeout,
	}, cf_lager.New("auctioneer-http"))

	getCommunicationMode := func(r *http.Request) auctiontypes.RepPoolClient {
		var repClient auctiontypes.RepPoolClient
		if r.URL.Query().Get("mode") == "NATS" {
			repClient = repNATSClient
		} else {
			repClient = repHTTPClient
		}
		return repClient
	}

	lock := &sync.Mutex{}
	results := []auctiontypes.StartAuctionResult{}
	done := false

	t := time.Now()
	http.HandleFunc("/start-auctions", func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		results = []auctiontypes.StartAuctionResult{}
		done = false
		lock.Unlock()

		var auctionRequests []auctiontypes.StartAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequests)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		maxConcurrent, err := strconv.Atoi(r.URL.Query().Get("maxConcurrent"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		repClient := getCommunicationMode(r)

		go func() {
			workers := workpool.NewWorkPool(maxConcurrent)

			wg := &sync.WaitGroup{}
			wg.Add(len(auctionRequests))
			for _, auctionRequest := range auctionRequests {
				auctionRequest := auctionRequest
				workers.Submit(func() {
					auctionResult, _ := auctionrunner.New(repClient).RunLRPStartAuction(auctionRequest)
					auctionResult.Duration = time.Since(t)
					lock.Lock()
					results = append(results, auctionResult)
					lock.Unlock()
					wg.Done()
				})
			}

			wg.Wait()
			lock.Lock()
			done = true
			lock.Unlock()
			workers.Stop()
		}()
	})

	http.HandleFunc("/start-auctions-results", func(w http.ResponseWriter, r *http.Request) {
		var statusCode int
		lock.Lock()
		if done {
			statusCode = http.StatusCreated
		} else {
			statusCode = http.StatusOK
		}
		payload, _ := json.Marshal(results)
		results = []auctiontypes.StartAuctionResult{}
		lock.Unlock()
		w.WriteHeader(statusCode)
		w.Write(payload)
	})

	http.HandleFunc("/stop-auctions", func(w http.ResponseWriter, r *http.Request) {
		var auctionRequests []auctiontypes.StopAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequests)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		maxConcurrent, err := strconv.Atoi(r.URL.Query().Get("maxConcurrent"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		repClient := getCommunicationMode(r)
		workers := workpool.NewWorkPool(maxConcurrent)

		lock := &sync.Mutex{}
		wg := &sync.WaitGroup{}
		wg.Add(len(auctionRequests))
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		for _, auctionRequest := range auctionRequests {
			auctionRequest := auctionRequest
			workers.Submit(func() {
				auctionResult, _ := auctionrunner.New(repClient).RunLRPStopAuction(auctionRequest)
				lock.Lock()
				encoder.Encode(auctionResult)
				lock.Unlock()
				wg.Done()
			})
		}

		wg.Wait()
		workers.Stop()
	})

	fmt.Println("auctioneering")

	panic(http.ListenAndServe(*httpAddr, nil))
}

func connectToNATS() auctiontypes.RepPoolClient {
	if *natsAddrs != "" {
		natsMembers := []string{}
		for _, addr := range strings.Split(*natsAddrs, ",") {
			uri := url.URL{
				Scheme: "nats",
				Host:   addr,
			}
			natsMembers = append(natsMembers, uri.String())
		}
		client, err := yagnats.Connect(natsMembers)
		if err != nil {
			log.Fatalln("no nats:", err)
		}

		repClient, err := auction_nats_client.New(client, *timeout, cf_lager.New("auctioneer-nats"))
		if err != nil {
			log.Fatalln("no rep client:", err)
		}

		return repClient
	}

	return nil
}
