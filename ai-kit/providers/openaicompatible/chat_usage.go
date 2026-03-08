// Ported from: packages/openai-compatible/src/chat/convert-openai-compatible-chat-usage.ts
package openaicompatible

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// OpenAICompatibleTokenUsage represents the token usage from an OpenAI-compatible API response.
type OpenAICompatibleTokenUsage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	TotalTokens      *int `json:"total_tokens,omitempty"`

	PromptTokensDetails *struct {
		CachedTokens *int `json:"cached_tokens,omitempty"`
	} `json:"prompt_tokens_details,omitempty"`

	CompletionTokensDetails *struct {
		ReasoningTokens          *int `json:"reasoning_tokens,omitempty"`
		AcceptedPredictionTokens *int `json:"accepted_prediction_tokens,omitempty"`
		RejectedPredictionTokens *int `json:"rejected_prediction_tokens,omitempty"`
	} `json:"completion_tokens_details,omitempty"`
}

// ConvertChatUsage converts an OpenAI-compatible API usage response into
// the standard languagemodel.Usage type.
func ConvertChatUsage(usage *OpenAICompatibleTokenUsage) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{
			InputTokens:  languagemodel.InputTokenUsage{},
			OutputTokens: languagemodel.OutputTokenUsage{},
			Raw:          nil,
		}
	}

	promptTokens := 0
	if usage.PromptTokens != nil {
		promptTokens = *usage.PromptTokens
	}

	completionTokens := 0
	if usage.CompletionTokens != nil {
		completionTokens = *usage.CompletionTokens
	}

	cacheReadTokens := 0
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens != nil {
		cacheReadTokens = *usage.PromptTokensDetails.CachedTokens
	}

	reasoningTokens := 0
	if usage.CompletionTokensDetails != nil && usage.CompletionTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.CompletionTokensDetails.ReasoningTokens
	}

	noCache := promptTokens - cacheReadTokens
	textTokens := completionTokens - reasoningTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:    &promptTokens,
			NoCache:  &noCache,
			CacheRead: &cacheReadTokens,
			// CacheWrite: not available from OpenAI-compatible APIs
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &completionTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: structToJSONObject(usage),
	}
}

// structToJSONObject converts a struct to a jsonvalue.JSONObject by marshaling
// through JSON. Returns nil on any error.
func structToJSONObject(v any) map[string]any {
	if v == nil {
		return nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}
