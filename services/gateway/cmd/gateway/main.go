package main

import (
	"context"
	"log"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/server"
)

var (
	CommitHash string
	Tag        string
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx, CommitHash, Tag)
	if err != nil || cfg == nil {
		log.Fatalf("failed to load application configuration: %s", err)
	}

	promReg := prometheus.NewRegistry()

	telemetry, err := observability.InitTracer(ctx, cfg.App.Name, cfg.App.Version, cfg.App.Environment, cfg.Telemetry, promReg)
	if err != nil {
		log.Fatalf("failed to initialize telemetry providers: %s", err)
	}

	lgr := observability.NewLogger(cfg.App.Name, cfg.App.Version, cfg.App.LogLevel, cfg.App.Environment)
	lgr.Info("starting llm-gateway service...")

	s := server.NewServer(
		*cfg,
		lgr,
		*telemetry,
		promReg,
	)

	if err := s.Run(ctx); err != nil {
		log.Fatalf("failed to run server: %s", err)
	}
}
