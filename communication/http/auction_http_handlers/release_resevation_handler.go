package auction_http_handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/auction/auctiontypes"
	"github.com/pivotal-golang/lager"
)

type releaseReservation struct {
	rep    auctiontypes.AuctionRep
	logger lager.Logger
}

func (h *releaseReservation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := h.logger.Session("release-reservation")
	logger.Info("handling")

	var startAuctionInfo auctiontypes.StartAuctionInfo
	if !decodeJSON(w, r, &startAuctionInfo, logger) {
		return
	}

	logger = logger.WithData(lagerDataForStartAuctionInfo(startAuctionInfo))

	err := h.rep.ReleaseReservation(startAuctionInfo)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(err.Error()))
		logger.Error("failed", err)
		return
	}

	w.WriteHeader(http.StatusOK)

	logger.Info("success")
}
