// Ported from: packages/groq/src/map-groq-finish-reason.ts
package groq

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapGroqFinishReason maps a Groq finish reason string to the unified FinishReason value.
func MapGroqFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
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
