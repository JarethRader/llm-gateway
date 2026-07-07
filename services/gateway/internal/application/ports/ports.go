package ports

import (
	"context"
	"io"
	"net/http"
	"packages/lib/golang/shared/config"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/dto"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type ConnectionPool interface {
	Client(id model.BackendID) (model.BackendConnection, bool)
	Sync(desired []model.Backend)
}

type Proxy interface {
	RelaySSE(ctx context.Context, dispatchStart time.Time, src io.ReadCloser, w http.ResponseWriter, flusher http.Flusher, cfg config.SSEStreaming) dto.RelayResult
	SetSSEHeaders(w http.ResponseWriter) (http.Flusher, bool)
}

type LoadBalancer interface {
	Select(model model.LargeLanguageModelID) (model.BackendID, bool)
	Observe(b model.BackendID, ttftMS float64)
	Inc(b model.BackendID)
	Dec(b model.BackendID)
	Sync(desired []model.Backend)
}

type Authenticator interface {
	Authenticate(ctx context.Context, bearer string) (model.Identity, error)
	Sync(keys []model.Identity)
}

type Limiter interface {
	Allow(key model.KeyID, model model.LargeLanguageModelID, tokens int) model.Decision
	Run(ctx context.Context)
}

// Permit must be released exactly once when the request completes.
type Permit interface {
	Release()
}
type Admitter interface {
	Acquire(ctx context.Context) (Permit, model.Decision)
	QueueDepth() int
	InFlight() int
}
