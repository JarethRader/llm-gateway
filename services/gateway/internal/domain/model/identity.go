package model

// ModelID is the LLM model name a client requests, e.g. "llama-3.1-70b".
type ModelID string

// BackendID uniquely identifies one vLLM/OpenAI compatible backend instance.
type BackendID string

// KeyID is a stable, opaque identifier for an API key. It is NOT the key value itself.
type KeyID string

// Tier groups API keys for shared limits/labels (e.g. "free", "internal").
type Tier string

type Identity struct {
	KeyID KeyID
	Tier  Tier
}

func (i Identity) Valid() bool {
	return i.KeyID != ""
}
