// Ported from: packages/huggingface/src/responses/convert-huggingface-responses-usage.ts
package huggingface

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// ResponsesUsage represents the usage information from HuggingFace responses API.
type ResponsesUsage struct {
	InputTokens        int                      `json:"input_tokens"`
	InputTokensDetails *InputTokensDetails      `json:"input_tokens_details,omitempty"`
	OutputTokens       int                      `json:"output_tokens"`
	OutputTokensDetails *OutputTokensDetails    `json:"output_tokens_details,omitempty"`
	TotalTokens        int                      `json:"total_tokens"`
}

// InputTokensDetails contains details about input token usage.
type InputTokensDetails struct {
	CachedTokens *int `json:"cached_tokens,omitempty"`
}

// OutputTokensDetails contains details about output token usage.
type OutputTokensDetails struct {
	ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
}

// convertHuggingFaceResponsesUsage converts HuggingFace usage to the standard usage format.
func convertHuggingFaceResponsesUsage(usage *ResponsesUsage) languagemodel.Usage {
	if usage == nil {
		return languagemodel.Usage{
			InputTokens:  languagemodel.InputTokenUsage{},
			OutputTokens: languagemodel.OutputTokenUsage{},
			Raw:          nil,
		}
	}

	inputTokens := usage.InputTokens
	outputTokens := usage.OutputTokens

	cachedTokens := 0
	if usage.InputTokensDetails != nil && usage.InputTokensDetails.CachedTokens != nil {
		cachedTokens = *usage.InputTokensDetails.CachedTokens
	}

	reasoningTokens := 0
	if usage.OutputTokensDetails != nil && usage.OutputTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.OutputTokensDetails.ReasoningTokens
	}

	noCache := inputTokens - cachedTokens
	textTokens := outputTokens - reasoningTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:      &inputTokens,
			NoCache:    &noCache,
			CacheRead:  &cachedTokens,
			CacheWrite: nil,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &outputTokens,
			Text:      &textTokens,
			Reasoning: &reasoningTokens,
		},
		Raw: usageToJSONObject(usage),
	}
}

// usageToJSONObject converts a ResponsesUsage struct to a jsonvalue.JSONObject.
func usageToJSONObject(usage *ResponsesUsage) jsonvalue.JSONObject {
	if usage == nil {
		return nil
	}
	// Marshal and unmarshal to convert struct to map[string]any.
	b, err := json.Marshal(usage)
	if err != nil {
		return nil
	}
	var obj jsonvalue.JSONObject
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil
	}
	return obj
}
