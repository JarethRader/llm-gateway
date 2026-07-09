package http

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"packages/lib/golang/shared/config"
	"slices"
	"strings"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
)

var IgnoreHeaders = []string{
	"Authorization",
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
	"Accept-Encoding",
}

type handlers struct {
	cfg   config.Proxy
	lgr   *slog.Logger
	ready func() bool
	pool  ports.ConnectionPool
	proxy ports.Proxy
	lb    ports.LoadBalancer
	ad    ports.Admitter
	cbr   ports.CircuitBreaker
}

func NewHandler(
	cfg config.Proxy,
	lgr *slog.Logger,
	ready func() bool,
	pool ports.ConnectionPool,
	proxy ports.Proxy,
	lb ports.LoadBalancer,
	ad ports.Admitter,
	cbr ports.CircuitBreaker,
) transport.Handler {
	return handlers{
		cfg:   cfg,
		lgr:   lgr,
		ready: ready,
		pool:  pool,
		proxy: proxy,
		lb:    lb,
		ad:    ad,
		cbr:   cbr,
	}
}

// HandleChatCompletion implements [transport.Handler].
func (h handlers) HandleChatCompletion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dispatchStart := time.Now()

		bodyBytes := ctx.Value(model.BodyKey).([]byte)

		var payload struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no model requested"})
			return
		}

		if connHeaders := r.Header.Get("Connection"); connHeaders != "" {
			for _, header := range strings.Split(connHeaders, ",") {
				IgnoreHeaders = append(IgnoreHeaders, http.CanonicalHeaderKey(strings.TrimSpace(header)))
			}
		}

		dispatches := 0
		var (
			backendID model.BackendID
			probe     bool
			tried     []model.BackendID
			resp      *http.Response
			gotResp   bool
		)
		for dispatches < max(h.cfg.MaxRetries, 1) {
			id, ok := h.lb.Select(model.LargeLanguageModelID(payload.Model), tried)
			if !ok {
				h.lgr.ErrorContext(ctx, "no eligible backends left to try")
				break // no eligible backends left to try
			}

			admit, isProbe := h.cbr.Allow(id, dispatchStart)
			if !admit {
				h.lgr.WarnContext(ctx, "breaker gated backend", slog.Any("backend", id))
				continue // breaker gated backend
			}

			backend, ok := h.pool.Client(id)
			if !ok {
				h.cbr.Release(id)
				continue
			}

			upstreamURL, err := url.JoinPath(backend.Model.BaseURL, "/v1/chat/completions")
			if err != nil {
				h.lgr.ErrorContext(ctx, "LLM server url is misconfigured", slog.Any("error", err))
				h.cbr.Release(id)
				continue
			}

			req, err := http.NewRequestWithContext(
				ctx,
				r.Method,
				upstreamURL,
				bytes.NewReader(bodyBytes))
			if err != nil {
				h.lgr.WarnContext(ctx, "failed to reconstruct request with context", slog.Any("backend", id))
				h.cbr.Release(id)
				continue
			}

			for k, v := range r.Header.Clone() {
				if slices.Contains(IgnoreHeaders, k) {
					continue
				}
				req.Header[k] = v
			}

			if !isProbe {
				h.lb.Inc(id)
			}
			dispatches++
			resp, err = backend.Connection.Do(req)
			if err != nil {
				h.lgr.ErrorContext(ctx, "failed to connect to upstream", slog.Any("error", err))
				h.cbr.RecordFailure(id)
				if !isProbe {
					h.lb.Dec(id)
				}
				tried = append(tried, id)
				continue
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				if h.cbr.IsFailure(resp.StatusCode) {
					h.lgr.ErrorContext(ctx, "received error from upstream", slog.Any("backend", id), slog.Int("status_code", resp.StatusCode))
					h.cbr.RecordFailure(id)
					if !isProbe {
						h.lb.Dec(id)
					}
					tried = append(tried, id)
					continue
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(body)
				h.cbr.Release(id)
				if !isProbe {
					h.lb.Dec(id)
				}
				return
			}

			backendID, probe, gotResp = id, isProbe, true
			break
		}

		if !gotResp {
			w.Header().Set("Retry-After", "120")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no available backend"})
			return
		}
		defer resp.Body.Close()

		flusher, ok := h.proxy.SetSSEHeaders(w)
		if !ok {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "streaming not supported"})
			h.cbr.Release(backendID)
			if !probe {
				h.lb.Dec(backendID)
			}
			return
		}

		result := h.proxy.RelaySSE(ctx, dispatchStart, resp.Body, w, flusher, config.SSEStreaming{
			IdleTimeout:       30 * time.Second,
			HeartbeatInterval: 15 * time.Second,
			FrameAware:        true,
			FlushEveryWrite:   true,
			MaxBodyBytes:      2048,
		})
		identity, _ := ctx.Value(model.IdentityKey).(model.Identity)
		h.lgr.With(slog.String("key_id", string(identity.KeyID)), slog.String("tier", string(identity.Tier))).DebugContext(r.Context(), "relay result", slog.String("end_reason", result.GetEndReason()))

		if result.TTFTMS > 0 && !probe {
			h.lb.Observe(backendID, result.TTFTMS)
		}

		if !probe {
			h.lb.Dec(backendID)
		}
		switch result.EndReason {
		case "done", "eof":
			h.cbr.RecordSuccess(backendID)
		case "client_gone":
			h.cbr.Release(backendID)
		default:
			h.cbr.RecordFailure(backendID)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
