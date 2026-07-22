package connectionpool

import (
	"log/slog"
	"packages/lib/golang/shared/config"
	"sync/atomic"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type Pool struct {
	cfg     config.ConnectionPool
	lgr     *slog.Logger
	clients atomic.Pointer[map[model.BackendID]model.BackendConnection]
}

func New(cfg config.ConnectionPool, lgr *slog.Logger) ports.ConnectionPool {
	pool := &Pool{
		cfg: cfg,
		lgr: lgr,
	}
	clients := make(map[model.BackendID]model.BackendConnection)
	pool.clients.Store(&clients)
	return pool
}

func (p *Pool) Client(id model.BackendID) (model.BackendConnection, bool) {
	m := *p.clients.Load()
	c, ok := m[id]
	return c, ok
}

// Sync reconciles the pool with the desired backend set.
func (p *Pool) Sync(desired []model.Backend) {
	old := *p.clients.Load()
	next := make(map[model.BackendID]model.BackendConnection, len(desired))
	for _, b := range desired {
		if c, ok := old[b.ID]; ok {
			next[b.ID] = c
			continue
		}
		next[b.ID] = newClient(b, p.cfg)
	}
	p.clients.Store(&next)

	for id, c := range old {
		if _, kept := next[id]; !kept {
			c.Connection.CloseIdleConnections()
		}
	}
}

func (p *Pool) IsModelAvailable(requestedModel model.LargeLanguageModelID) bool {
	availableModels := make(map[string]struct{}, 0)
	for _, backend := range *p.clients.Load() {
		for _, model := range backend.Model.Models {
			availableModels[string(model)] = struct{}{}
		}
	}
	_, ok := availableModels[string(requestedModel)]
	return ok
}
