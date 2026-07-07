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
	"strconv"
	"strings"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
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
	lgr *slog.Logger
}

func NewMiddleware(lgr *slog.Logger) MiddlewareHandlers {
	return &middleware{
		lgr: lgr,
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
			permit, decision := admitter.Acquire(r.Context())
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

			defer permit.Release()
			next.ServeHTTP(w, r)
		})
	}
}

func writeJSONError(w http.ResponseWriter, status int, message map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(message)
}
