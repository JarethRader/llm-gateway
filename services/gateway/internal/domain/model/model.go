package model

import (
	"net/http"
	"time"
)

// Protocol selects the wire protocol used to reach a backend.
type Protocol string

const (
	ProtocolH1  Protocol = "h1"  // HTTP/1.1 (vLLM/uvicorn default)
	ProtocolH2  Protocol = "h2"  // HTTP/2 over TLS (ALPN)
	ProtocolH2C Protocol = "h2c" // HTTP/2 cleartext (required h2c server)
)

// Model is a logical model served by one or more backends
type Model struct {
	ID           LargeLanguageModelID
	MaxContext   int      // max prompt+completion tokens advertised
	Capabilities []string // e.g. "chat", "tools", "version"; used by capability filter
}

// Backend is one OpenAI compatible LLM serving instance. Identity is stable; liveness/load are tracked
// separately as mutable BackendStat in the routing layer
type Backend struct {
	ID       BackendID
	BaseURL  string // e.g. "http://vllm-0.vllm.svc:8000"
	Protocol Protocol
	Models   []LargeLanguageModelID // models this backend serves
	Weight   float64                // static capacity hint (>0), default 1.0
}

func (b Backend) Serves(m LargeLanguageModelID) bool {
	for _, id := range b.Models {
		if id == m {
			return true
		}
	}
	return false
}

// Request carries onlywhat the gateway needs to make policy decisions.
// The opaque request body is held in the transport/application layer and
// never in the domain, so the domain is decoupled from the OpenAI schema.
type Request struct {
	ID              RequestID
	Model           LargeLanguageModelID
	Stream          bool
	EstimatedTokens int // estimated prompt tokens for admission/limit reservation
	Identity        Identity
	ReceivedAt      time.Time
}

type RequestID string

// BackendConnection
type BackendConnection struct {
	Model      Backend
	Connection *http.Client
}
