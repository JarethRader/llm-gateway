package http

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/jarethrader/llm-gateway/api-service/internal/domain/backend"
	"github.com/jarethrader/llm-gateway/api-service/internal/lib"
	"github.com/jarethrader/llm-gateway/api-service/internal/lib/validator"
	"github.com/jarethrader/llm-gateway/api-service/internal/models"
)

type handler struct {
	usecase backend.Usecase
	lgr     *slog.Logger
}

func NewHandler(usecase backend.Usecase, lgr *slog.Logger) backend.Handler {
	return &handler{
		usecase: usecase,
		lgr:     lgr,
	}
}

// CreateBackend implements [backend.Handler].
func (h *handler) CreateBackend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.Backend
		if err := validator.ParseRequest(r, &req); err != nil {
			h.lgr.Error("received invalid request", slog.String("path", "/backend"), slog.String("method", "POST"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusBadRequest, map[string]any{"errors": err})
			return
		}

		if err := h.usecase.CreateBackend(r.Context(), req); err != nil {
			h.lgr.Error("failed to complete request", slog.String("path", "/backend"), slog.String("method", "POST"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		lib.WriteJSON(w, http.StatusCreated, nil)
	}
}

// DeleteBackend implements [backend.Handler].
func (h *handler) DeleteBackend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "backendID")
		backendID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.lgr.Error("invalid backend id", slog.String("path", "/backend/{backendID}"), slog.String("method", "DELETE"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		if err = h.usecase.DeleteBackend(r.Context(), backendID); err != nil {
			h.lgr.Error("failed to complete request", slog.String("path", "/backend/{backendID}"), slog.String("method", "DELETE"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		lib.WriteJSON(w, http.StatusNoContent, nil)
	}
}

// GetBackend implements [backend.Handler].
func (h *handler) GetBackend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "backendID")
		backendID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.lgr.Error("invalid backend id", slog.String("path", "/backend/{backendID}"), slog.String("method", "GET"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		backend, err := h.usecase.GetBackendByID(r.Context(), backendID)
		if err != nil {
			h.lgr.Error("failed to complete request", slog.String("path", "/backend/{backendID}"), slog.String("method", "GET"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		lib.WriteJSON(w, http.StatusOK, map[string]any{"backend": backend})
	}
}

// ListBackends implements [backend.Handler].
func (h *handler) ListBackends() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		backends, err := h.usecase.SparseListBackends(r.Context())
		if err != nil {
			h.lgr.Error("failed to complete request", slog.String("path", "/backend"), slog.String("method", "GET"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		lib.WriteJSON(w, http.StatusOK, map[string]any{"backends": backends})
	}
}

// UpdateBackend implements [backend.Handler].
func (h *handler) UpdateBackend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "backendID")
		backendID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			h.lgr.Error("invalid backend id", slog.String("path", "/backend/{backendID}"), slog.String("method", "GET"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		var req models.Backend
		if err := validator.ParseRequest(r, &req); err != nil {
			lib.WriteJSON(w, http.StatusBadRequest, map[string]any{"errors": err})
			return
		}

		if err := h.usecase.Updatebackend(r.Context(), backendID, req); err != nil {
			h.lgr.Error("failed to complete request", slog.String("path", "/backend"), slog.String("method", "POST"), slog.Any("error", err))
			lib.WriteJSON(w, http.StatusInternalServerError, map[string]any{"errors": err})
			return
		}

		lib.WriteJSON(w, http.StatusCreated, nil)
	}
}
