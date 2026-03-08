// Ported from: packages/mistral/src/convert-mistral-usage.ts
package mistral

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MistralUsage represents the usage information from a Mistral API response.
type MistralUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ConvertMistralUsage converts Mistral usage information to the standard Usage type.
func ConvertMistralUsage(usage *MistralUsage) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{}
	}

	promptTokens := usage.PromptTokens
	completionTokens := usage.CompletionTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:   &promptTokens,
			NoCache: &promptTokens,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total: &completionTokens,
			Text:  &completionTokens,
		},
		Raw: map[string]any{
			"prompt_tokens":     usage.PromptTokens,
			"completion_tokens": usage.CompletionTokens,
			"total_tokens":      usage.TotalTokens,
		},
	}
}
