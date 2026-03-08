// Ported from: packages/xai/src/responses/map-xai-responses-finish-reason.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// mapXaiResponsesFinishReason maps an xAI responses finish reason string to a unified finish reason.
func mapXaiResponsesFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
	if finishReason == nil {
		return languagemodel.FinishReasonOther
	}

	switch *finishReason {
	case "stop", "completed":
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
