// Ported from: packages/anthropic/src/map-anthropic-stop-reason.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// mapAnthropicStopReason maps Anthropic stop reasons to unified finish reasons.
// See https://docs.anthropic.com/en/api/messages#response-stop-reason
func mapAnthropicStopReason(finishReason *string, isJsonResponseFromTool bool) languagemodel.UnifiedFinishReason {
	if finishReason == nil {
		return languagemodel.FinishReasonOther
	}

	switch *finishReason {
	case "pause_turn", "end_turn", "stop_sequence":
		return languagemodel.FinishReasonStop
	case "refusal":
		return languagemodel.FinishReasonContentFilter
	case "tool_use":
		if isJsonResponseFromTool {
			return languagemodel.FinishReasonStop
		}
		return languagemodel.FinishReasonToolCalls
	case "max_tokens", "model_context_window_exceeded":
		return languagemodel.FinishReasonLength
	case "compaction":
		return languagemodel.FinishReasonOther
	default:
		return languagemodel.FinishReasonOther
	}
}
