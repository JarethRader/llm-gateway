package backend

import (
	"context"

	"github.com/jarethrader/llm-gateway/api-service/internal/models"
)

type Usecase interface {
	CreateBackend(ctx context.Context, backend models.Backend) (int64, error)
	Updatebackend(ctx context.Context, backendID int64, backend models.Backend) error
	DeleteBackend(ctx context.Context, backendID int64) error

	SparseListBackends(ctx context.Context) ([]models.SparseBackend, error)
	GetBackendByID(ctx context.Context, backendID int64) (*models.Backend, error)
}
