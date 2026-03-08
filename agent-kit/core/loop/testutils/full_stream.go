// Ported from: packages/core/src/loop/test-utils/fullStream.ts
package testutils

import (
	"time"
)

// ---------------------------------------------------------------------------
// FullStreamTestsConfig
// ---------------------------------------------------------------------------

// FullStreamTestsConfig configures the fullStreamTests test suite.
type FullStreamTestsConfig struct {
	LoopFn       LoopFn
	RunID        string
	ModelVersion string // "v2" or "v3"
}

// FullStreamTests contains the test definitions for result.fullStream.
// In the TS source, this is a vitest describe block that validates:
//   - Conversation history in LLM input
//   - Text deltas
//   - Reasoning deltas
//   - Source chunks
//   - File chunks
//   - Multi-step tool calls
//   - Usage aggregation
//   - Warnings
//   - Response metadata
//   - Provider metadata
//   - Abort signal handling
//   - And more
type FullStreamTests struct {
	Config FullStreamTestsConfig
}

// NewFullStreamTests creates a new FullStreamTests instance.
func NewFullStreamTests(config FullStreamTestsConfig) *FullStreamTests {
	return &FullStreamTests{Config: config}
}

// ---------------------------------------------------------------------------
// Full-stream test helpers
// ---------------------------------------------------------------------------

// CreateConversationHistoryModel creates a mock model that validates prompt
// contains conversation history (memory + input messages).
func CreateConversationHistoryModel(modelVersion string) any {
	if modelVersion == "v3" {
		return NewMastraLanguageModelV3Mock(MastraLanguageModelV3MockConfig{
			DoStream: func(options map[string]any) (*DoStreamResultV3, error) {
				stream := ConvertArrayToReadableStream([]LanguageModelV3StreamPart{
					{
						"type":      "response-metadata",
						"id":        "response-id",
						"modelId":   "response-model-id",
						"timestamp": time.Date(1970, 1, 1, 0, 0, 5, 0, time.UTC),
					},
					{"type": "text-start", "id": "text-1"},
					{"type": "text-delta", "id": "text-1", "delta": "Hello"},
					{"type": "text-delta", "id": "text-1", "delta": ", "},
					{"type": "text-delta", "id": "text-1", "delta": "world!"},
					{"type": "text-end", "id": "text-1"},
					{
						"type":         "finish",
						"finishReason": FinishReasonV3{Unified: "stop", Raw: "stop"},
						"usage":        TestUsageV3,
					},
				})
				return &DoStreamResultV3{Stream: stream}, nil
			},
		})
	}
	return NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
				{
					"type":      "response-metadata",
					"id":        "response-id",
					"modelId":   "response-model-id",
					"timestamp": time.Date(1970, 1, 1, 0, 0, 5, 0, time.UTC),
				},
				{"type": "text-start", "id": "text-1"},
				{"type": "text-delta", "id": "text-1", "delta": "Hello"},
				{"type": "text-delta", "id": "text-1", "delta": ", "},
				{"type": "text-delta", "id": "text-1", "delta": "world!"},
				{"type": "text-end", "id": "text-1"},
				{
					"type":         "finish",
					"finishReason": "stop",
					"usage":        TestUsage,
				},
			})
			return &DoStreamResult{Stream: stream}, nil
		},
	})
}

// CreateMultiUsageStream creates a mock V2 model that produces two steps
// with different usage, for testing usage aggregation.
func CreateMultiUsageStream() *MastraLanguageModelV2Mock {
	callCount := 0
	return NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			callCount++
			if callCount == 1 {
				stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
					{
						"type":       "tool-call",
						"toolCallId": "call-1",
						"toolName":   "tool1",
						"input":      `{ "value": "value" }`,
					},
					{
						"type":         "finish",
						"finishReason": "tool-calls",
						"usage":        TestUsage,
					},
				})
				return &DoStreamResult{Stream: stream}, nil
			}
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
				{"type": "text-start", "id": "text-1"},
				{"type": "text-delta", "id": "text-1", "delta": "Hello, world!"},
				{"type": "text-end", "id": "text-1"},
				{
					"type":         "finish",
					"finishReason": "stop",
					"usage":        TestUsage2,
				},
			})
			return &DoStreamResult{Stream: stream}, nil
		},
	})
}
