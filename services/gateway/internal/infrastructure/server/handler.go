package server

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/proxy"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/registry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *Server) RegisterRouteHandlers(m *chi.Mux, r *registry.Registry) error {
	v1 := chi.NewRouter()

	m.Handle("/metrics", promhttp.HandlerFor(r.Prometheus, promhttp.HandlerOpts{}))
	v1.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK,
			struct {
				Message string `json:"message"`
				Status  int    `json:"status"`
			}{
				Message: "healthy",
				Status:  http.StatusOK,
			})
	})

	// TODO: replace backend stub with proper implementation
	m.Handle("/v1/chat/completions", proxy.NewHandler("http://127.0.0.1:11434"))

	m.Mount("/api/v1", v1)

	return nil
}

func (s *Server) RegisterHealthHandlers(m *chi.Mux, ready func() bool) {
	// Liveness probe: returns 200 if the process is alive
	m.Get("/livez", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
	})

	// Startup probe: returns 200 if the app sucessfully started up
	m.Get("/startupz", func(w http.ResponseWriter, _ *http.Request) {
		if ready() {
			writeJSON(w, http.StatusOK, map[string]string{"status": "OK"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "Starting"})
	})

	// Readiness probe: returns 200 if the app is ready to serve traffic
	m.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ready() {
			writeJSON(w, http.StatusOK, map[string]string{"status": "Ready"})
			return
		}
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "NotReady"})
	})
}
