package auction_http_handlers

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
)

type rebidThenReserve struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *rebidThenReserve) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("rebid-then-reserve")
	logger.Info("handling")

	var startAuctionInfo auctiontypes.StartAuctionInfo
	if !decodeJSON(w, r, &startAuctionInfo, logger) {
		return
	}

	logger = logger.WithData(lagerDataForStartAuctionInfo(startAuctionInfo))

	response := auctiontypes.StartAuctionBid{
		Rep: h.rep.Guid(),
	}

	var status int
	bid, err := h.rep.RebidThenTentativelyReserve(startAuctionInfo)
	if err != nil {
		status = http.StatusForbidden
		response.Error = err.Error()
		logger.Info("not-bidding", lager.Data{"reason": err.Error()})
	} else {
		status = http.StatusOK
		response.Bid = bid
		logger.Info("bidding", lager.Data{"bid": bid})
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)

	logger.Info("success")
}
