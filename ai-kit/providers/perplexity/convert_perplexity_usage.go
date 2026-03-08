// Ported from: packages/perplexity/src/convert-perplexity-usage.ts
package perplexity

import (
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// PerplexityUsage represents the raw usage data from the Perplexity API.
type PerplexityUsage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	ReasoningTokens  *int `json:"reasoning_tokens,omitempty"`
}

// ConvertPerplexityUsage converts Perplexity-specific usage information
// to the standard LanguageModelV3Usage format.
func ConvertPerplexityUsage(usage *PerplexityUsage) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{}
	}

	promptTokens := 0
	if usage.PromptTokens != nil {
		promptTokens = *usage.PromptTokens
	}
	completionTokens := 0
	if usage.CompletionTokens != nil {
		completionTokens = *usage.CompletionTokens
	}
	reasoningTokens := 0
	if usage.ReasoningTokens != nil {
		reasoningTokens = *usage.ReasoningTokens
	}

	textTokens := completionTokens - reasoningTokens

	// Build raw usage as a JSONObject (map[string]any)
	raw := jsonvalue.JSONObject{
		"prompt_tokens":     usage.PromptTokens,
		"completion_tokens": usage.CompletionTokens,
		"reasoning_tokens":  usage.ReasoningTokens,
	}

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:   &promptTokens,
			NoCache: &promptTokens,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &completionTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: raw,
	}
}
