package registry

import (
	"fmt"
	"log/slog"
	"packages/lib/golang/shared/config"
	"packages/lib/golang/shared/observability"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/admission"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/authentication"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/circuitbreaker"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/connectionpool"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/loadbalancer"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/proxy"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/ratelimit"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/transport/http"
	"github.com/prometheus/client_golang/prometheus"
)

type Registry struct {
	cfg config.Config
	lgr *slog.Logger

	Prometheus *prometheus.Registry
	Meter      *observability.Metrics

	Authenticator  ports.Authenticator
	ConnectionPool ports.ConnectionPool
	ProxyRelay     ports.Proxy
	LoadBalancer   ports.LoadBalancer
	RateLimiter    ports.Limiter
	Admitter       ports.Admitter
	CircuitBreaker ports.CircuitBreaker
}

func Init(cfg config.Config, lgr *slog.Logger, prom *prometheus.Registry) (*Registry, error) {
	registry := &Registry{
		cfg: cfg,
		lgr: lgr,
	}

	registry.Prometheus = prom
	metric, err := observability.NewMetrics()
	if err != nil {
		return nil, fmt.Errorf("initialize prometheus meter: %w", err)
	}
	registry.Meter = metric

	registry.Authenticator = authentication.New()
	registry.ConnectionPool = connectionpool.New(cfg.ConnectionPool, lgr.With("component", "connection_pool"))
	registry.ProxyRelay = proxy.New(cfg.SSEStreaming, lgr.With("component", "proxy_relay"))

	if cfg.Circuit.Enabled {
		registry.CircuitBreaker = circuitbreaker.NewManager(cfg.Circuit, lgr.With("component", "circuit_breaker"), registry.Meter)
	} else {
		registry.CircuitBreaker = circuitbreaker.NewNoopManager()
	}

	registry.LoadBalancer = loadbalancer.New(cfg.LoadBalancer, lgr.With("component", "load_balancer"), registry.CircuitBreaker)
	registry.RateLimiter = ratelimit.NewLimiter(cfg.RateLimit)
	registry.Admitter = admission.NewController(cfg.Admission)

	return registry, nil
}

func (r *Registry) CreateHTTPHandler() transport.Handler {
	return http.NewHandler(
		r.cfg.Proxy,
		r.lgr.With("component", "transport_http"),
		r.Meter,
		func() bool { return true },
		r.ConnectionPool,
		r.ProxyRelay,
		r.LoadBalancer,
		r.Admitter,
		r.CircuitBreaker,
	)
}
