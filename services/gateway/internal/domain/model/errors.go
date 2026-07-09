package model

import "errors"

var (
	ErrEmptyBearer    = errors.New("empty bearer token")
	ErrUnknownKey     = errors.New("unknown key")
	ErrModelNotFound  = errors.New("model not found")
	ErrNoBackend      = errors.New("no eligible backend")
	ErrRateLimited    = errors.New("rate limited")
	ErrQueueFull      = errors.New("admission queue full")
	ErrCircuitOpen    = errors.New("circuit open")
	ErrBackendFailure = errors.New("backend failure")
	ErrClientGone     = errors.New("client disconnected")
)
