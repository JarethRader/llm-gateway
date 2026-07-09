package circuitbreaker

import (
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/application/ports"
	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

type noopbreaker struct{}

func NewNoopManager() ports.CircuitBreaker {
	return &noopbreaker{}
}

// Allow implements [ports.CircuitBreaker].
func (n *noopbreaker) Allow(_ model.BackendID, _ time.Time) (bool, bool) {
	return true, false
}

// IsFailure implements [ports.CircuitBreaker].
func (n *noopbreaker) IsFailure(_ int) bool {
	return false
}

// IsProbeReady implements [ports.CircuitBreaker].
func (n *noopbreaker) IsProbeReady(_ model.BackendID) bool {
	return true
}

// RecordFailure implements [ports.CircuitBreaker].
func (n *noopbreaker) RecordFailure(_ model.BackendID) {
	/* noop */
}

// RecordSuccess implements [ports.CircuitBreaker].
func (n *noopbreaker) RecordSuccess(_ model.BackendID) {
	/* noop */
}

// Release implements [ports.CircuitBreaker].
func (n *noopbreaker) Release(_ model.BackendID) {
	/* noop */
}

// Snapshot implements [ports.CircuitBreaker].
func (n *noopbreaker) Snapshot(_ model.BackendID) model.CircuitBreakerState {
	return model.CircuitBreakerState{
		Phase: model.PhaseClosed,
	}
}

// Sync implements [ports.CircuitBreaker].
func (n *noopbreaker) Sync(_ []model.Backend) {
	/* noop */
}
