package model

import "errors"

var (
	ErrUnknownKey     = errors.New("unknown api key")
	ErrModelNotFound  = errors.New("model not found")
	ErrNoBackend      = errors.New("no eligible backend")
	ErrRateLimited    = errors.New("rate limited")
	ErrQueueFull      = errors.New("admission queue full")
	ErrCircuitOpen    = errors.New("circuit open")
	ErrBackendFailure = errors.New("backend failure")
	ErrClientGone     = errors.New("client disconnected")
)
