// Ported from: packages/openai/src/chat/convert-openai-chat-usage.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// OpenAIChatUsage represents the token usage from an OpenAI chat API response.
type OpenAIChatUsage struct {
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

// ConvertOpenAIChatUsage converts an OpenAI chat API usage response into
// the standard languagemodel.Usage type.
func ConvertOpenAIChatUsage(usage *OpenAIChatUsage) languagemodel.Usage {
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

	cachedTokens := 0
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens != nil {
		cachedTokens = *usage.PromptTokensDetails.CachedTokens
	}

	reasoningTokens := 0
	if usage.CompletionTokensDetails != nil && usage.CompletionTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.CompletionTokensDetails.ReasoningTokens
	}

	noCache := promptTokens - cachedTokens
	textTokens := completionTokens - reasoningTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:    &promptTokens,
			NoCache:  &noCache,
			CacheRead: &cachedTokens,
			// CacheWrite: not available from OpenAI chat API
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &completionTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: chatUsageToJSONObject(usage),
	}
}

// chatUsageToJSONObject converts a usage struct to a map[string]any via JSON.
func chatUsageToJSONObject(v any) map[string]any {
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
