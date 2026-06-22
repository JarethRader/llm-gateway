package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	backendHTTP "github.com/jarethrader/llm-gateway/api-service/internal/domain/backend/delivery/http"
	"github.com/jarethrader/llm-gateway/api-service/internal/infrastructure/registry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) RegisterHandlers(m *chi.Mux, r *registry.Registry) error {
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

	backend := chi.NewRouter()
	backendHTTP.RegisterRoutes(backend, r.CreateBackendHandler())
	v1.Mount("/backend", backend)

	m.Mount("/api/v1", v1)

	return nil
}
