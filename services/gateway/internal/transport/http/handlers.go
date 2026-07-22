package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"
	"strings"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
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
	meter *observability.Metrics
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
	meter *observability.Metrics,
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
		meter: meter,
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
		rootCtx := r.Context()
		dispatchStart := time.Now()
		outcome := new(string)
		modelLabel := "unknown"

		bodyBytes := rootCtx.Value(model.BodyKey).([]byte)

		var payload struct {
			Model  string `json:"model"`
			Stream *bool  `json:"stream"`
		}
		defer func(ctx context.Context) {
			modelAttr := attribute.String("model", modelLabel)
			streamAttr := attribute.Bool("stream", payload.Stream != nil && *payload.Stream)
			trace.SpanFromContext(ctx).SetAttributes(attribute.String("outcome", *outcome))
			h.meter.RequestsTotal.Add(ctx, 1, metric.WithAttributes(modelAttr, streamAttr, attribute.String("outcome", *outcome)))
			h.meter.RequestDuration.Record(ctx, time.Since(dispatchStart).Seconds(), metric.WithAttributes(modelAttr, attribute.String("outcome", *outcome)))
		}(rootCtx)

		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no model requested"})
			*outcome = "bad_request"
			return
		}

		if !h.pool.IsModelAvailable(model.LargeLanguageModelID(payload.Model)) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "requested model not available"})
			*outcome = "bad_request"
			return
		}
		modelLabel = payload.Model
		modelAttr := attribute.String("model", modelLabel)

		ignore := make(map[string]struct{}, len(IgnoreHeaders))
		for _, header := range IgnoreHeaders {
			ignore[http.CanonicalHeaderKey(header)] = struct{}{}
		}
		if connHeaders := r.Header.Get("Connection"); connHeaders != "" {
			for _, header := range strings.Split(connHeaders, ",") {
				ignore[http.CanonicalHeaderKey(strings.TrimSpace(header))] = struct{}{}
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
			ctx, span := observability.Tracer().Start(rootCtx, "gateway.route")
			id, ok := h.lb.Select(model.LargeLanguageModelID(modelLabel), tried)
			if !ok {
				h.lgr.ErrorContext(ctx, "no eligible backends left to try")
				span.End()
				break
			}
			h.meter.RouteSelected.Add(ctx, 1, metric.WithAttributes(attribute.String("backend", string(id)), modelAttr))
			span.SetAttributes(
				attribute.String("algo", "p2c"),
				attribute.String("model", modelLabel),
				attribute.String("selected_backend", string(id)),
			)

			admit, isProbe := h.cbr.Allow(id, dispatchStart)
			snap := h.cbr.Snapshot(id)
			trace.SpanFromContext(rootCtx).AddEvent("circuit", trace.WithAttributes(
				attribute.String("backend", string(id)),
				attribute.String("phase", snap.Phase.String()),
				attribute.Bool("admitted", admit),
				attribute.Bool("probe", isProbe),
			))

			if !admit {
				h.lgr.WarnContext(ctx, "breaker gated backend", slog.Any("backend", id))
				h.meter.CircuitShortCircuit.Add(ctx, 1, metric.WithAttributes(attribute.String("backend", string(id))))
				span.End()
				continue
			}

			backend, ok := h.pool.Client(id)
			if !ok {
				h.cbr.Release(id)
				span.End()
				continue
			}

			upstreamURL, err := url.JoinPath(backend.Model.BaseURL, "/v1/chat/completions")
			if err != nil {
				h.lgr.ErrorContext(ctx, "LLM server url is misconfigured", slog.Any("error", err))
				h.cbr.Release(id)
				span.End()
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
				span.End()
				continue
			}

			for k, v := range r.Header.Clone() {
				if _, skip := ignore[k]; skip {
					continue
				}
				req.Header[k] = v
			}

			if !isProbe {
				h.lb.Inc(id)
				h.meter.BackendInflight.Add(ctx, 1, metric.WithAttributes(attribute.String("backend", string(id))))
			}
			dispatches++
			resp, err = backend.Connection.Do(req)
			if err != nil {
				h.lgr.ErrorContext(ctx, "failed to connect to upstream", slog.Any("error", err))
				h.cbr.RecordFailure(id)
				if !isProbe {
					h.lb.Dec(id)
					h.meter.BackendInflight.Add(ctx, -1, metric.WithAttributes(attribute.String("backend", string(id))))
				}
				tried = append(tried, id)
				h.meter.RetriesTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", "connect_error")))
				trace.SpanFromContext(rootCtx).AddEvent("retry", trace.WithAttributes(
					attribute.Int("attempt", dispatches),
					attribute.String("reason", "connect_error"),
					attribute.String("backend", string(id)),
				))
				span.End()

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
						h.meter.BackendInflight.Add(ctx, -1, metric.WithAttributes(attribute.String("backend", string(id))))
					}
					tried = append(tried, id)
					h.meter.RetriesTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", "status_5xx")))
					trace.SpanFromContext(rootCtx).AddEvent("retry", trace.WithAttributes(
						attribute.Int("attempt", dispatches),
						attribute.String("reason", "status_5xx"),
						attribute.String("backend", string(id)),
					))
					span.End()

					continue
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(resp.StatusCode)
				_, _ = w.Write(body)
				h.cbr.Release(id)
				if !isProbe {
					h.lb.Dec(id)
					h.meter.BackendInflight.Add(ctx, -1, metric.WithAttributes(attribute.String("backend", string(id))))
				}
				*outcome = "upstream_error"
				span.End()
				return
			}

			backendID, probe, gotResp = id, isProbe, true
			span.End()
			break
		}
		backendAttr := attribute.String("backend", string(backendID))

		h.meter.RequestAttempts.Record(rootCtx, int64(dispatches), metric.WithAttributes(modelAttr))

		if !gotResp {
			h.meter.RouteNoBackend.Add(rootCtx, 1, metric.WithAttributes(modelAttr))

			w.Header().Set("Retry-After", "120")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "no available backend"})
			*outcome = "no_backend"
			return
		}
		defer resp.Body.Close()

		flusher, ok := h.proxy.SetSSEHeaders(w)
		if !ok {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "streaming not supported"})
			h.cbr.Release(backendID)
			if !probe {
				h.lb.Dec(backendID)
				h.meter.BackendInflight.Add(rootCtx, -1, metric.WithAttributes(backendAttr))
			}
			*outcome = "upstream_error"
			return
		}

		ctx, span := observability.Tracer().Start(rootCtx, "gateway.stream.relay")
		result := h.proxy.RelaySSE(ctx, dispatchStart, resp.Body, w, flusher, config.SSEStreaming{
			IdleTimeout:       30 * time.Second,
			HeartbeatInterval: 15 * time.Second,
			FrameAware:        true,
			FlushEveryWrite:   true,
			MaxBodyBytes:      2048,
		})
		span.SetAttributes(
			attribute.String("model", modelLabel),
			attribute.String("backend", string(backendID)),
			attribute.Int64("bytes", result.GetBytes()),
			attribute.Float64("ttft_ms", result.TTFTMS),
			attribute.Int("completion_tokens", result.GetCompletionTokens()),
			attribute.String("end_reason", result.GetEndReason()),
			attribute.Bool("frame_aware", true),
		)
		span.End()
		identity, _ := ctx.Value(model.IdentityKey).(model.Identity)
		h.lgr.With(slog.String("key_id", string(identity.KeyID)), slog.String("tier", string(identity.Tier))).DebugContext(r.Context(), "relay result", slog.String("end_reason", result.GetEndReason()))

		h.meter.TTFT.Record(ctx, result.GetTTFT().Seconds(), metric.WithAttributes(modelAttr, backendAttr))
		h.meter.TokensTotal.Add(ctx, int64(result.GetCompletionTokens()), metric.WithAttributes(modelAttr, backendAttr, attribute.String("kind", "completion")))
		h.meter.TokensTotal.Add(ctx, int64(result.GetPromptTokens()), metric.WithAttributes(modelAttr, backendAttr, attribute.String("kind", "prompt")))
		h.meter.StreamBytes.Add(ctx, result.GetBytes(), metric.WithAttributes(backendAttr, attribute.String("dir", "upstream")))

		if result.Err != nil {
			h.meter.StreamErrors.Add(ctx, 1, metric.WithAttributes(backendAttr, attribute.String("phase", result.EndReason)))
		}
		if result.EndReason == "client_gone" {
			h.meter.ClientDisconnect.Add(ctx, 1, metric.WithAttributes(attribute.String("phase", "client_gone")))
		}

		if result.TTFTMS > 0 && !probe {
			h.lb.Observe(backendID, result.TTFTMS)
		}

		if !probe {
			h.lb.Dec(backendID)
			h.meter.BackendInflight.Add(ctx, -1, metric.WithAttributes(backendAttr))
		}
		switch result.EndReason {
		case "done", "eof":
			h.cbr.RecordSuccess(backendID)
		case "client_gone":
			h.cbr.Release(backendID)
		default:
			h.cbr.RecordFailure(backendID)
		}
		*outcome = classifyOutcome(result.EndReason)
	}
}

func classifyOutcome(endReason string) string {
	switch endReason {
	case "done", "eof":
		return "ok"
	case "client_gone":
		return "client_cancelled"
	case "idle_timeout":
		return "timeout"
	case "error":
		return "upstream_error"
	default:
		return endReason
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
