package dto

// PartialRequest is decoded from the body to extract policy-relevant fields
// WITHOUT fully parsing messages. The raw body is forwarded unchanged.
type PartialRequest struct {
	Model  string `json:"model"`
	Stream bool   `json:"stream"`
	// StreamOptions.IncludeUsage asks vLLM to emit a usage frame when streaming.
	StreamOptions *struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options,omitempty"`
	// MaxTokens is used (with a heuristic prompt estimate) for token weighting.
	MaxTokens int `json:"max_tokens"`
}

// ChatMessage / ChatCompletionRequest are provided for completeness
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletionChunk is one SSE frame's JSON in streaming mode.
type ChatCompletionChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role    string `json:"role,omitempty"`
			Content string `json:"content,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

// ModelsResponse is GET /v1/models
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelCard `json:"data"`
}

type ModelCard struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}
