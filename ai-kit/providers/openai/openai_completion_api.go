// Ported from: packages/openai/src/completion/openai-completion-api.ts
package openai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// --- Response types ---

// openaiCompletionResponse represents the non-streaming completion API response.
type openaiCompletionResponse struct {
	ID      *string                        `json:"id,omitempty"`
	Created *float64                       `json:"created,omitempty"`
	Model   *string                        `json:"model,omitempty"`
	Choices []openaiCompletionChoice       `json:"choices"`
	Usage   *OpenAICompletionUsage         `json:"usage,omitempty"`
}

// openaiCompletionChoice represents a choice in the completion response.
type openaiCompletionChoice struct {
	Text         string                      `json:"text"`
	FinishReason string                      `json:"finish_reason"`
	Logprobs     *openaiCompletionLogprobs   `json:"logprobs,omitempty"`
}

// openaiCompletionLogprobs holds log probability data for completions.
type openaiCompletionLogprobs struct {
	Tokens        []string              `json:"tokens"`
	TokenLogprobs []float64             `json:"token_logprobs"`
	TopLogprobs   []map[string]float64  `json:"top_logprobs,omitempty"`
}

// openaiCompletionResponseSchema is the schema for non-streaming completion responses.
var openaiCompletionResponseSchema = &providerutils.Schema[openaiCompletionResponse]{}

// --- Streaming chunk types ---

// openaiCompletionChunkSchema is the schema for streaming completion chunks.
// Uses map[string]any to handle the union of regular chunks and error chunks.
var openaiCompletionChunkSchema = &providerutils.Schema[any]{}
