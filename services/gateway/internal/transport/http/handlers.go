package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/transport"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/infrastructure/proxy"
)

type handlers struct {
	lgr   *slog.Logger
	ready func() bool
}

func NewHandler(lgr *slog.Logger, ready func() bool) transport.Handler {
	return handlers{
		lgr:   lgr,
		ready: ready,
	}
}

// HandleChatCompletion implements [transport.Handler].
func (h handlers) HandleChatCompletion() http.HandlerFunc {
	return proxy.NewHandler("http://localhost:11434")
	// return func(w http.ResponseWriter, r *http.Request) {
	// 	panic("unimplemented")
	// }
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
