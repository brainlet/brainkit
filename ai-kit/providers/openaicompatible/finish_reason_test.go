// Ported from: packages/openai-compatible/src/chat/map-openai-compatible-finish-reason.ts (tests)
package openaicompatible

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestMapFinishReason(t *testing.T) {
	t.Run("should return stop for 'stop'", func(t *testing.T) {
		reason := "stop"
		result := MapFinishReason(&reason)
		if result != languagemodel.FinishReasonStop {
			t.Errorf("expected FinishReasonStop, got %v", result)
		}
	})

	t.Run("should return length for 'length'", func(t *testing.T) {
		reason := "length"
		result := MapFinishReason(&reason)
		if result != languagemodel.FinishReasonLength {
			t.Errorf("expected FinishReasonLength, got %v", result)
		}
	})

	t.Run("should return content_filter for 'content_filter'", func(t *testing.T) {
		reason := "content_filter"
		result := MapFinishReason(&reason)
		if result != languagemodel.FinishReasonContentFilter {
			t.Errorf("expected FinishReasonContentFilter, got %v", result)
		}
	})

	t.Run("should return tool_calls for 'function_call'", func(t *testing.T) {
		reason := "function_call"
		result := MapFinishReason(&reason)
		if result != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected FinishReasonToolCalls, got %v", result)
		}
	})

	t.Run("should return tool_calls for 'tool_calls'", func(t *testing.T) {
		reason := "tool_calls"
		result := MapFinishReason(&reason)
		if result != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected FinishReasonToolCalls, got %v", result)
		}
	})

	t.Run("should return other for nil", func(t *testing.T) {
		result := MapFinishReason(nil)
		if result != languagemodel.FinishReasonOther {
			t.Errorf("expected FinishReasonOther, got %v", result)
		}
	})

	t.Run("should return other for unknown value", func(t *testing.T) {
		reason := "unknown_reason"
		result := MapFinishReason(&reason)
		if result != languagemodel.FinishReasonOther {
			t.Errorf("expected FinishReasonOther, got %v", result)
		}
	})
}
