package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/auction/communication/http/auction_http_client"
	"github.com/cloudfoundry-incubator/auction/communication/nats/auction_nats_client"
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

	http.HandleFunc("/start-auction", func(w http.ResponseWriter, r *http.Request) {
		var auctionRequest auctiontypes.StartAuctionRequest
		err := json.NewDecoder(r.Body).Decode(&auctionRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		repClient := repHTTPClient
		if r.URL.Query().Get("mode") == "NATS" {
			repClient = repNATSClient
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

		repClient := repHTTPClient
		if r.URL.Query().Get("mode") == "NATS" {
			repClient = repNATSClient
		}

		auctionResult, _ := auctionrunner.New(repClient).RunLRPStopAuction(auctionRequest)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(auctionResult)
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
