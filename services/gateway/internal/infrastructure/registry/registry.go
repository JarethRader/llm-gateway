package registry

import (
	"log/slog"
	"packages/lib/golang/shared/config"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/connectionpool"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/proxy"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/transport/http"
)

type Registry struct {
	cfg config.Config
	lgr *slog.Logger

	Prometheus *prometheus.Registry

	ConnectionPool ports.ConnectionPool
	ProxyRelay     ports.Proxy
}

func Init(cfg config.Config, lgr *slog.Logger) (*Registry, error) {
	registry := &Registry{
		cfg: cfg,
		lgr: lgr,
	}

	registry.Prometheus = prometheus.NewRegistry()

	registry.ConnectionPool = connectionpool.New(cfg.ConnectionPool, lgr.With("component", "connection_pool"))
	registry.ProxyRelay = proxy.New(cfg.SSEStreaming, lgr.With("component", "proxy_relay"))

	return registry, nil
}

func (r *Registry) CreateHTTPHandler() transport.Handler {
	return http.NewHandler(r.lgr.With("component", "transport_http"), func() bool { return true }, r.ConnectionPool, r.ProxyRelay)
}
