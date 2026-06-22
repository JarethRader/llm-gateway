package registry

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/pkg/tursoutil"

	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jarethrader/llm-gateway/api-service/internal/domain/backend"
	backendHTTP "github.com/jarethrader/llm-gateway/api-service/internal/domain/backend/delivery/http"
	backendRepository "github.com/jarethrader/llm-gateway/api-service/internal/domain/backend/repository"
	backendService "github.com/jarethrader/llm-gateway/api-service/internal/domain/backend/service"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Registry struct {
	cfg config.Config
	lgr *slog.Logger

	Prometheus *prometheus.Registry
	TursoDB    *sqlx.DB

	backendRepository backend.Repository
	backendUsecase    backend.Usecase
}

func Init(ctx context.Context, cfg config.Config, lgr *slog.Logger) (*Registry, error) {
	registry := &Registry{
		cfg: cfg,
		lgr: lgr,
	}

	registry.Prometheus = prometheus.NewRegistry()

	tursoDB, err := tursoutil.Connect(cfg.Turso.URL, cfg.Turso.Token)
	if err != nil {
		lgr.Error("connect to turso db", slog.Any("error", err))
		return nil, err
	}
	registry.TursoDB = tursoDB

	migrator := tursoutil.NewMigrator(tursoDB, "enrichment", lgr, migrationFS)
	if err := migrator.Migrate(ctx); err != nil {
		return nil, fmt.Errorf("apply database migrations: %w", err)
	}

	registry.backendRepository = backendRepository.NewRepository(registry.TursoDB, lgr.With("component", "backend_repository"))
	registry.backendUsecase = backendService.NewService(registry.backendRepository, lgr.With("component", "backend_service"))

	return registry, nil
}

func (r *Registry) CreateBackendHandler() backend.Handler {
	return backendHTTP.NewHandler(r.backendUsecase, r.lgr.With("component", "backend_http"))
}
