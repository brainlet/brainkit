// Ported from: packages/deepseek/src/chat/convert-to-deepseek-usage.ts
package deepseek

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// ConvertDeepSeekUsage converts a DeepSeek API usage response into
// the standard languagemodel.Usage type.
func ConvertDeepSeekUsage(usage *DeepSeekChatTokenUsage) languagemodel.Usage {
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
	if usage.PromptCacheHitTokens != nil {
		cacheReadTokens = *usage.PromptCacheHitTokens
	}

	reasoningTokens := 0
	if usage.CompletionTokensDetails != nil && usage.CompletionTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.CompletionTokensDetails.ReasoningTokens
	}

	noCache := promptTokens - cacheReadTokens
	textTokens := completionTokens - reasoningTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:     &promptTokens,
			NoCache:   &noCache,
			CacheRead: &cacheReadTokens,
			// CacheWrite: not available from DeepSeek API
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &completionTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: structToJSONObject(usage),
	}
}

// structToJSONObject converts a struct to a map[string]any by marshaling through JSON.
// Returns nil on any error.
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
