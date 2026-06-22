package http

import (
	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/api-service/internal/domain/backend"
)

func RegisterRoutes(m *chi.Mux, h backend.Handler) {
	m.Post("/", h.CreateBackend())
	m.Put("/{backendID}", h.UpdateBackend())
	m.Delete("/{backendID}", h.DeleteBackend())

	m.Get("/", h.ListBackends())
	m.Get("/{backendID}", h.GetBackend())
}
