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
