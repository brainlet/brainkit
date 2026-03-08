// Ported from: packages/core/src/loop/test-utils/generateText.ts
package testutils

// ---------------------------------------------------------------------------
// GenerateTextTestsConfig
// ---------------------------------------------------------------------------

// GenerateTextTestsConfig configures the generateTextTestsV5 test suite.
type GenerateTextTestsConfig struct {
	LoopFn LoopFn
	RunID  string
}

// GenerateTextTestsV5 contains the V5 test definitions for generateText.
// In the TS source, this is a vitest describe block that validates:
//   - result.text (basic text generation)
//   - result.content (structured content with tool calls, sources, files, reasoning)
//   - result.reasoningText (reasoning extraction)
//   - result.sources (source attribution)
//   - result.files (generated files)
//   - result.toolCalls (tool call extraction)
//   - result.toolResults (tool result extraction)
//   - result.finishReason
//   - result.usage
//   - result.warnings
//   - result.response (metadata)
//   - result.response.headers
//   - result.request
//   - result.steps (multi-step scenarios)
//   - result.providerMetadata
//
// Each test creates a message list with a user message, invokes loopFn
// with methodType "generate", and asserts the full output shape.
type GenerateTextTestsV5 struct {
	Config GenerateTextTestsConfig
}

// NewGenerateTextTestsV5 creates a new GenerateTextTestsV5 instance.
func NewGenerateTextTestsV5(config GenerateTextTestsConfig) *GenerateTextTestsV5 {
	return &GenerateTextTestsV5{Config: config}
}

// ---------------------------------------------------------------------------
// DummyResponseValues
// ---------------------------------------------------------------------------

// DummyResponseValues holds default response values used across generate-text tests.
var DummyResponseValues = map[string]any{
	"finishReason": "stop",
	"usage": map[string]any{
		"inputTokens":       3,
		"outputTokens":      10,
		"totalTokens":       13,
		"reasoningTokens":   nil,
		"cachedInputTokens": nil,
	},
	"warnings": []any{},
}

// ---------------------------------------------------------------------------
// GenerateText helpers
// ---------------------------------------------------------------------------

// CreateGenerateTextModelWithToolCalls creates a V2 mock model that produces
// a tool call followed by text response for generate-text multi-step tests.
func CreateGenerateTextModelWithToolCalls(
	toolCallID string,
	toolName string,
	input string,
	textResponse string,
) *MastraLanguageModelV2Mock {
	callCount := 0
	return NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			callCount++
			if callCount == 1 {
				stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
					{
						"type":       "tool-call",
						"toolCallId": toolCallID,
						"toolName":   toolName,
						"input":      input,
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
				{"type": "text-delta", "id": "text-1", "delta": textResponse},
				{"type": "text-end", "id": "text-1"},
				{
					"type":         "finish",
					"finishReason": "stop",
					"usage":        TestUsage,
				},
			})
			return &DoStreamResult{Stream: stream}, nil
		},
		DoGenerate: func(options map[string]any) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Content: []map[string]any{
					{"type": "text", "text": textResponse},
				},
				FinishReason: "stop",
				Usage:        TestUsage,
			}, nil
		},
	})
}

// CreateGenerateTextModelWithSources returns the ModelWithSources mock.
// This is a convenience alias to match the TS test pattern.
func CreateGenerateTextModelWithSources() *MastraLanguageModelV2Mock {
	return ModelWithSources
}

// CreateGenerateTextModelWithReasoning returns the ModelWithReasoning mock.
// This is a convenience alias to match the TS test pattern.
func CreateGenerateTextModelWithReasoning() *MastraLanguageModelV2Mock {
	return ModelWithReasoning
}

// CreateGenerateTextModelWithFiles returns the ModelWithFiles mock.
// This is a convenience alias to match the TS test pattern.
func CreateGenerateTextModelWithFiles() *MastraLanguageModelV2Mock {
	return ModelWithFiles
}
