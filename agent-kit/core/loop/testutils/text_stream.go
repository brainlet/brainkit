// Ported from: packages/core/src/loop/test-utils/textStream.ts
package testutils

// ---------------------------------------------------------------------------
// Stub types for unported packages
// ---------------------------------------------------------------------------

// LoopFn is a stub for the loop function signature used in test suites.
// TODO: import from loop package once ported.
type LoopFn func(opts any) any

// ---------------------------------------------------------------------------
// TextStreamTestsConfig
// ---------------------------------------------------------------------------

// TextStreamTestsConfig configures the textStreamTests test suite.
type TextStreamTestsConfig struct {
	LoopFn LoopFn
	RunID  string
}

// TextStreamTests contains the test definitions for result.textStream.
// In the TS source, this is a vitest describe block that validates text
// deltas are correctly sent through the stream.
//
// The test:
//  1. Creates a message list with a single user message.
//  2. Invokes loopFn with methodType "stream" and a mock model that
//     produces ["Hello", ", ", "world!"] text deltas.
//  3. Asserts that result.textStream yields those three delta strings.
//
// In Go, the actual test runner will instantiate these scenarios; this file
// provides the configuration types and helpers to mirror the TS structure.
type TextStreamTests struct {
	Config TextStreamTestsConfig
}

// NewTextStreamTests creates a new TextStreamTests instance.
func NewTextStreamTests(config TextStreamTestsConfig) *TextStreamTests {
	return &TextStreamTests{Config: config}
}

// CreateTextStreamModel creates a mock V2 model that emits text deltas
// for "Hello", ", ", "world!" — the default text-stream test scenario.
func CreateTextStreamModel() *MastraLanguageModelV2Mock {
	return NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
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
