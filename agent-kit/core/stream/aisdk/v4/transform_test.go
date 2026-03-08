// Ported from: packages/core/src/stream/aisdk/v4/transform.test.ts
package v4

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func TestConvertFullStreamChunkToMastraV4(t *testing.T) {
	ctx := TransformContext{RunID: "run-1"}

	t.Run("should convert step-start", func(t *testing.T) {
		value := LanguageModelV1StreamPart{
			Type:      "step-start",
			MessageID: "msg-1",
			Request:   &RequestData{Body: `{"messages":[]}`},
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "step-start" {
			t.Errorf("expected type 'step-start', got %q", result.Type)
		}
		if result.RunID != "run-1" {
			t.Errorf("expected runID 'run-1', got %q", result.RunID)
		}
	})

	t.Run("should convert step-start with nil request", func(t *testing.T) {
		value := LanguageModelV1StreamPart{
			Type: "step-start",
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "step-start" {
			t.Errorf("expected type 'step-start', got %q", result.Type)
		}
	})

	t.Run("should convert tool-call", func(t *testing.T) {
		value := LanguageModelV1StreamPart{
			Type:       "tool-call",
			ToolCallID: "tc-1",
			ToolName:   "search",
			Args:       map[string]any{"query": "test"},
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-call" {
			t.Errorf("expected type 'tool-call', got %q", result.Type)
		}
		if result.From != stream.ChunkFromAgent {
			t.Errorf("expected From to be agent, got %q", result.From)
		}
		payload, ok := result.Payload.(stream.ToolCallPayload)
		if !ok {
			t.Fatal("expected ToolCallPayload")
		}
		if payload.ToolCallID != "tc-1" {
			t.Errorf("expected toolCallId 'tc-1', got %q", payload.ToolCallID)
		}
		if payload.ToolName != "search" {
			t.Errorf("expected toolName 'search', got %q", payload.ToolName)
		}
	})

	t.Run("should convert tool-result", func(t *testing.T) {
		value := LanguageModelV1StreamPart{
			Type:       "tool-result",
			ToolCallID: "tc-1",
			ToolName:   "search",
			Result:     map[string]any{"data": "found"},
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-result" {
			t.Errorf("expected type 'tool-result', got %q", result.Type)
		}
		payload, ok := result.Payload.(stream.ToolResultPayload)
		if !ok {
			t.Fatal("expected ToolResultPayload")
		}
		if payload.ToolCallID != "tc-1" {
			t.Errorf("expected toolCallId 'tc-1', got %q", payload.ToolCallID)
		}
	})

	t.Run("should convert text-delta", func(t *testing.T) {
		value := LanguageModelV1StreamPart{
			Type:      "text-delta",
			ID:        "t1",
			TextDelta: "Hello World",
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", result.Type)
		}
		payload, ok := result.Payload.(stream.TextDeltaPayload)
		if !ok {
			t.Fatal("expected TextDeltaPayload")
		}
		if payload.Text != "Hello World" {
			t.Errorf("expected text 'Hello World', got %q", payload.Text)
		}
		if payload.ID != "t1" {
			t.Errorf("expected ID 't1', got %q", payload.ID)
		}
	})

	t.Run("should convert step-finish", func(t *testing.T) {
		totalTokens := 13
		value := LanguageModelV1StreamPart{
			Type:         "step-finish",
			FinishReason: "stop",
			Usage: &LanguageModelUsageV4{
				PromptTokens:     3,
				CompletionTokens: 10,
				TotalTokens:      &totalTokens,
			},
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "step-finish" {
			t.Errorf("expected type 'step-finish', got %q", result.Type)
		}
		payload, ok := result.Payload.(stream.StepFinishPayload)
		if !ok {
			t.Fatal("expected StepFinishPayload")
		}
		if payload.StepResult.Reason != stream.LanguageModelV2FinishReason("stop") {
			t.Errorf("expected reason 'stop', got %q", payload.StepResult.Reason)
		}
		if payload.Output.Usage.InputTokens != 3 {
			t.Errorf("expected inputTokens 3, got %d", payload.Output.Usage.InputTokens)
		}
		if payload.Output.Usage.OutputTokens != 10 {
			t.Errorf("expected outputTokens 10, got %d", payload.Output.Usage.OutputTokens)
		}
	})

	t.Run("should convert finish", func(t *testing.T) {
		totalTokens := 30
		value := LanguageModelV1StreamPart{
			Type:         "finish",
			FinishReason: "stop",
			Usage: &LanguageModelUsageV4{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      &totalTokens,
			},
			TotalUsage: &LanguageModelUsageV4{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:      &totalTokens,
			},
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "finish" {
			t.Errorf("expected type 'finish', got %q", result.Type)
		}
		payload, ok := result.Payload.(FinishChunkPayload)
		if !ok {
			t.Fatal("expected FinishChunkPayload")
		}
		if payload.Usage.InputTokens != 10 {
			t.Errorf("expected usage inputTokens 10, got %d", payload.Usage.InputTokens)
		}
		if payload.TotalUsage.OutputTokens != 20 {
			t.Errorf("expected totalUsage outputTokens 20, got %d", payload.TotalUsage.OutputTokens)
		}
	})

	t.Run("should convert finish with nil messages", func(t *testing.T) {
		value := LanguageModelV1StreamPart{
			Type:         "finish",
			FinishReason: "stop",
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		payload, ok := result.Payload.(FinishChunkPayload)
		if !ok {
			t.Fatal("expected FinishChunkPayload")
		}
		// When messages is nil, should get empty arrays
		if payload.Messages.All == nil {
			t.Error("expected non-nil All messages")
		}
		if len(payload.Messages.All) != 0 {
			t.Errorf("expected 0 All messages, got %d", len(payload.Messages.All))
		}
	})

	t.Run("should convert tripwire", func(t *testing.T) {
		retry := true
		value := LanguageModelV1StreamPart{
			Type:        "tripwire",
			Reason:      "pii detected",
			Retry:       &retry,
			ProcessorID: "pii-detector",
			Metadata:    map[string]any{"match": "ssn"},
		}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tripwire" {
			t.Errorf("expected type 'tripwire', got %q", result.Type)
		}
		payload, ok := result.Payload.(stream.TripwirePayload)
		if !ok {
			t.Fatal("expected TripwirePayload")
		}
		if payload.Reason != "pii detected" {
			t.Errorf("expected reason 'pii detected', got %q", payload.Reason)
		}
		if payload.Retry == nil || !*payload.Retry {
			t.Error("expected retry to be true")
		}
		if payload.ProcessorID != "pii-detector" {
			t.Errorf("expected processorId 'pii-detector', got %q", payload.ProcessorID)
		}
	})

	t.Run("should return nil for unknown type", func(t *testing.T) {
		value := LanguageModelV1StreamPart{Type: "unknown"}
		result := ConvertFullStreamChunkToMastra(value, ctx)
		if result != nil {
			t.Error("expected nil for unknown type")
		}
	})
}

func TestToUsage(t *testing.T) {
	t.Run("should handle nil usage", func(t *testing.T) {
		result := toUsage(nil)
		if result.InputTokens != 0 || result.OutputTokens != 0 {
			t.Errorf("expected zero usage, got input=%d output=%d", result.InputTokens, result.OutputTokens)
		}
	})

	t.Run("should convert V4 usage", func(t *testing.T) {
		totalTokens := 15
		result := toUsage(&LanguageModelUsageV4{
			PromptTokens:     5,
			CompletionTokens: 10,
			TotalTokens:      &totalTokens,
		})
		if result.InputTokens != 5 {
			t.Errorf("expected inputTokens 5, got %d", result.InputTokens)
		}
		if result.OutputTokens != 10 {
			t.Errorf("expected outputTokens 10, got %d", result.OutputTokens)
		}
		if result.TotalTokens != 15 {
			t.Errorf("expected totalTokens 15, got %d", result.TotalTokens)
		}
	})

	t.Run("should handle nil totalTokens", func(t *testing.T) {
		result := toUsage(&LanguageModelUsageV4{
			PromptTokens:     5,
			CompletionTokens: 10,
		})
		if result.TotalTokens != 0 {
			t.Errorf("expected totalTokens 0, got %d", result.TotalTokens)
		}
	})
}

func TestToCallWarnings(t *testing.T) {
	t.Run("should handle nil warnings", func(t *testing.T) {
		result := toCallWarnings(nil)
		if result != nil {
			t.Error("expected nil for nil warnings")
		}
	})

	t.Run("should convert map warnings", func(t *testing.T) {
		warnings := []any{
			map[string]any{
				"type":    "unsupported-setting",
				"setting": "temperature",
				"message": "not supported",
			},
		}
		result := toCallWarnings(warnings)
		if len(result) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result))
		}
		if result[0].Type != "unsupported-setting" {
			t.Errorf("expected type 'unsupported-setting', got %q", result[0].Type)
		}
		if result[0].Setting != "temperature" {
			t.Errorf("expected setting 'temperature', got %q", result[0].Setting)
		}
	})

	t.Run("should pass through typed warnings", func(t *testing.T) {
		warnings := []any{
			stream.LanguageModelV2CallWarning{
				Type:    "unsupported-setting",
				Setting: "top_p",
			},
		}
		result := toCallWarnings(warnings)
		if len(result) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result))
		}
		if result[0].Setting != "top_p" {
			t.Errorf("expected setting 'top_p', got %q", result[0].Setting)
		}
	})
}

func TestToRequestMetadata(t *testing.T) {
	t.Run("should handle nil request", func(t *testing.T) {
		result := toRequestMetadata(nil)
		if result != nil {
			t.Error("expected nil for nil request")
		}
	})

	t.Run("should convert request data", func(t *testing.T) {
		result := toRequestMetadata(&RequestData{Body: `{"prompt":"test"}`})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Body == nil {
			t.Fatal("expected non-nil body")
		}
	})
}
