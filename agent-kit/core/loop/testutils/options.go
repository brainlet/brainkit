// Ported from: packages/core/src/loop/test-utils/options.ts
package testutils

// ---------------------------------------------------------------------------
// Stub types for unported packages (options-specific)
// ---------------------------------------------------------------------------

// ChunkType is a stub for ../../stream/types.ChunkType.
// TODO: import from stream package once ported.
type ChunkType = map[string]any

// ---------------------------------------------------------------------------
// OptionsTestsConfig
// ---------------------------------------------------------------------------

// OptionsTestsConfig configures the optionsTests test suite.
type OptionsTestsConfig struct {
	LoopFn LoopFn
	RunID  string
}

// OptionsTests contains the test definitions for loop options.
// In the TS source, this is a vitest describe block (~8400 lines) that validates:
//
// options.abortSignal:
//   - Forward abort signal to tool execution during streaming
//   - Stream abortion and cleanup
//
// options.onError:
//   - Error callback invocation
//   - Error propagation
//
// options.onFinish:
//   - Finish callback with final result data
//   - Usage, warnings, response metadata in onFinish
//
// options.onStepFinish:
//   - Per-step finish callbacks
//   - Multi-step tool-call scenarios
//   - Usage per step
//
// options.onChunk:
//   - Chunk callback for each stream part
//   - Text deltas, tool calls, reasoning, sources
//
// tools:
//   - Tool execution and result mapping
//   - Tool call streaming (input start/delta/end)
//   - Error handling for tool calls
//   - Tool not found scenarios
//   - Dynamic tools
//   - Provider-executed tools
//   - Multi-step tool interactions
//   - stepCountIs stop condition
//
// stopWhen:
//   - Custom stop conditions
//   - Step count limits
//
// toolChoice:
//   - Auto, none, required modes
//   - Named tool choice
//
// activeTools:
//   - Active tool filtering
//
// providerOptions:
//   - Provider-specific options forwarding
//
// model fallback:
//   - Fallback to secondary model on error
//   - Retry with exponential backoff
//
// processor integration:
//   - Input processors
//   - Output processors
//   - Tripwire handling
//   - Processor retry
//
// structured output:
//   - JSON schema output
//   - Zod schema output
//
// resume context:
//   - Resume from suspended state
//   - Tool approval flow
//   - Tool suspension flow
type OptionsTests struct {
	Config OptionsTestsConfig
}

// NewOptionsTests creates a new OptionsTests instance.
func NewOptionsTests(config OptionsTestsConfig) *OptionsTests {
	return &OptionsTests{Config: config}
}

// ---------------------------------------------------------------------------
// Options test helpers
// ---------------------------------------------------------------------------

// AbortController mirrors the browser AbortController for test usage.
type AbortController struct {
	Signal *AbortSignal
}

// AbortSignal is a simplified abort signal for test usage.
type AbortSignal struct {
	Aborted bool
}

// NewAbortController creates a new AbortController.
func NewAbortController() *AbortController {
	return &AbortController{
		Signal: &AbortSignal{Aborted: false},
	}
}

// Abort marks the signal as aborted.
func (ac *AbortController) Abort() {
	ac.Signal.Aborted = true
}

// CreateErrorModel creates a mock model that throws an error on doStream.
func CreateErrorModel(errMsg string) *MastraLanguageModelV2Mock {
	return NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			return nil, &mockError{msg: errMsg}
		},
		DoGenerate: func(options map[string]any) (*DoGenerateResult, error) {
			return nil, &mockError{msg: errMsg}
		},
	})
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}

// CreateFallbackModels creates a pair of models where the first fails and the
// second succeeds — used for testing model fallback behavior.
func CreateFallbackModels(errMsg string) []ModelManagerModelConfig {
	failModel := CreateErrorModel(errMsg)
	successModel := NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
				{"type": "text-start", "id": "text-1"},
				{"type": "text-delta", "id": "text-1", "delta": "Fallback response"},
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

	return []ModelManagerModelConfig{
		{Model: failModel, MaxRetries: 0, ID: "primary-model"},
		{Model: successModel, MaxRetries: 0, ID: "fallback-model"},
	}
}

// CreateRetryModels creates a model config with maxRetries > 0 for testing
// retry with exponential backoff.
func CreateRetryModels(maxRetries int) []ModelManagerModelConfig {
	callCount := 0
	model := NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			callCount++
			if callCount <= maxRetries {
				return nil, &mockError{msg: "transient error"}
			}
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
				{"type": "text-start", "id": "text-1"},
				{"type": "text-delta", "id": "text-1", "delta": "Success after retry"},
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

	return []ModelManagerModelConfig{
		{Model: model, MaxRetries: maxRetries, ID: "retry-model"},
	}
}
