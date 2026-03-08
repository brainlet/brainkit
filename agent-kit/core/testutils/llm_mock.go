// Package testutils provides test utilities for agent-kit, including mock
// LLM models and providers for testing agent behavior without real API calls.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts
package testutils

import (
	"encoding/json"
	"fmt"
	"strings"
)

// MockModelVersion specifies which AI SDK model version to mock.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — version?: 'v1' | 'v2'
type MockModelVersion string

const (
	// MockModelVersionV1 targets AI SDK v4 (LanguageModelV1).
	MockModelVersionV1 MockModelVersion = "v1"

	// MockModelVersionV2 targets AI SDK v5 (LanguageModelV2).
	MockModelVersionV2 MockModelVersion = "v2"
)

// CreateMockModelOptions holds the options for creating a mock model.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — createMockModel params
type CreateMockModelOptions struct {
	// ObjectGenerationMode controls how objects are serialized. "json" mode
	// wraps the text in JSON.stringify.
	//
	// Ported from: objectGenerationMode?: 'json'
	ObjectGenerationMode string

	// MockText is the text content the mock model will return.
	// Can be a plain string or a structured value that will be JSON-marshaled.
	//
	// Ported from: mockText: string | Record<string, any>
	MockText any

	// SpyGenerate is an optional callback invoked on each generate call,
	// receiving the generation props for inspection.
	//
	// Ported from: spyGenerate?: (props: any) => void
	SpyGenerate func(props any)

	// SpyStream is an optional callback invoked on each stream call,
	// receiving the stream props for inspection.
	//
	// Ported from: spyStream?: (props: any) => void
	SpyStream func(props any)

	// Version selects the model version to mock. Defaults to V2.
	//
	// Ported from: version?: 'v1' | 'v2'
	Version MockModelVersion
}

// MockGenerateResult is the result returned by a mock model's Generate method.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — doGenerate return values
type MockGenerateResult struct {
	// RawCall contains the raw prompt and settings (always nil/empty for mocks).
	RawCall MockRawCall `json:"rawCall"`

	// FinishReason indicates why generation stopped.
	FinishReason string `json:"finishReason"`

	// Usage contains token usage information.
	Usage MockUsage `json:"usage"`

	// Text is the generated text (v1 style).
	Text string `json:"text,omitempty"`

	// Content is the generated content array (v2 style).
	Content []MockContentBlock `json:"content,omitempty"`

	// Warnings holds any generation warnings (v2 style).
	Warnings []any `json:"warnings,omitempty"`
}

// MockRawCall represents the raw call data in mock responses.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — rawCall: { rawPrompt: null, rawSettings: {} }
type MockRawCall struct {
	RawPrompt   any            `json:"rawPrompt"`
	RawSettings map[string]any `json:"rawSettings"`
}

// MockUsage holds token usage data for mock models.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — usage objects
type MockUsage struct {
	// V1 fields
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`

	// V2 fields
	InputTokens  int `json:"inputTokens,omitempty"`
	OutputTokens int `json:"outputTokens,omitempty"`
	TotalTokens  int `json:"totalTokens,omitempty"`
}

// MockContentBlock represents a content block in v2 responses.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — content: [{ type: 'text', text: ... }]
type MockContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MockStreamChunk represents a single chunk in a mock stream.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — stream chunks
type MockStreamChunk struct {
	Type         string     `json:"type"`
	TextDelta    string     `json:"textDelta,omitempty"`
	Delta        string     `json:"delta,omitempty"`
	ID           string     `json:"id,omitempty"`
	FinishReason string     `json:"finishReason,omitempty"`
	Usage        *MockUsage `json:"usage,omitempty"`
	Warnings     []any      `json:"warnings,omitempty"`
}

// MockModel is a mock language model that returns pre-configured responses.
// It supports both v1 and v2 model interfaces.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — createMockModel return value
type MockModel struct {
	// Version is the model version this mock targets.
	Version MockModelVersion

	// text is the pre-configured response text.
	text string

	// objectGenerationMode controls JSON wrapping.
	objectGenerationMode string

	// spyGenerate is called on each generate invocation.
	spyGenerate func(props any)

	// spyStream is called on each stream invocation.
	spyStream func(props any)
}

// CreateMockModel creates a mock language model that returns pre-configured
// text responses for both generate and stream operations.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — createMockModel()
func CreateMockModel(opts CreateMockModelOptions) *MockModel {
	version := opts.Version
	if version == "" {
		version = MockModelVersionV2
	}

	// Resolve mockText to a string
	var text string
	switch v := opts.MockText.(type) {
	case string:
		text = v
	case nil:
		text = ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			text = fmt.Sprintf("%v", v)
		} else {
			text = string(b)
		}
	}

	// If objectGenerationMode is "json", wrap in JSON.stringify equivalent
	if opts.ObjectGenerationMode == "json" {
		b, err := json.Marshal(opts.MockText)
		if err == nil {
			text = string(b)
		}
	}

	return &MockModel{
		Version:              version,
		text:                 text,
		objectGenerationMode: opts.ObjectGenerationMode,
		spyGenerate:          opts.SpyGenerate,
		spyStream:            opts.SpyStream,
	}
}

// Generate performs a mock generation, returning pre-configured text.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — doGenerate
func (m *MockModel) Generate(props any) (*MockGenerateResult, error) {
	if m.spyGenerate != nil {
		m.spyGenerate(props)
	}

	if m.Version == MockModelVersionV1 {
		// V1 response format
		return &MockGenerateResult{
			RawCall:      MockRawCall{RawSettings: map[string]any{}},
			FinishReason: "stop",
			Usage: MockUsage{
				PromptTokens:     10,
				CompletionTokens: 20,
			},
			Text: m.text,
		}, nil
	}

	// V2 response format (default)
	return &MockGenerateResult{
		RawCall:      MockRawCall{RawSettings: map[string]any{}},
		FinishReason: "stop",
		Usage: MockUsage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
		Content: []MockContentBlock{
			{Type: "text", Text: m.text},
		},
		Warnings: []any{},
	}, nil
}

// StreamChunks returns the mock stream chunks for the pre-configured text.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — doStream
func (m *MockModel) StreamChunks(props any) ([]MockStreamChunk, error) {
	if m.spyStream != nil {
		m.spyStream(props)
	}

	if m.Version == MockModelVersionV1 {
		// V1 streaming format: split text into word-based text-delta chunks
		words := strings.Split(m.text, " ")
		chunks := make([]MockStreamChunk, 0, len(words)+1)
		for _, word := range words {
			chunks = append(chunks, MockStreamChunk{
				Type:      "text-delta",
				TextDelta: word + " ",
			})
		}
		// Finish chunk
		chunks = append(chunks, MockStreamChunk{
			Type:         "finish",
			FinishReason: "stop",
			Usage: &MockUsage{
				CompletionTokens: 10,
				PromptTokens:     3,
			},
		})
		return chunks, nil
	}

	// V2 streaming format
	return []MockStreamChunk{
		{Type: "stream-start", Warnings: []any{}},
		{Type: "response-metadata", ID: "id-0"},
		{Type: "text-start", ID: "text-1"},
		{Type: "text-delta", ID: "text-1", Delta: m.text},
		{Type: "text-end", ID: "text-1"},
		{
			Type:         "finish",
			FinishReason: "stop",
			Usage: &MockUsage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}, nil
}

// MockProviderOptions holds options for creating a MockProvider.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — MockProvider constructor params
type MockProviderOptions struct {
	// SpyGenerate is called on each generate invocation.
	SpyGenerate func(props any)

	// SpyStream is called on each stream invocation.
	SpyStream func(props any)

	// ObjectGenerationMode controls JSON wrapping. Use "json" for object mode.
	ObjectGenerationMode string

	// MockText is the text the mock will return. Defaults to "Hello, world!".
	MockText any
}

// MockProvider wraps a MockModel as a provider that can be used in place of
// a real LLM provider in tests. In TS this extends MastraLLMV1.
//
// Note: The TS MockProvider extends MastraLLMV1 and overrides stream/__streamObject.
// In Go, we provide the mock model directly since the LLM abstraction differs.
// Tests should use CreateMockModel directly or wrap in the appropriate interface.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — MockProvider
type MockProvider struct {
	// Model is the underlying mock model.
	Model *MockModel

	// options preserves the original construction options.
	options MockProviderOptions
}

// NewMockProvider creates a new MockProvider wrapping a v1 mock model.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — MockProvider constructor
func NewMockProvider(opts MockProviderOptions) *MockProvider {
	mockText := opts.MockText
	if mockText == nil {
		mockText = "Hello, world!"
	}

	model := CreateMockModel(CreateMockModelOptions{
		ObjectGenerationMode: opts.ObjectGenerationMode,
		MockText:             mockText,
		SpyGenerate:          opts.SpyGenerate,
		SpyStream:            opts.SpyStream,
		Version:              MockModelVersionV1,
	})

	return &MockProvider{
		Model:   model,
		options: opts,
	}
}

// Generate delegates to the underlying mock model's Generate method.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — inherited from MastraLLMV1
func (p *MockProvider) Generate(props any) (*MockGenerateResult, error) {
	return p.Model.Generate(props)
}

// StreamChunks delegates to the underlying mock model's StreamChunks method.
//
// Ported from: packages/core/src/test-utils/llm-mock.ts — overridden stream()
func (p *MockProvider) StreamChunks(props any) ([]MockStreamChunk, error) {
	return p.Model.StreamChunks(props)
}
