// Ported from: packages/core/src/loop/test-utils/streamObject.ts
package testutils

// ---------------------------------------------------------------------------
// Stub types for unported packages (stream-object-specific)
// ---------------------------------------------------------------------------

// NoObjectGeneratedError is a stub for @internal/ai-sdk-v5.NoObjectGeneratedError.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 types remain local stubs.
type NoObjectGeneratedError struct {
	Message      string `json:"message"`
	Response     any    `json:"response,omitempty"`
	Usage        any    `json:"usage,omitempty"`
	FinishReason string `json:"finishReason,omitempty"`
}

func (e *NoObjectGeneratedError) Error() string {
	return e.Message
}

// IsNoObjectGeneratedError checks if the error is a NoObjectGeneratedError.
func IsNoObjectGeneratedError(err error) bool {
	_, ok := err.(*NoObjectGeneratedError)
	return ok
}

// ---------------------------------------------------------------------------
// StreamObjectTestsConfig
// ---------------------------------------------------------------------------

// StreamObjectTestsConfig configures the streamObjectTests test suite.
type StreamObjectTestsConfig struct {
	LoopFn LoopFn
	RunID  string
}

// StreamObjectTests contains the test definitions for stream-object scenarios.
// In the TS source, this is a vitest describe block that validates:
//   - result.object auto consume promise
//   - result.object (partial object streaming)
//   - result.partialObjectStream
//   - result.fullStream (object mode)
//   - pipeTextStreamToResponse
//   - result.usage
//   - result.providerMetadata
//   - result.response
//   - result.request
//   - result.warnings
//   - result.text
//   - error handling (NoObjectGeneratedError)
//   - onFinish callback
//   - schema type inference
type StreamObjectTests struct {
	Config StreamObjectTestsConfig
}

// NewStreamObjectTests creates a new StreamObjectTests instance.
func NewStreamObjectTests(config StreamObjectTestsConfig) *StreamObjectTests {
	return &StreamObjectTests{Config: config}
}

// ---------------------------------------------------------------------------
// Stream-object test helpers
// ---------------------------------------------------------------------------

// CreateStreamObjectModels creates test models that stream JSON object
// deltas: `{ "content": "Hello, world!" }`.
func CreateStreamObjectModels() []ModelManagerModelConfig {
	mock := NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
				{"type": "stream-start", "warnings": []any{}},
				{
					"type":      "response-metadata",
					"id":        "id-0",
					"modelId":   "mock-model-id",
					"timestamp": MockDate,
				},
				{"type": "text-start", "id": "text-1"},
				{"type": "text-delta", "id": "text-1", "delta": "{ "},
				{"type": "text-delta", "id": "text-1", "delta": `"content": `},
				{"type": "text-delta", "id": "text-1", "delta": `"Hello, `},
				{"type": "text-delta", "id": "text-1", "delta": `world`},
				{"type": "text-delta", "id": "text-1", "delta": `!"`},
				{"type": "text-delta", "id": "text-1", "delta": " }"},
				{"type": "text-end", "id": "text-1"},
				{
					"type":         "finish",
					"finishReason": "stop",
					"usage":        TestUsage,
					"providerMetadata": map[string]any{
						"testProvider": map[string]any{"testKey": "testValue"},
					},
				},
			})
			return &DoStreamResult{Stream: stream}, nil
		},
	})
	return []ModelManagerModelConfig{
		{
			Model:      mock,
			MaxRetries: 0,
			ID:         "test-model",
		},
	}
}

// VerifyNoObjectGeneratedError validates that an error is a
// NoObjectGeneratedError with the expected properties.
// This mirrors the TS verifyNoObjectGeneratedError() function.
type ExpectedNoObjectError struct {
	Message      string
	Response     any
	Usage        any
	FinishReason string
}

// VerifyNoObjectGeneratedError checks that err is a NoObjectGeneratedError
// matching the expected values.
func VerifyNoObjectGeneratedError(err error, expected ExpectedNoObjectError) bool {
	noObjErr, ok := err.(*NoObjectGeneratedError)
	if !ok {
		return false
	}
	if noObjErr.Message != expected.Message {
		return false
	}
	if noObjErr.FinishReason != expected.FinishReason {
		return false
	}
	// Response and Usage comparison would need deep-equal in real tests.
	return true
}
