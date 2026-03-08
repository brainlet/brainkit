// Ported from: packages/mistral/src/map-mistral-finish-reason.ts
package mistral

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapMistralFinishReason maps a Mistral finish reason string to the unified finish reason.
func MapMistralFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
	if finishReason == nil {
		return languagemodel.FinishReasonOther
	}

	switch *finishReason {
	case "stop":
		return languagemodel.FinishReasonStop
	case "length", "model_length":
		return languagemodel.FinishReasonLength
	case "tool_calls":
		return languagemodel.FinishReasonToolCalls
	default:
		return languagemodel.FinishReasonOther
	}
}
