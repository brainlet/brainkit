// Ported from: packages/xai/src/responses/convert-xai-responses-usage.ts
package xai

import (
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// convertXaiResponsesUsage converts xAI responses API usage to the standard format.
func convertXaiResponsesUsage(usage XaiResponsesUsage) languagemodel.Usage {
	cacheReadTokens := 0
	if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens != nil {
		cacheReadTokens = *usage.InputTokensDetails.CachedTokens
	}

	reasoningTokens := 0
	if usage.OutputTokensDetails != nil && usage.OutputTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.OutputTokensDetails.ReasoningTokens
	}

	inputTokensIncludesCached := cacheReadTokens <= usage.InputTokens

	var total int
	var noCache int
	if inputTokensIncludesCached {
		total = usage.InputTokens
		noCache = usage.InputTokens - cacheReadTokens
	} else {
		total = usage.InputTokens + cacheReadTokens
		noCache = usage.InputTokens
	}

	textTokens := usage.OutputTokens - reasoningTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:      &total,
			NoCache:    &noCache,
			CacheRead:  &cacheReadTokens,
			CacheWrite: nil,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &usage.OutputTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: mapToJSONObject(usage),
	}
}
