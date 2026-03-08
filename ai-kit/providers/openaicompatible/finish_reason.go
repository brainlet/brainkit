// Ported from: packages/openai-compatible/src/chat/map-openai-compatible-finish-reason.ts
// NOTE: chat/ and completion/ versions are identical; ported once here.
package openaicompatible

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapFinishReason maps an OpenAI-compatible finish reason string to the
// unified FinishReason value. Returns languagemodel.FinishReasonOther for
// unrecognized values.
func MapFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
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
