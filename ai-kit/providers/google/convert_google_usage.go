// Ported from: packages/google/src/convert-google-generative-ai-usage.ts
package google

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// GoogleUsageMetadata represents the usage metadata returned by the Google
// Generative AI API.
type GoogleUsageMetadata struct {
	PromptTokenCount      *int    `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount  *int    `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount       *int    `json:"totalTokenCount,omitempty"`
	CachedContentTokenCount *int  `json:"cachedContentTokenCount,omitempty"`
	ThoughtsTokenCount    *int    `json:"thoughtsTokenCount,omitempty"`
	TrafficType           *string `json:"trafficType,omitempty"`
}

// ConvertGoogleUsage converts Google Generative AI usage metadata to the
// standard LanguageModel Usage type.
func ConvertGoogleUsage(usage *GoogleUsageMetadata) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{}
	}

	promptTokens := intOrZero(usage.PromptTokenCount)
	candidatesTokens := intOrZero(usage.CandidatesTokenCount)
	cachedContentTokens := intOrZero(usage.CachedContentTokenCount)
	thoughtsTokens := intOrZero(usage.ThoughtsTokenCount)

	noCache := promptTokens - cachedContentTokens
	outputTotal := candidatesTokens + thoughtsTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:     intPtr(promptTokens),
			NoCache:   intPtr(noCache),
			CacheRead: intPtr(cachedContentTokens),
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     intPtr(outputTotal),
			Text:      intPtr(candidatesTokens),
			Reasoning: intPtr(thoughtsTokens),
		},
		Raw: usageToRaw(usage),
	}
}

func intOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func intPtr(v int) *int {
	return &v
}

func usageToRaw(usage *GoogleUsageMetadata) map[string]any {
	if usage == nil {
		return nil
	}
	raw := make(map[string]any)
	if usage.PromptTokenCount != nil {
		raw["promptTokenCount"] = *usage.PromptTokenCount
	}
	if usage.CandidatesTokenCount != nil {
		raw["candidatesTokenCount"] = *usage.CandidatesTokenCount
	}
	if usage.TotalTokenCount != nil {
		raw["totalTokenCount"] = *usage.TotalTokenCount
	}
	if usage.CachedContentTokenCount != nil {
		raw["cachedContentTokenCount"] = *usage.CachedContentTokenCount
	}
	if usage.ThoughtsTokenCount != nil {
		raw["thoughtsTokenCount"] = *usage.ThoughtsTokenCount
	}
	if usage.TrafficType != nil {
		raw["trafficType"] = *usage.TrafficType
	}
	return raw
}
