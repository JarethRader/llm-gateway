package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type MiddlewareHandlers interface {
	Recovery(next http.Handler) http.Handler
	Auth(next http.Handler) http.Handler
}

type middleware struct {
	auth ports.Authenticator
	lgr  *slog.Logger
}

func NewMiddleware(auth ports.Authenticator, lgr *slog.Logger) MiddlewareHandlers {
	return &middleware{
		lgr:  lgr,
		auth: auth,
	}
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

func (m *middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		authPrefix, authKey, found := strings.Cut(authHeader, " ")
		const prefix = "Bearer"

		if !found || len(authPrefix) == 0 {
			writeUnauthorized(w, "Bearer error=\"invalid_request\", error_description=\"Missing authorization header\"")
			return
		}
		if !strings.EqualFold(authPrefix, prefix) {
			writeUnauthorized(w, "Bearer error=\"invalid_request\", error_description=\"Unsupported authentication scheme\"")
			return
		}

		bearer := ""
		if len(authHeader) >= len(prefix) && strings.EqualFold(authPrefix, prefix) {
			bearer = authKey
		}

		identity, err := m.auth.Authenticate(r.Context(), bearer)
		if err != nil {
			m.lgr.WarnContext(r.Context(), "auth failed", slog.String("error", err.Error()))
			if errors.Is(err, model.ErrEmptyBearer) {
				writeUnauthorized(w, "Bearer error=\"invalid_request\", error_description=\"Missing authorization header\"")
			} else {
				writeUnauthorized(w, "Bearer error=\"invalid_token\", error_description=\"The access token is invalid or expired\"")
			}
			return
		}

		ctx := context.WithValue(r.Context(), model.IdentityKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var exemptRoutes = []string{
	"/livez",
	"/readyz",
	"/startupz",
	"/metrics",
	"/api/v1/health",
}

func isExempt(path string) bool {
	for _, p := range exemptRoutes {
		if path == p {
			return true
		}
	}
	return false
}

func writeUnauthorized(w http.ResponseWriter, challenge string) {
	w.Header().Set("WWW-Authenticate", challenge)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
