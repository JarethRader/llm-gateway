package main

import (
	"context"
	"log"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"

	"github.com/jarethrader/llm-gateway/api-service/internal/infrastructure/server"
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

	telemetry, err := observability.InitTracer(ctx, cfg.App.Name, cfg.App.Version, cfg.App.Environment, cfg.Telemetry)
	if err != nil {
		log.Fatalf("failed to initialize telemetry providers: %s", err)
	}

	lgr := observability.NewLogger(cfg.App.Name, cfg.App.Version, cfg.App.LogLevel, cfg.App.Environment)
	lgr.Info("starting api service...")

	s := server.NewServer(*cfg, lgr, *telemetry)

	if err := s.Run(ctx); err != nil {
		log.Fatalf("server could not be started: %v", err)
	}
}
