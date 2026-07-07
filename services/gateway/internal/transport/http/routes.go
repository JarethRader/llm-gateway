package http

import (
	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
)

func RegisterProxyRoutes(m chi.Router, h transport.Handler) {
	// Chat
	m.Post("/v1/chat/completions", h.HandleChatCompletion())
}

func RegisterHealthRoutes(m *chi.Mux, h transport.Handler) {
	// Health Checks
	// Liveness probe: returns 200 if the process is alive
	m.Get("/livez", h.Livez())
	// Startup probe: returns 200 if the app sucessfully started up
	m.Get("/startupz", h.Startupz())
	// Readiness probe: returns 200 if the app is ready to serve traffic
	m.Get("/readyz", h.Readyz())
}

func RegisterDebugRoutes(m chi.Router, h transport.Handler) {
	m.Get("/queue-depth", h.DebugQueueDepth())
}
