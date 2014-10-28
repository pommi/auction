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
	"github.com/cloudfoundry/gunk/workpool"
)

type externalAuctionDistributor struct {
	hosts                    []string
	auctionCommunicationMode string
	maxConcurrent            int
}

func NewExternalAuctionDistributor(hosts []string, maxConcurrent int, auctionCommunicationMode string) AuctionDistributor {
	return &externalAuctionDistributor{
		auctionCommunicationMode: auctionCommunicationMode,
		maxConcurrent:            maxConcurrent,
		hosts:                    hosts,
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

	workPool := workpool.NewWorkPool(50)

	wg := &sync.WaitGroup{}
	wg.Add(len(groupedRequests))
	for i := range groupedRequests {
		i := i
		workPool.Submit(func() {
			defer wg.Done()
			payload, _ := json.Marshal(groupedRequests[i])
			url := fmt.Sprintf("http://%s/start-auctions?mode=%s&maxConcurrent=%d", d.hosts[i], d.auctionCommunicationMode, d.maxConcurrent)
			_, err := http.Post(url, "application/json", bytes.NewReader(payload))
			if err != nil {
				fmt.Println("Failed to run auctions on index", i, err.Error())
				return
			}
		})
	}

	wg.Wait()
	workPool.Stop()

	results := []auctiontypes.StartAuctionResult{}
	for {
		stillGoing := false
		for i := range groupedRequests {
			res, err := http.Get("http://" + d.hosts[i] + "/start-auctions-results")
			if err != nil {
				fmt.Println("Failed to get auctions on index", i, err.Error())
			}
			result := []auctiontypes.StartAuctionResult{}
			err = json.NewDecoder(res.Body).Decode(&result)
			if err != nil {
				res.Body.Close()
				fmt.Println("Failed to decode results", err.Error())
				continue
			}

			stillGoing = stillGoing || (res.StatusCode == http.StatusOK)
			results = append(results, result...)
			bar.Set(len(results))
			res.Body.Close()
		}
		if !stillGoing {
			break
		}
	}

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
			url := fmt.Sprintf("http://%s/stop-auctions?mode=%s&maxConcurrent=%d", d.hosts[i], d.auctionCommunicationMode, d.maxConcurrent)

			res, err := http.Post(url, "application/json", bytes.NewReader(payload))
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
