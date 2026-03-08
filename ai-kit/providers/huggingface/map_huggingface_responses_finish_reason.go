// Ported from: packages/huggingface/src/responses/map-huggingface-responses-finish-reason.ts
package huggingface

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// mapHuggingFaceResponsesFinishReason maps a HuggingFace finish reason string
// to a unified FinishReason.
func mapHuggingFaceResponsesFinishReason(finishReason string) languagemodel.UnifiedFinishReason {
	switch finishReason {
	case "stop":
		return languagemodel.FinishReasonStop
	case "length":
		return languagemodel.FinishReasonLength
	case "content_filter":
		return languagemodel.FinishReasonContentFilter
	case "tool_calls":
		return languagemodel.FinishReasonToolCalls
	case "error":
		return languagemodel.FinishReasonError
	default:
		return languagemodel.FinishReasonOther
	}
}
