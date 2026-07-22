package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"packages/lib/golang/shared/observability"
	"strconv"
	"strings"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const maxBodySize = 4 * 1024 * 1024 // 4MB

type MiddlewareHandlers interface {
	StreamExtract(next http.Handler) http.Handler
	Recovery(next http.Handler) http.Handler
	Auth(auth ports.Authenticator) func(next http.Handler) http.Handler
	RateLimit(limiter ports.Limiter) func(next http.Handler) http.Handler
	Admit(admitter ports.Admitter) func(next http.Handler) http.Handler
}

type middleware struct {
	lgr   *slog.Logger
	meter *observability.Metrics
}

func NewMiddleware(lgr *slog.Logger, meter *observability.Metrics) MiddlewareHandlers {
	return &middleware{
		lgr:   lgr,
		meter: meter,
	}
}

func (m *middleware) StreamExtract(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			if _, ok := err.(*http.MaxBytesError); ok {
				writeJSONError(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
			} else {
				writeJSONError(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			}
			return
		}

		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(bodyBytes), r.Body))

		ctx := context.WithValue(r.Context(), model.BodyKey, bodyBytes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *middleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				m.lgr.Error("internal server error", slog.Any("error", err))

				json, _ := json.Marshal(map[string]string{
					"error": "There was an internal error",
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write(json)
				if err != nil {
					m.lgr.Error("internal server error", slog.Any("error", err))
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (m *middleware) Auth(auth ports.Authenticator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			authPrefix, authKey, found := strings.Cut(authHeader, " ")
			const prefix = "Bearer"

			if !found || len(authPrefix) == 0 {
				w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_request\", error_description=\"Missing authorization header\"")
				writeJSONError(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			if !strings.EqualFold(authPrefix, prefix) {
				w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_request\", error_description=\"Unsupported authentication scheme\"")
				writeJSONError(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			identity, err := auth.Authenticate(r.Context(), authKey)
			if err != nil {
				m.lgr.WarnContext(r.Context(), "auth failed", slog.String("error", err.Error()))
				if errors.Is(err, model.ErrEmptyBearer) {
					w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_request\", error_description=\"Missing authorization header\"")
					writeJSONError(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				} else {
					w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\", error_description=\"The access token is invalid or expired\"")
					writeJSONError(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				}
				return
			}

			span := trace.SpanFromContext(r.Context())
			span.AddEvent("authenticate", trace.WithAttributes(
				attribute.String("key_id", string(identity.KeyID)),
				attribute.String("tier", string(identity.Tier)),
				attribute.String("result", "ok"),
			))

			ctx := context.WithValue(r.Context(), model.IdentityKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (m *middleware) RateLimit(limiter ports.Limiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity := r.Context().Value(model.IdentityKey).(model.Identity)

			bodyBytes := r.Context().Value(model.BodyKey).([]byte)
			var req struct {
				Model *string `json:"model"`
			}
			if err := json.Unmarshal(bodyBytes, &req); err != nil {
				writeJSONError(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload"})
				return
			}
			if req.Model == nil || *req.Model == "" {
				writeJSONError(w, http.StatusBadRequest, map[string]string{"error": "request does not select a model"})
				return
			}

			decision := limiter.Allow(identity.KeyID, model.LargeLanguageModelID(*req.Model), 1)

			metricAttrs := make([]attribute.KeyValue, 0, 2)
			metricAttrs = append(metricAttrs, attribute.String("scope", string(decision.Scope)))
			if decision.Allowed {
				metricAttrs = append(metricAttrs, attribute.String("decision", "allow"))
			} else {
				metricAttrs = append(metricAttrs, attribute.String("decision", "deny"))
			}
			m.meter.RatelimitDecisions.Add(r.Context(), 1, metric.WithAttributes(metricAttrs...))

			span := trace.SpanFromContext(r.Context())
			span.AddEvent("ratelimit", trace.WithAttributes(
				metricAttrs[0],
				metricAttrs[1],
				attribute.Float64("retry_after_seconds", decision.RetryAfter.Seconds()),
			))

			if !decision.Allowed {
				m.lgr.WarnContext(r.Context(), "request rate-limited", slog.Any("decision", decision))
				retryAfter := strconv.FormatFloat(math.Ceil(decision.RetryAfter.Seconds()), 'f', -1, 64)
				w.Header().Set("Retry-After", retryAfter)
				writeJSONError(w, http.StatusTooManyRequests, map[string]string{
					"error":       fmt.Sprintf("rate limit exceeded for %s scope", decision.Scope),
					"scope":       string(decision.Scope),
					"retry-after": fmt.Sprintf("%ss", retryAfter),
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (m *middleware) Admit(admitter ports.Admitter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startAt := time.Now()
			permit, decision := admitter.Acquire(r.Context())
			decisionAttr := attribute.String("decision", "allow")
			if !decision.Allowed {
				decisionAttr = attribute.String("decision", "deny")
			}
			m.meter.AdmissionDecisions.Add(r.Context(), 1, metric.WithAttributes(decisionAttr))
			depth := admitter.QueueDepth()
			m.meter.QueueDepth.Record(r.Context(), int64(depth))
			m.meter.AdmissionWait.Record(r.Context(), time.Since(startAt).Seconds())

			_, span := observability.Tracer().Start(r.Context(), "gateway.admit")
			span.SetAttributes(
				attribute.Int("queue_depth", depth),
				attribute.Int("in_flight", admitter.InFlight()),
				attribute.Float64("waited_seconds", time.Since(startAt).Seconds()),
				decisionAttr,
				attribute.Float64("retry_after_seconds", decision.RetryAfter.Seconds()),
			)
			span.End()

			if err := r.Context().Err(); err != nil {
				if permit != nil {
					permit.Release()
				}
				return
			}

			if !decision.Allowed {
				if r.Context().Err() != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(499 /*Client Closed Request*/)
					_ = json.NewEncoder(w).Encode(map[string]string{"error": "client closed request"})
					return
				}
				m.lgr.WarnContext(r.Context(), "request not admitted", slog.Any("decision", decision))
				retryAfter := strconv.FormatFloat(math.Ceil(decision.RetryAfter.Seconds()), 'f', -1, 64)
				w.Header().Set("Retry-After", retryAfter)
				writeJSONError(w, http.StatusTooManyRequests, map[string]string{
					"error":       "admission rejected, please retry your request again later",
					"retry-after": fmt.Sprintf("%ss", retryAfter),
				})
				return
			}

			m.meter.Inflight.Add(r.Context(), 1)

			defer func(ctx context.Context) {
				permit.Release()
				m.meter.Inflight.Add(ctx, -1)
			}(r.Context())

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSONError(w http.ResponseWriter, status int, message map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(message)
}
