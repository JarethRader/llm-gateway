package registry

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
)

type Registry struct {
	lgr *slog.Logger

	Prometheus *prometheus.Registry
}

func Init(lgr *slog.Logger) (*Registry, error) {
	registry := &Registry{
		lgr: lgr,
	}

	registry.Prometheus = prometheus.NewRegistry()

	return registry, nil
}
