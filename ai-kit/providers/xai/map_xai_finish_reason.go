// Ported from: packages/xai/src/map-xai-finish-reason.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// mapXaiFinishReason maps an xAI finish reason string to a unified finish reason.
func mapXaiFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
	if finishReason == nil {
		return languagemodel.FinishReasonOther
	}

	switch *finishReason {
	case "stop":
		return languagemodel.FinishReasonStop
	case "length":
		return languagemodel.FinishReasonLength
	case "tool_calls", "function_call":
		return languagemodel.FinishReasonToolCalls
	case "content_filter":
		return languagemodel.FinishReasonContentFilter
	default:
		return languagemodel.FinishReasonOther
	}
}
