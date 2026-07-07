package http

import (
	"net/http"
)

// Debug QueueDepth
func (h handlers) DebugQueueDepth() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		queueDepth := h.ad.QueueDepth()
		inFlight := h.ad.InFlight()
		writeJSON(w, http.StatusOK, map[string]int{
			"queue_depth": queueDepth,
			"in_flight":   inFlight,
		})
	}
}
