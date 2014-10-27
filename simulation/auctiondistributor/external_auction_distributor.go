package auctiondistributor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/cheggaaa/pb"
	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
)

type externalAuctionDistributor struct {
	hosts                    []string
	auctionCommunicationMode string
}

func NewExternalAuctionDistributor(hosts []string, auctionCommunicationMode string) AuctionDistributor {
	return &externalAuctionDistributor{
		auctionCommunicationMode: auctionCommunicationMode,
		hosts: hosts,
	}
}

func (d *externalAuctionDistributor) HoldStartAuctions(numAuctioneers int, startAuctions []models.LRPStartAuction, repAddresses []auctiontypes.RepAddress, rules auctiontypes.StartAuctionRules) []auctiontypes.StartAuctionResult {
	startAuctionRequests := buildStartAuctionRequests(startAuctions, repAddresses, rules)
	groupedRequests := map[int][]auctiontypes.StartAuctionRequest{}
	i := 0
	for _, request := range startAuctionRequests {
		index := i % numAuctioneers
		groupedRequests[index] = append(groupedRequests[index], request)
		i++
	}

	bar := pb.StartNew(len(startAuctions))

	results := []auctiontypes.StartAuctionResult{}
	lock := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(len(groupedRequests))
	for i := range groupedRequests {
		go func(i int) {
			defer wg.Done()
			payload, _ := json.Marshal(groupedRequests[i])
			res, err := http.Post("http://"+d.hosts[i]+"/start-auctions?mode="+d.auctionCommunicationMode, "application/json", bytes.NewReader(payload))
			if err != nil {
				fmt.Println("Failed to run auctions on index", i, err.Error())
				return
			}
			defer res.Body.Close()
			decoder := json.NewDecoder(res.Body)
			for {
				result := auctiontypes.StartAuctionResult{}
				err = decoder.Decode(&result)
				if err != nil {
					break
				}
				lock.Lock()
				results = append(results, result)
				bar.Set(len(results))
				lock.Unlock()
			}
		}(i)
	}

	wg.Wait()
	bar.Finish()
	return results
}

func (d *externalAuctionDistributor) HoldStopAuctions(numAuctioneers int, stopAuctions []models.LRPStopAuction, repAddresses []auctiontypes.RepAddress) []auctiontypes.StopAuctionResult {
	stopAuctionRequests := buildStopAuctionRequests(stopAuctions, repAddresses)

	groupedRequests := map[int][]auctiontypes.StopAuctionRequest{}
	i := 0
	for _, request := range stopAuctionRequests {
		index := i % numAuctioneers
		groupedRequests[index] = append(groupedRequests[index], request)
		i++
	}

	results := []auctiontypes.StopAuctionResult{}
	lock := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for i := 0; i < numAuctioneers; i++ {
		if len(groupedRequests[i]) == 0 {
			continue
		}
		go func(i int) {
			defer wg.Done()
			payload, _ := json.Marshal(groupedRequests[i])
			res, err := http.Post("http://"+d.hosts[i]+"/stop-auctions?mode="+d.auctionCommunicationMode, "application/json", bytes.NewReader(payload))
			if err != nil {
				fmt.Println("Failed to run auctions on index", i, err.Error())
				return
			}
			defer res.Body.Close()
			decoder := json.NewDecoder(res.Body)
			for {
				result := auctiontypes.StopAuctionResult{}
				err = decoder.Decode(&result)
				if err != nil {
					break
				}
				lock.Lock()
				results = append(results, result)
				lock.Unlock()
			}
		}(i)
	}

	wg.Wait()
	return results
}
