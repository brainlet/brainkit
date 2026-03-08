// Ported from: packages/openai/src/completion/convert-openai-completion-usage.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// OpenAICompletionUsage represents the token usage from an OpenAI completion API response.
type OpenAICompletionUsage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	TotalTokens      *int `json:"total_tokens,omitempty"`
}

// ConvertOpenAICompletionUsage converts an OpenAI completion API usage response into
// the standard languagemodel.Usage type.
func ConvertOpenAICompletionUsage(usage *OpenAICompletionUsage) languagemodel.Usage {
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

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:   usage.PromptTokens,
			NoCache: &promptTokens,
			// CacheRead: not available from OpenAI completion API
			// CacheWrite: not available from OpenAI completion API
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total: usage.CompletionTokens,
			Text:  &completionTokens,
			// Reasoning: not available from OpenAI completion API
		},
		Raw: completionUsageToJSONObject(usage),
	}
}

// completionUsageToJSONObject converts a usage struct to a map[string]any via JSON.
func completionUsageToJSONObject(v any) map[string]any {
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
