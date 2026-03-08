// Ported from: packages/openai/src/completion/map-openai-finish-reason.ts
package openai

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapOpenAICompletionFinishReason maps an OpenAI completion finish reason string to the
// unified FinishReason value. Returns FinishReasonOther for unrecognized values.
func MapOpenAICompletionFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
	if finishReason == nil {
		return languagemodel.FinishReasonOther
	}
	switch *finishReason {
	case "stop":
		return languagemodel.FinishReasonStop
	case "length":
		return languagemodel.FinishReasonLength
	case "content_filter":
		return languagemodel.FinishReasonContentFilter
	case "function_call", "tool_calls":
		return languagemodel.FinishReasonToolCalls
	default:
		return languagemodel.FinishReasonOther
	}
}
