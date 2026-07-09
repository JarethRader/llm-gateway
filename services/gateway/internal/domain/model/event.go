package model

import "time"

// Event is a domain event emitted for observability/audit.
type Event interface{ occuredAt() time.Time }

type CircuitOpened struct {
	Backend BackendID
	At      time.Time
	Trips   int
}

func (e CircuitOpened) occuredAt() time.Time { return e.At }

type CircuitClosed struct {
	Backend BackendID
	At      time.Time
}

func (e CircuitClosed) occuredAt() time.Time { return e.At }

type BackendEjected struct {
	Backend BackendID
	At      time.Time
	Reason  string
}

func (e BackendEjected) occuredAt() time.Time { return e.At }
