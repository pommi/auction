package auction_http_client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/pivotal-golang/lager"
)

type AuctionHTTPClient struct {
	client        *http.Client
	logger        lager.Logger
	addressLookup AddressLookup
}

type AddressLookup func(string) (string, error)

func AddressLookupFromMap(lookupTable map[string]string) AddressLookup {
	return func(repGuid string) (string, error) {
		address, ok := lookupTable[repGuid]
		if !ok {
			return "", errors.New("couldn't find address for " + repGuid)
		}
		return address, nil
	}
}

type Response struct {
	Body []byte
}

func New(client *http.Client, logger lager.Logger, addressLookup AddressLookup) *AuctionHTTPClient {
	return &AuctionHTTPClient{
		client:        client,
		logger:        logger,
		addressLookup: addressLookup,
	}
}

func (c *AuctionHTTPClient) BidForStartAuction(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	logger := c.logger.Session("bid-for-start-auction", lagerDataForStartAuctionInfo(startAuctionInfo))
	logger = logger.WithData(lager.Data{
		"num-requests": len(repGuids),
	})
	logger.Info("requesting")

	body, _ := json.Marshal(startAuctionInfo)
	responses := c.batch(repGuids, "GET", "/bids/start_auction", body)

	startAuctionBids := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		startAuctionBid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response.Body, &startAuctionBid)
		if err != nil {
			logger.Error("failed-to-parse-response", err)
			continue
		}
		startAuctionBids = append(startAuctionBids, startAuctionBid)
	}

	logger = logger.WithData(lager.Data{
		"num-responses": len(startAuctionBids),
	})
	logger.Info("done")

	return startAuctionBids
}

func (c *AuctionHTTPClient) BidForStopAuction(repGuids []string, stopAuctionInfo auctiontypes.StopAuctionInfo) auctiontypes.StopAuctionBids {
	logger := c.logger.Session("bid-for-stop-auction", lager.Data{
		"process-guid": stopAuctionInfo.ProcessGuid,
		"index":        stopAuctionInfo.Index,
		"num-requests": len(repGuids),
	})
	logger.Info("requesting")

	body, _ := json.Marshal(stopAuctionInfo)
	responses := c.batch(repGuids, "GET", "/bids/stop_auction", body)

	stopAuctionBids := auctiontypes.StopAuctionBids{}
	for _, response := range responses {
		stopAuctionBid := auctiontypes.StopAuctionBid{}
		err := json.Unmarshal(response.Body, &stopAuctionBid)
		if err != nil {
			logger.Error("failed-to-parse-response", err)
			continue
		}
		stopAuctionBids = append(stopAuctionBids, stopAuctionBid)
	}

	logger = logger.WithData(lager.Data{
		"num-responses": len(stopAuctionBids),
	})
	logger.Info("done")

	return stopAuctionBids
}

func (c *AuctionHTTPClient) RebidThenTentativelyReserve(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) auctiontypes.StartAuctionBids {
	logger := c.logger.Session("rebid-then-tentatively-reserve", lagerDataForStartAuctionInfo(startAuctionInfo))
	logger = logger.WithData(lager.Data{
		"num-requests": len(repGuids),
	})
	logger.Info("requesting")

	body, _ := json.Marshal(startAuctionInfo)
	responses := c.batch(repGuids, "POST", "/reservations", body)

	startAuctionBids := auctiontypes.StartAuctionBids{}
	for _, response := range responses {
		startAuctionBid := auctiontypes.StartAuctionBid{}
		err := json.Unmarshal(response.Body, &startAuctionBid)
		if err != nil {
			logger.Error("failed-to-parse-response", err)
			continue
		}
		startAuctionBids = append(startAuctionBids, startAuctionBid)
	}

	logger = logger.WithData(lager.Data{
		"num-responses": len(startAuctionBids),
	})
	logger.Info("done")

	return startAuctionBids
}

func (c *AuctionHTTPClient) ReleaseReservation(repGuids []string, startAuctionInfo auctiontypes.StartAuctionInfo) {
	logger := c.logger.Session("release-reservation", lagerDataForStartAuctionInfo(startAuctionInfo))
	logger.Info("requesting")
	body, _ := json.Marshal(startAuctionInfo)
	c.batch(repGuids, "DELETE", "/reservations", body)
	logger.Info("done")
}

func (c *AuctionHTTPClient) Run(repGuid string, lrpStartAuction models.LRPStartAuction) {
	logger := c.logger.Session("run", lager.Data{
		"process-guid":  lrpStartAuction.DesiredLRP.ProcessGuid,
		"instance-guid": lrpStartAuction.InstanceGuid,
		"index":         lrpStartAuction.Index,
	})
	logger.Info("requesting")
	body, _ := json.Marshal(lrpStartAuction)
	c.batch([]string{repGuid}, "POST", "/run", body)
	logger.Info("done")
}

func (c *AuctionHTTPClient) Stop(repGuid string, stopInstance models.StopLRPInstance) {
	logger := c.logger.Session("stop", lager.Data{
		"process-guid":  stopInstance.ProcessGuid,
		"instance-guid": stopInstance.InstanceGuid,
		"index":         stopInstance.Index,
	})
	logger.Info("requesting")
	body, _ := json.Marshal(stopInstance)
	c.batch([]string{repGuid}, "POST", "/stop", body)
	logger.Info("done")
}

func (c *AuctionHTTPClient) TotalResources(repGuid string) auctiontypes.Resources {
	responses := c.batch([]string{repGuid}, "GET", "/sim/total_resources", nil)
	if len(responses) != 1 {
		return auctiontypes.Resources{}
	}
	resources := auctiontypes.Resources{}
	err := json.Unmarshal(responses[0].Body, &resources)
	if err != nil {
		return auctiontypes.Resources{}
	}
	return resources
}

func (c *AuctionHTTPClient) SimulatedInstances(repGuid string) []auctiontypes.SimulatedInstance {
	responses := c.batch([]string{repGuid}, "GET", "/sim/simulated_instances", nil)
	if len(responses) != 1 {
		return nil
	}
	instances := []auctiontypes.SimulatedInstance{}
	err := json.Unmarshal(responses[0].Body, &instances)
	if err != nil {
		return nil
	}
	return instances
}

func (c *AuctionHTTPClient) SetSimulatedInstances(repGuid string, instances []auctiontypes.SimulatedInstance) {
	body, _ := json.Marshal(instances)
	c.batch([]string{repGuid}, "POST", "/sim/simulated_instances", body)
}

func (c *AuctionHTTPClient) Reset(repGuid string) {
	c.batch([]string{repGuid}, "POST", "/sim/reset", nil)
}

func lagerDataForStartAuctionInfo(startAuctionInfo auctiontypes.StartAuctionInfo) lager.Data {
	return lager.Data{
		"process-guid":  startAuctionInfo.ProcessGuid,
		"instance-guid": startAuctionInfo.InstanceGuid,
		"disk-mb":       startAuctionInfo.DiskMB,
		"memory-mb":     startAuctionInfo.MemoryMB,
		"index":         startAuctionInfo.InstanceGuid,
	}
}

/// batch http requests

func (c *AuctionHTTPClient) batch(repGuids []string, method string, path string, body []byte) []Response {
	requests := []*http.Request{}
	for _, repGuid := range repGuids {
		reader := bytes.NewBuffer(body)
		url, err := c.addressLookup(repGuid)
		if err != nil {
			continue
		}
		request, err := http.NewRequest(method, url+path, reader)
		if err != nil {
			continue
		}
		requests = append(requests, request)
	}

	return c.performRequests(requests)
}

func (c *AuctionHTTPClient) performRequests(requests []*http.Request) []Response {
	if len(requests) == 0 {
		return []Response{}
	}

	responsesChan := make(chan Response, len(requests))
	wg := &sync.WaitGroup{}
	wg.Add(len(requests))

	for _, request := range requests {
		go func(request *http.Request) {
			defer wg.Done()
			resp, err := c.client.Do(request)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				return
			}

			responseBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			responsesChan <- Response{
				Body: responseBody,
			}
		}(request)
	}

	wg.Wait()
	close(responsesChan)
	responses := []Response{}
	for response := range responsesChan {
		responses = append(responses, response)
	}

	return responses
}
