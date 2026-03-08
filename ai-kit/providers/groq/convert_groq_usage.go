// Ported from: packages/groq/src/convert-groq-usage.ts
package groq

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// GroqTokenUsage represents the token usage from a Groq API response.
type GroqTokenUsage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	TotalTokens      *int `json:"total_tokens,omitempty"`

	PromptTokensDetails *struct {
		CachedTokens *int `json:"cached_tokens,omitempty"`
	} `json:"prompt_tokens_details,omitempty"`

	CompletionTokensDetails *struct {
		ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
	} `json:"completion_tokens_details,omitempty"`
}

// ConvertGroqUsage converts a Groq API usage response into the standard languagemodel.Usage type.
func ConvertGroqUsage(usage *GroqTokenUsage) languagemodel.Usage {
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

	var reasoningTokens *int
	if usage.CompletionTokensDetails != nil && usage.CompletionTokensDetails.ReasoningTokens != nil {
		reasoningTokens = usage.CompletionTokensDetails.ReasoningTokens
	}

	textTokens := completionTokens
	if reasoningTokens != nil {
		textTokens = completionTokens - *reasoningTokens
	}

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:   &promptTokens,
			NoCache: &promptTokens,
			// CacheRead and CacheWrite are not available from Groq
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &completionTokens,
			Text:      &textTokens,
			Reasoning: reasoningTokens,
		},
		Raw: structToJSONObject(usage),
	}
}

// structToJSONObject converts a struct to a map[string]any by marshaling through JSON.
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
