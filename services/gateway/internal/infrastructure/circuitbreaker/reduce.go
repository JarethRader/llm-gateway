package circuitbreaker

import (
	"packages/lib/golang/shared/config"
	"time"

	"github.com/jarethrader/llm-gateway/gateway-service/internal/domain/model"
)

// Event is and input to the FSM that can mutate the state
type Event int

const (
	EventSuccess Event = iota // a request succeeded
	EventFailed               // a request failed
	EventClock                // a time tick / allow-check with no outcome
)

// Inputs are the precomputed decision inputs the infrastructure
// derives from its atomic counters, so the reducer stays pure.
type Inputs struct {
	Total        int     // samples in the rolling window
	Failures     int     // failures in the rolling window
	FailureRatio float64 // Failure/Totals, 0 if Total==0
}

// Reduce computes the next state. It is a total function of (state, event,
// inputs, config). The infrastructure applies side effects (metrics, events)
// by diffing old vs new Phase.
func Reduce(st model.CircuitBreakerState, ev Event, in Inputs, cfg config.Circuit, now time.Time) model.CircuitBreakerState {
	switch st.Phase {
	case model.PhaseClosed:
		if ev == EventFailed {
			if in.Total >= cfg.MinRequests && in.FailureRatio >= cfg.FailureRatio {
				st.Phase = model.PhaseOpen
				st.OpenedAt = now
				st.Trips++
			}
		}
		return st

	case model.PhaseOpen:
		// Transition to HalfOpen only when the cooldown has elapsed.
		if now.Sub(st.OpenedAt) >= OpenTimeout(cfg, st.Trips) {
			st.Phase = model.PhaseHalfOpen
			st.HalfOpenSuccess = 0
			st.HalfOpenInFlight = 0
		}
		return st

	case model.PhaseHalfOpen:
		switch ev {
		case EventSuccess:
			st.HalfOpenSuccess++
			if st.HalfOpenSuccess >= cfg.HalfOpenSuccess {
				st.Phase = model.PhaseClosed
				st.Trips = 0
				st.HalfOpenSuccess = 0
				st.HalfOpenInFlight = 0
			}
		case EventFailed:
			// any probe failure re-opens with increased backoff
			st.Phase = model.PhaseOpen
			st.OpenedAt = now
			st.Trips++
			st.HalfOpenSuccess = 0
			st.HalfOpenInFlight = 0
		}
		return st
	}
	return st
}

func OpenTimeout(cfg config.Circuit, trips int32) time.Duration {
	d := float64(cfg.OpenBase)
	for i := 1; i < int(trips); i++ {
		d *= cfg.BackoffFactor
		if d >= float64(cfg.OpenMax) {
			return cfg.OpenMax
		}
	}
	if d > float64(cfg.OpenMax) {
		return cfg.OpenMax
	}
	return time.Duration(d)
}
