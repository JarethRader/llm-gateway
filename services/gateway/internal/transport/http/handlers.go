package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"packages/lib/golang/shared/config"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
)

type handlers struct {
	lgr   *slog.Logger
	ready func() bool
	pool  ports.ConnectionPool
	proxy ports.Proxy
	lb    ports.LoadBalancer
}

func NewHandler(
	lgr *slog.Logger,
	ready func() bool,
	pool ports.ConnectionPool,
	proxy ports.Proxy,
	lb ports.LoadBalancer,
) transport.Handler {
	return handlers{
		lgr:   lgr,
		ready: ready,
		pool:  pool,
		proxy: proxy,
		lb:    lb,
	}
}

// HandleChatCompletion implements [transport.Handler].
func (h handlers) HandleChatCompletion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dispatchStart := time.Now()

		bodyBytes, err := io.ReadAll(r.Body)

		var payload struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no model requested"})
			return
		}

		backendID, ok := h.lb.Select(r.Context(), model.LargeLanguageModelID(payload.Model))
		if !ok {
			w.Header().Set("Retry-After", "120")
			writeJSON(w, http.StatusServiceUnavailable, nil)
			return
		}

		backend, ok := h.pool.Client(backendID)
		if !ok {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("model %s not found", payload.Model)})
			return
		}

		r.Body.Close()
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		req, err := http.NewRequestWithContext(
			r.Context(),
			r.Method,
			fmt.Sprintf("%s/v1/chat/completions", backend.Model.BaseURL),
			bytes.NewReader(bodyBytes))
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}

		for k, v := range r.Header {
			req.Header[k] = v
		}

		h.lb.Inc(backendID)
		defer h.lb.Dec(backendID)
		resp, err := backend.Connection.Do(req)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			payload, _ := io.ReadAll(resp.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			w.Write(payload)
			return
		}

		flusher, ok := h.proxy.SetSSEHeaders(w)
		if !ok {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "streaming not supported"})
			return
		}

		result := h.proxy.RelaySSE(r.Context(), dispatchStart, resp.Body, w, flusher, config.SSEStreaming{
			IdleTimeout:       30 * time.Second,
			HeartbeatInterval: 15 * time.Second,
			FrameAware:        true,
			FlushEveryWrite:   true,
			MaxBodyBytes:      2048,
		})
		h.lgr.Debug("relay result", slog.String("end_reason", result.GetEndReason()))

		if result.TTFTMS > 0 {
			h.lb.Observe(backendID, result.TTFTMS)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
