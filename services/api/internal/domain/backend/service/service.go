package service

import (
	"context"
	"log/slog"

	"github.com/jarethrader/llm-gateway/api-service/internal/domain/backend"
	"github.com/jarethrader/llm-gateway/api-service/internal/models"
)

type service struct {
	lgr        *slog.Logger
	repository backend.Repository
}

func NewService(repository backend.Repository, lgr *slog.Logger) backend.Usecase {
	return &service{
		repository: repository,
		lgr:        lgr,
	}
}

// CreateBackend implements [backend.Usecase].
func (s *service) CreateBackend(ctx context.Context, backend models.Backend) (int64, error) {
	id, err := s.repository.CreateBackend(ctx, backend)
	if err != nil {
		s.lgr.Error("failed to create backend", slog.Any("error", err))
		return 0, err
	}

	return id, nil
}

// GetBackendByID implements [backend.Usecase].
func (s *service) GetBackendByID(ctx context.Context, backendID int64) (*models.Backend, error) {
	backend, err := s.repository.GetBackendByID(ctx, backendID)
	if err != nil {
		s.lgr.Error("failed to get backend by id", slog.Any("error", err))
		return nil, err
	}

	return backend, nil
}

// DeleteBackend implements [backend.Usecase].
func (s *service) DeleteBackend(ctx context.Context, backendID int64) error {
	if err := s.repository.DeleteBackend(ctx, backendID); err != nil {
		s.lgr.Error("failed to delete backend", slog.Any("error", err))
		return err
	}

	return nil
}

// SparseListBackends implements [backend.Usecase].
func (s *service) SparseListBackends(ctx context.Context) ([]models.SparseBackend, error) {
	backends, err := s.repository.SparseListBackends(ctx)
	if err != nil {
		s.lgr.Error("failed to sparse details for backend", slog.Any("error", err))
		return []models.SparseBackend{}, err
	}

	return backends, nil
}

// Updatebackend implements [backend.Usecase].
func (s *service) Updatebackend(ctx context.Context, backendID int64, backend models.Backend) error {
	if err := s.repository.UpdateBackend(ctx, backendID, backend); err != nil {
		s.lgr.Error("failed to update backend", slog.Any("error", err))
		return err
	}

	return nil
}
