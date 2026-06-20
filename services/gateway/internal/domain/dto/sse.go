package dto

import "time"

type RelayResult struct {
	TTFTMS           float64
	Bytes            int64
	PromptTokens     int
	CompletionTokens int
	EndReason        string // "done", "eof", "client_gone", "idle_timeout", "error"
	Err              error
}

func (r RelayResult) GetTTFT() time.Duration {
	return time.Duration(r.TTFTMS * float64(time.Millisecond))
}
func (r RelayResult) GetBytes() int64          { return r.Bytes }
func (r RelayResult) GetPromptTokens() int     { return r.PromptTokens }
func (r RelayResult) GetCompletionTokens() int { return r.CompletionTokens }
func (r RelayResult) GetEndReason() string     { return r.EndReason }
func (r RelayResult) GetErr() error            { return r.Err }
