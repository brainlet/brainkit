// Ported from: packages/openai-compatible/src/completion/convert-openai-compatible-completion-usage.ts
package openaicompatible

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// CompletionUsageRaw represents the raw usage information from an OpenAI-compatible
// completion API response.
type CompletionUsageRaw struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
}

// ConvertCompletionUsage converts raw OpenAI-compatible completion usage
// to the unified Usage type.
func ConvertCompletionUsage(usage *CompletionUsageRaw) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{
			InputTokens: languagemodel.InputTokenUsage{
				Total:      nil,
				NoCache:    nil,
				CacheRead:  nil,
				CacheWrite: nil,
			},
			OutputTokens: languagemodel.OutputTokenUsage{
				Total:     nil,
				Text:      nil,
				Reasoning: nil,
			},
			Raw: nil,
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
			Total:      &promptTokens,
			NoCache:    &promptTokens,
			CacheRead:  nil,
			CacheWrite: nil,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &completionTokens,
			Text:      &completionTokens,
			Reasoning: nil,
		},
		Raw: structToJSONObject(usage),
	}
}
