package model

import "time"

// Phase defines the circuit breaker phases
type BreakerPhase int64

const (
	PhaseClosed BreakerPhase = iota
	PhaseOpen
	PhaseHalfOpen
)

func (p BreakerPhase) String() string {
	switch p {
	case PhaseClosed:
		return "closed"
	case PhaseOpen:
		return "open"
	case PhaseHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

func (p BreakerPhase) Value() int64 {
	return int64(p)
}

// State is the full, serializable state of one breaker
type CircuitBreakerState struct {
	Phase            BreakerPhase
	OpenedAt         time.Time // time when it was last opened
	Trips            int32     // consecutive trips, drives backoff
	HalfOpenSuccess  int32     // successful probes so far in HalfOpen
	HalfOpenInFlight int32     // probes currently in flight
	IsOpen           bool
}
