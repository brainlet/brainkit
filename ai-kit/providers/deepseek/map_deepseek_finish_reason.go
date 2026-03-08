// Ported from: packages/deepseek/src/chat/map-deepseek-finish-reason.ts
package deepseek

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapDeepSeekFinishReason maps a DeepSeek finish reason string to a unified finish reason.
func MapDeepSeekFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
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
	case "tool_calls":
		return languagemodel.FinishReasonToolCalls
	case "insufficient_system_resource":
		return languagemodel.FinishReasonError
	default:
		return languagemodel.FinishReasonOther
	}
}
