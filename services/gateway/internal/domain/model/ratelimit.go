package model

import "time"

// Scope identifies which dimension a limit applies to.
type Scope string

// Decision is the outcome of a limit check.
type Decision struct {
	Allowed    bool
	Scope      Scope         // with scope denied (when not allowed)
	RetryAfter time.Duration // suggested wait when denied
}
