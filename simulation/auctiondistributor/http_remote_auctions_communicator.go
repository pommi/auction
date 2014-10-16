package auctiondistributor

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync/atomic"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
)

type httpRemoteAuctions struct {
	hosts   []string
	mode    string
	counter uint64
}

func newHttpRemoteAuctions(hosts []string, mode string) *httpRemoteAuctions {
	return &httpRemoteAuctions{
		hosts:   hosts,
		mode:    mode,
		counter: 0,
	}
}

func (h *httpRemoteAuctions) RemoteStartAuction(auctionRequest auctiontypes.StartAuctionRequest) (auctiontypes.StartAuctionResult, error) {
	index := atomic.AddUint64(&(h.counter), 1) - 1
	host := h.hosts[int(index)%len(h.hosts)]

	payload, _ := json.Marshal(auctionRequest)
	res, err := http.Post("http://"+host+"/start-auction?mode="+h.mode, "application/json", bytes.NewReader(payload))
	if err != nil {
		return auctiontypes.StartAuctionResult{}, err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return auctiontypes.StartAuctionResult{}, err
	}

	var result auctiontypes.StartAuctionResult
	json.Unmarshal(data, &result)

	return result, nil
}

func (h *httpRemoteAuctions) RemoteStopAuction(auctionRequest auctiontypes.StopAuctionRequest) (auctiontypes.StopAuctionResult, error) {
	index := atomic.AddUint64(&(h.counter), 1) - 1
	host := h.hosts[int(index)%len(h.hosts)]

	payload, _ := json.Marshal(auctionRequest)
	res, err := http.Post("http://"+host+"/stop-auction?mode="+h.mode, "application/json", bytes.NewReader(payload))
	if err != nil {
		return auctiontypes.StopAuctionResult{}, err
	}

	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return auctiontypes.StopAuctionResult{}, err
	}

	var result auctiontypes.StopAuctionResult
	json.Unmarshal(data, &result)

	return result, nil
}
