package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/registry"
	transportHTTP "github.com/jarethrader/llm-gateway/gateway-service/internal/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) RegisterRouteHandlers(m *chi.Mux, r *registry.Registry, mw MiddlewareHandlers) error {
	v1 := chi.NewRouter()

	m.Handle("/metrics", promhttp.HandlerFor(r.Prometheus, promhttp.HandlerOpts{}))
	v1.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(struct {
			Message string `json:"message"`
			Status  int    `json:"status"`
		}{
			Message: "healthy",
			Status:  http.StatusOK,
		})
	})
	transportHTTP.RegisterHealthRoutes(m, r.CreateHTTPHandler())

	m.Route("/", func(subrouter chi.Router) {
		subrouter.Use(mw.Auth(r.Authenticator))
		subrouter.Use(mw.StreamExtract)
		if s.cfg.RateLimit.Enabled {
			subrouter.Use(mw.RateLimit(r.RateLimiter))
		}

		transportHTTP.RegisterProxyRoutes(subrouter, r.CreateHTTPHandler())
	})

	m.Mount("/api/v1", v1)

	return nil
}
