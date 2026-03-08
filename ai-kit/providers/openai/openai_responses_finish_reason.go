// Ported from: packages/openai/src/responses/map-openai-responses-finish-reason.ts
package openai

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapOpenAIResponseFinishReasonOptions are the options for MapOpenAIResponseFinishReason.
type MapOpenAIResponseFinishReasonOptions struct {
	// FinishReason is the raw finish reason from the OpenAI response.
	FinishReason *string
	// HasFunctionCall is a flag that checks if there have been client-side
	// tool calls (not executed by OpenAI).
	HasFunctionCall bool
}

// MapOpenAIResponseFinishReason maps an OpenAI Responses API finish reason to
// the unified LanguageModelV3 finish reason.
func MapOpenAIResponseFinishReason(opts MapOpenAIResponseFinishReasonOptions) languagemodel.UnifiedFinishReason {
	if opts.FinishReason == nil {
		if opts.HasFunctionCall {
			return languagemodel.FinishReasonToolCalls
		}
		return languagemodel.FinishReasonStop
	}

	switch *opts.FinishReason {
	case "max_output_tokens":
		return languagemodel.FinishReasonLength
	case "content_filter":
		return languagemodel.FinishReasonContentFilter
	default:
		if opts.HasFunctionCall {
			return languagemodel.FinishReasonToolCalls
		}
		return languagemodel.FinishReasonOther
	}
}
