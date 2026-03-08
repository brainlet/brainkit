// Ported from: packages/openai/src/responses/convert-openai-responses-usage.ts
package openai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// OpenAIResponsesUsage represents the usage information from an OpenAI Responses API call.
type OpenAIResponsesUsage struct {
	InputTokens        int                           `json:"input_tokens"`
	OutputTokens       int                           `json:"output_tokens"`
	InputTokensDetails *OpenAIResponsesInputDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *OpenAIResponsesOutputDetails `json:"output_tokens_details,omitempty"`
}

// OpenAIResponsesInputDetails contains details about input token usage.
type OpenAIResponsesInputDetails struct {
	CachedTokens *int `json:"cached_tokens,omitempty"`
}

// OpenAIResponsesOutputDetails contains details about output token usage.
type OpenAIResponsesOutputDetails struct {
	ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
}

// ConvertOpenAIResponsesUsage converts OpenAI Responses API usage to the
// unified LanguageModelV3 usage format.
func ConvertOpenAIResponsesUsage(usage *OpenAIResponsesUsage) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{}
	}

	inputTokens := usage.InputTokens
	outputTokens := usage.OutputTokens

	var cachedTokens int
	if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens != nil {
		cachedTokens = *usage.InputTokensDetails.CachedTokens
	}

	var reasoningTokens int
	if usage.OutputTokensDetails != nil && usage.OutputTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.OutputTokensDetails.ReasoningTokens
	}

	noCache := inputTokens - cachedTokens
	textTokens := outputTokens - reasoningTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:   &inputTokens,
			NoCache: &noCache,
			CacheRead: &cachedTokens,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &outputTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: responsesUsageToJSONObject(usage),
	}
}

// responsesUsageToJSONObject converts a usage struct to a map[string]any via JSON.
func responsesUsageToJSONObject(v any) map[string]any {
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
