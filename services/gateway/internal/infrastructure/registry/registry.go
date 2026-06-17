package registry

import (
	"log/slog"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/transport/http"
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

func (r *Registry) CreateHTTPHandler() transport.Handler {
	return http.NewHandler(r.lgr.With("component", "transport_http"), func() bool { return true })
}
