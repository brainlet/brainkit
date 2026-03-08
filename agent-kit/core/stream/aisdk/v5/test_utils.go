// Ported from: packages/core/src/stream/aisdk/v5/test-utils.ts
package v5

import (
	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// Test utilities for AI SDK v5 stream testing
// ---------------------------------------------------------------------------

// TestUsage provides standard usage metrics for tests.
var TestUsage = stream.LanguageModelUsage{
	InputTokens:    3,
	OutputTokens:   10,
	TotalTokens:    13,
	ReasoningTokens:    0,
	CachedInputTokens:  0,
}

// ---------------------------------------------------------------------------
// MockLanguageModelV2
// ---------------------------------------------------------------------------

// MockLanguageModelV2DoStreamFn is the function signature for a mocked doStream.
type MockLanguageModelV2DoStreamFn func() (*stream.LanguageModelV2StreamResult, error)

// MockLanguageModelV2 is a mock language model for testing.
// It implements the minimum interface needed to provide a stream.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 test types remain local stubs.
type MockLanguageModelV2 struct {
	doStream MockLanguageModelV2DoStreamFn
}

// MockLanguageModelV2Options configures a MockLanguageModelV2.
type MockLanguageModelV2Options struct {
	DoStream MockLanguageModelV2DoStreamFn
}

// NewMockLanguageModelV2 creates a new mock language model.
func NewMockLanguageModelV2(opts MockLanguageModelV2Options) *MockLanguageModelV2 {
	return &MockLanguageModelV2{
		doStream: opts.DoStream,
	}
}

// DoStream calls the mock's doStream function.
func (m *MockLanguageModelV2) DoStream() (*stream.LanguageModelV2StreamResult, error) {
	return m.doStream()
}

// ---------------------------------------------------------------------------
// ConvertArrayToReadableStream
// ---------------------------------------------------------------------------

// ConvertArrayToReadableStream converts a slice of LanguageModelV2StreamPart to a channel,
// mimicking the TS convertArrayToReadableStream helper.
func ConvertArrayToReadableStream(parts []stream.LanguageModelV2StreamPart) <-chan stream.LanguageModelV2StreamPart {
	ch := make(chan stream.LanguageModelV2StreamPart, len(parts))
	for _, part := range parts {
		ch <- part
	}
	close(ch)
	return ch
}

// ---------------------------------------------------------------------------
// CreateTestModel
// ---------------------------------------------------------------------------

// CreateTestModelOptions configures the test model.
type CreateTestModelOptions struct {
	Warnings []any
	Stream   <-chan stream.LanguageModelV2StreamPart
	Request  any
	Response any
}

// CreateTestModel creates a test model with a default stream containing
// representative chunks for testing: stream-start, response-metadata,
// reasoning, source, file, tool-call, tool-result, tool-input streaming,
// text, and finish events.
func CreateTestModel(opts *CreateTestModelOptions) *MockLanguageModelV2 {
	var warnings []any
	var streamCh <-chan stream.LanguageModelV2StreamPart
	var request any
	var response any

	if opts != nil {
		warnings = opts.Warnings
		streamCh = opts.Stream
		request = opts.Request
		response = opts.Response
	}

	if streamCh == nil {
		defaultParts := []stream.LanguageModelV2StreamPart{
			{Type: "stream-start", Data: map[string]any{"warnings": warnings}},
			{Type: "response-metadata", Data: map[string]any{"id": "id-0", "modelId": "mock-model-id"}},
			{Type: "reasoning-start", Data: map[string]any{"id": "reasoning-1"}},
			{Type: "reasoning-delta", Data: map[string]any{"id": "reasoning-1", "delta": "I need to think about this..."}},
			{Type: "reasoning-delta", Data: map[string]any{"id": "reasoning-1", "delta": " Let me process the request."}},
			{Type: "reasoning-end", Data: map[string]any{"id": "reasoning-1"}},
			{Type: "source", Data: map[string]any{
				"sourceType": "url",
				"id":         "source-1",
				"url":        "https://example.com/article",
				"title":      "Example Article",
			}},
			{Type: "file", Data: map[string]any{
				"mediaType": "image/png",
				"data":      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
			}},
			{Type: "tool-call", Data: map[string]any{
				"toolCallId":       "call-1",
				"toolName":         "get_weather",
				"input":            `{"location": "New York", "unit": "celsius"}`,
				"providerExecuted": false,
			}},
			{Type: "tool-result", Data: map[string]any{
				"toolCallId":       "call-1",
				"toolName":         "get_weather",
				"result":           map[string]any{"temperature": 22, "condition": "sunny", "humidity": 65},
				"isError":          false,
				"providerExecuted": false,
			}},
			{Type: "tool-input-start", Data: map[string]any{
				"id":               "input-1",
				"toolName":         "calculate_sum",
				"providerExecuted": false,
			}},
			{Type: "tool-input-delta", Data: map[string]any{"id": "input-1", "delta": `{"a": 5, `}},
			{Type: "tool-input-delta", Data: map[string]any{"id": "input-1", "delta": `"b": 10}`}},
			{Type: "tool-input-end", Data: map[string]any{"id": "input-1"}},
			{Type: "text-start", Data: map[string]any{"id": "text-1"}},
			{Type: "text-delta", Data: map[string]any{"id": "text-1", "delta": "Hello"}},
			{Type: "text-delta", Data: map[string]any{"id": "text-1", "delta": ", "}},
			{Type: "text-delta", Data: map[string]any{"id": "text-1", "delta": "world!"}},
			{Type: "text-end", Data: map[string]any{"id": "text-1"}},
			{Type: "finish", Data: map[string]any{
				"finishReason": "stop",
				"usage": map[string]any{
					"inputTokens":  3,
					"outputTokens": 10,
					"totalTokens":  13,
				},
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"testKey": "testValue"},
				},
			}},
		}
		streamCh = ConvertArrayToReadableStream(defaultParts)
	}

	// Build the result with proper type conversions
	req, _ := request.(stream.LLMStepResultRequest)
	resp, _ := response.(*stream.LLMStepResultResponse)
	var warns []stream.LanguageModelV2CallWarning
	for _, w := range warnings {
		if cw, ok := w.(stream.LanguageModelV2CallWarning); ok {
			warns = append(warns, cw)
		}
	}

	return NewMockLanguageModelV2(MockLanguageModelV2Options{
		DoStream: func() (*stream.LanguageModelV2StreamResult, error) {
			return &stream.LanguageModelV2StreamResult{
				Stream:   streamCh,
				Request:  req,
				Response: resp,
				Warnings: warns,
			}, nil
		},
	})
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
