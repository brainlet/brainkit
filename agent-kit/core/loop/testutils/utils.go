// Ported from: packages/core/src/loop/test-utils/utils.ts
package testutils

import (
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Stub types for unported packages
// ---------------------------------------------------------------------------

// MessageList is a stub for ../../agent/message-list.MessageList.
// TODO: import from agent package once ported.
type MessageList struct {
	mu       sync.Mutex
	messages []MessageEntry
}

// MessageEntry represents a single message in the list.
type MessageEntry struct {
	ID      string         `json:"id,omitempty"`
	Role    string         `json:"role"`
	Content []ContentPart  `json:"content"`
	Source  string         `json:"source,omitempty"` // "input", "memory", "response"
}

// ContentPart represents a content part within a message.
type ContentPart struct {
	Type             string         `json:"type"`
	Text             string         `json:"text,omitempty"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
	ProviderOptions  map[string]any `json:"providerOptions,omitempty"`
	Parts            []ContentPart  `json:"parts,omitempty"`
}

// NewMessageList creates a new empty MessageList.
func NewMessageList() *MessageList {
	return &MessageList{}
}

// Add appends messages to the list with the given source tag.
func (ml *MessageList) Add(msgs any, source string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	switch v := msgs.(type) {
	case []MessageEntry:
		for i := range v {
			v[i].Source = source
			ml.messages = append(ml.messages, v[i])
		}
	case MessageEntry:
		v.Source = source
		ml.messages = append(ml.messages, v)
	}
}

// GetResponseDB returns response messages (mimics messageList.get.response.db()).
func (ml *MessageList) GetResponseDB() []MessageEntry {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	var result []MessageEntry
	for _, m := range ml.messages {
		if m.Source == "response" {
			result = append(result, m)
		}
	}
	return result
}

// All returns all messages.
func (ml *MessageList) All() []MessageEntry {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	out := make([]MessageEntry, len(ml.messages))
	copy(out, ml.messages)
	return out
}

// ModelManagerModelConfig is a stub for ../../stream/types.ModelManagerModelConfig.
// TODO: import from stream package once ported.
type ModelManagerModelConfig struct {
	Model      any    `json:"model"`
	MaxRetries int    `json:"maxRetries"`
	ID         string `json:"id"`
}

// LanguageModelV2StreamPart is a stub for @ai-sdk/provider-v5.LanguageModelV2StreamPart.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V2/V5 types remain local stubs.
type LanguageModelV2StreamPart = map[string]any

// LanguageModelV2CallWarning is a stub for @ai-sdk/provider-v5.LanguageModelV2CallWarning.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V2/V5 types remain local stubs.
type LanguageModelV2CallWarning = map[string]any

// SharedV2ProviderMetadata is a stub for @ai-sdk/provider-v5.SharedV2ProviderMetadata.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V2/V5 types remain local stubs.
type SharedV2ProviderMetadata = map[string]any

// ---------------------------------------------------------------------------
// MockDate
// ---------------------------------------------------------------------------

// MockDate is the default mock date used across test utilities.
var MockDate = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// DefaultSettings
// ---------------------------------------------------------------------------

// DefaultSettingsResult holds the default settings returned by DefaultSettings().
type DefaultSettingsResult struct {
	Prompt                         string         `json:"prompt"`
	ExperimentalGenerateMessageID  func() string  `json:"-"`
	Internal                       InternalConfig `json:"-"`
	AgentID                        string         `json:"agentId"`
	OnError                        func(err error) `json:"-"`
}

// InternalConfig holds _internal config fields.
type InternalConfig struct {
	GenerateID  func() string    `json:"-"`
	CurrentDate func() time.Time `json:"-"`
	Now         func() int64     `json:"-"`
}

// DefaultSettings returns the default test settings matching the TS defaultSettings().
func DefaultSettings() DefaultSettingsResult {
	return DefaultSettingsResult{
		Prompt:                        "prompt",
		ExperimentalGenerateMessageID: MockID(MockIDOptions{Prefix: "msg"}),
		Internal: InternalConfig{
			GenerateID:  MockID(MockIDOptions{Prefix: "id"}),
			CurrentDate: func() time.Time { return time.Unix(0, 0) },
		},
		AgentID: "agent-id",
		OnError: func(err error) {},
	}
}

// ---------------------------------------------------------------------------
// Test Usage
// ---------------------------------------------------------------------------

// UsageV2 represents token usage in V2 format.
type UsageV2 struct {
	InputTokens      int  `json:"inputTokens"`
	OutputTokens     int  `json:"outputTokens"`
	TotalTokens      int  `json:"totalTokens"`
	ReasoningTokens  *int `json:"reasoningTokens,omitempty"`
	CachedInputTokens *int `json:"cachedInputTokens,omitempty"`
}

// TestUsage is the standard test usage (V2 format).
var TestUsage = UsageV2{
	InputTokens:      3,
	OutputTokens:     10,
	TotalTokens:      13,
	ReasoningTokens:  nil,
	CachedInputTokens: nil,
}

// TestUsage2 is the second test usage with cached/reasoning tokens (V2 format).
var TestUsage2 = UsageV2{
	InputTokens:      3,
	OutputTokens:     10,
	TotalTokens:      23,
	ReasoningTokens:  intPtr(10),
	CachedInputTokens: intPtr(3),
}

func intPtr(v int) *int { return &v }

// ---------------------------------------------------------------------------
// CreateTestModels
// ---------------------------------------------------------------------------

// CreateTestModelsOptions configures the mock models created by CreateTestModels.
type CreateTestModelsOptions struct {
	Warnings []LanguageModelV2CallWarning
	Stream   <-chan LanguageModelV2StreamPart
	Request  *RequestBody
	Response *ResponseHeaders
}

// RequestBody holds a request body string.
type RequestBody struct {
	Body string `json:"body"`
}

// ResponseHeaders holds response header key-value pairs.
type ResponseHeaders struct {
	Headers map[string]string `json:"headers"`
}

// CreateTestModels creates a slice of ModelManagerModelConfig with a V2 mock model.
// If no stream is provided, a default stream producing "Hello, world!" is used.
func CreateTestModels(opts ...CreateTestModelsOptions) []ModelManagerModelConfig {
	var opt CreateTestModelsOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	var stream <-chan LanguageModelV2StreamPart
	if opt.Stream != nil {
		stream = opt.Stream
	} else {
		stream = DefaultV2Stream(opt.Warnings)
	}

	mock := NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			return &DoStreamResult{
				Stream:   stream,
				Request:  opt.Request,
				Response: opt.Response,
				Warnings: opt.Warnings,
			}, nil
		},
		DoGenerate: func(options map[string]any) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Content: []map[string]any{
					{"type": "text", "text": "Hello, world!"},
				},
				FinishReason: "stop",
				Usage:        TestUsage,
				Warnings:     opt.Warnings,
				Request:      opt.Request,
				Response: &DoGenerateResponseMeta{
					ID:        "id-0",
					ModelID:   "mock-model-id",
					Timestamp: time.Unix(0, 0),
				},
			}, nil
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

// DefaultV2Stream creates the default V2 stream producing "Hello, world!".
func DefaultV2Stream(warnings []LanguageModelV2CallWarning) <-chan LanguageModelV2StreamPart {
	parts := []LanguageModelV2StreamPart{
		{"type": "stream-start", "warnings": warnings},
		{
			"type":      "response-metadata",
			"id":        "id-0",
			"modelId":   "mock-model-id",
			"timestamp": time.Unix(0, 0),
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
			"providerMetadata": map[string]any{
				"testProvider": map[string]any{"testKey": "testValue"},
			},
		},
	}
	return ConvertArrayToReadableStream(parts)
}

// ---------------------------------------------------------------------------
// Pre-built mock models
// ---------------------------------------------------------------------------

// ModelWithSources is a pre-built V2 mock model that emits sources.
var ModelWithSources = NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResult, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
			{
				"type":       "source",
				"sourceType": "url",
				"id":         "123",
				"url":        "https://example.com",
				"title":      "Example",
				"providerMetadata": map[string]any{
					"provider": map[string]any{"custom": "value"},
				},
			},
			{"type": "text-start", "id": "text-1"},
			{"type": "text-delta", "id": "text-1", "delta": "Hello!"},
			{"type": "text-end", "id": "text-1"},
			{
				"type":       "source",
				"sourceType": "url",
				"id":         "456",
				"url":        "https://example.com/2",
				"title":      "Example 2",
				"providerMetadata": map[string]any{
					"provider": map[string]any{"custom": "value2"},
				},
			},
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
				{
					"type":       "source",
					"sourceType": "url",
					"id":         "123",
					"url":        "https://example.com",
					"title":      "Example",
					"providerMetadata": map[string]any{
						"provider": map[string]any{"custom": "value"},
					},
				},
				{"type": "text", "text": "Hello!"},
				{
					"type":       "source",
					"sourceType": "url",
					"id":         "456",
					"url":        "https://example.com/2",
					"title":      "Example 2",
					"providerMetadata": map[string]any{
						"provider": map[string]any{"custom": "value2"},
					},
				},
			},
			FinishReason: "stop",
			Usage:        TestUsage,
			Warnings:     nil,
		}, nil
	},
})

// ModelWithDocumentSources is a pre-built V2 mock model that emits document sources.
var ModelWithDocumentSources = NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResult, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
			{
				"type":       "source",
				"sourceType": "document",
				"id":         "doc-123",
				"mediaType":  "application/pdf",
				"title":      "Document Example",
				"filename":   "example.pdf",
				"providerMetadata": map[string]any{
					"provider": map[string]any{"custom": "doc-value"},
				},
			},
			{"type": "text-start", "id": "text-1"},
			{"type": "text-delta", "id": "text-1", "delta": "Hello from document!"},
			{"type": "text-end", "id": "text-1"},
			{
				"type":       "source",
				"sourceType": "document",
				"id":         "doc-456",
				"mediaType":  "text/plain",
				"title":      "Text Document",
				"providerMetadata": map[string]any{
					"provider": map[string]any{"custom": "doc-value2"},
				},
			},
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
				{
					"type":       "source",
					"sourceType": "document",
					"id":         "doc-123",
					"mediaType":  "application/pdf",
					"title":      "Document Example",
					"filename":   "example.pdf",
					"providerMetadata": map[string]any{
						"provider": map[string]any{"custom": "doc-value"},
					},
				},
				{"type": "text", "text": "Hello from document!"},
				{
					"type":       "source",
					"sourceType": "document",
					"id":         "doc-456",
					"mediaType":  "text/plain",
					"title":      "Text Document",
					"providerMetadata": map[string]any{
						"provider": map[string]any{"custom": "doc-value2"},
					},
				},
			},
			FinishReason: "stop",
			Usage:        TestUsage,
			Warnings:     nil,
		}, nil
	},
})

// ModelWithFiles is a pre-built V2 mock model that emits files.
var ModelWithFiles = NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResult, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
			{"type": "file", "data": "Hello World", "mediaType": "text/plain"},
			{"type": "text-start", "id": "text-1"},
			{"type": "text-delta", "id": "text-1", "delta": "Hello!"},
			{"type": "text-end", "id": "text-1"},
			{"type": "file", "data": "QkFVRw==", "mediaType": "image/jpeg"},
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
				{"type": "file", "data": "Hello World", "mediaType": "text/plain"},
				{"type": "text", "text": "Hello!"},
				{"type": "file", "data": "QkFVRw==", "mediaType": "image/jpeg"},
			},
			FinishReason: "stop",
			Usage:        TestUsage,
			Warnings:     nil,
		}, nil
	},
})

// ModelWithReasoning is a pre-built V2 mock model that emits reasoning.
var ModelWithReasoning = NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResult, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
			{
				"type":      "response-metadata",
				"id":        "id-0",
				"modelId":   "mock-model-id",
				"timestamp": time.Unix(0, 0),
			},
			{"type": "reasoning-start", "id": "1"},
			{"type": "reasoning-delta", "id": "1", "delta": "I will open the conversation"},
			{"type": "reasoning-delta", "id": "1", "delta": " with witty banter."},
			{
				"type":  "reasoning-delta",
				"id":    "1",
				"delta": "",
				"providerMetadata": SharedV2ProviderMetadata{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{"type": "reasoning-end", "id": "1"},
			{
				"type": "reasoning-start",
				"id":   "2",
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"redactedData": "redacted-reasoning-data"},
				},
			},
			{"type": "reasoning-end", "id": "2"},
			{"type": "reasoning-start", "id": "3"},
			{"type": "reasoning-delta", "id": "3", "delta": " Once the user has relaxed,"},
			{"type": "reasoning-delta", "id": "3", "delta": " I will pry for valuable information."},
			{
				"type": "reasoning-end",
				"id":   "3",
				"providerMetadata": SharedV2ProviderMetadata{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{
				"type": "reasoning-start",
				"id":   "4",
				"providerMetadata": SharedV2ProviderMetadata{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{"type": "reasoning-delta", "id": "4", "delta": " I need to think about"},
			{"type": "reasoning-delta", "id": "4", "delta": " this problem carefully."},
			{
				"type": "reasoning-end",
				"id":   "4",
				"providerMetadata": SharedV2ProviderMetadata{
					"testProvider": map[string]any{"signature": "0987654321"},
				},
			},
			{
				"type": "reasoning-start",
				"id":   "5",
				"providerMetadata": SharedV2ProviderMetadata{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{"type": "reasoning-delta", "id": "5", "delta": " The best solution"},
			{"type": "reasoning-delta", "id": "5", "delta": " requires careful"},
			{"type": "reasoning-delta", "id": "5", "delta": " consideration of all factors."},
			{
				"type": "reasoning-end",
				"id":   "5",
				"providerMetadata": SharedV2ProviderMetadata{
					"testProvider": map[string]any{"signature": "0987654321"},
				},
			},
			{"type": "text-start", "id": "text-1"},
			{"type": "text-delta", "id": "text-1", "delta": "Hi"},
			{"type": "text-delta", "id": "text-1", "delta": " there!"},
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
				{
					"type": "reasoning",
					"text": "I will open the conversation with witty banter. Once the user has relaxed, I will pry for valuable information. I need to think about this problem carefully. The best solution requires careful consideration of all factors.",
				},
				{"type": "text", "text": "Hi there!"},
			},
			FinishReason: "stop",
			Usage:        TestUsage,
			Warnings:     nil,
			Response: &DoGenerateResponseMeta{
				ID:        "id-0",
				ModelID:   "mock-model-id",
				Timestamp: time.Unix(0, 0),
			},
		}, nil
	},
})

// ---------------------------------------------------------------------------
// CreateMessageListWithUserMessage
// ---------------------------------------------------------------------------

// CreateMessageListWithUserMessage creates a MessageList with a single user message.
func CreateMessageListWithUserMessage() *MessageList {
	ml := NewMessageList()
	ml.Add(MessageEntry{
		ID:   "msg-1",
		Role: "user",
		Content: []ContentPart{
			{Type: "text", Text: "test-input"},
		},
	}, "input")
	return ml
}

// ---------------------------------------------------------------------------
// Mock helpers (ported from @internal/ai-sdk-v5/test)
// ---------------------------------------------------------------------------

// MockIDOptions configures MockID behavior.
type MockIDOptions struct {
	Prefix string
}

// MockID returns a function that generates sequential IDs with the given prefix.
// E.g., MockID({Prefix: "id"}) returns "id-0", "id-1", "id-2", ...
func MockID(opts MockIDOptions) func() string {
	var mu sync.Mutex
	counter := 0
	return func() string {
		mu.Lock()
		defer mu.Unlock()
		id := opts.Prefix + "-" + itoa(counter)
		counter++
		return id
	}
}

// MockValues returns a function that cycles through the given values.
// Panics if called more times than values provided.
func MockValues[T any](values ...T) func() T {
	var mu sync.Mutex
	idx := 0
	return func() T {
		mu.Lock()
		defer mu.Unlock()
		if idx >= len(values) {
			// Return last value if exhausted (matches TS behavior)
			return values[len(values)-1]
		}
		v := values[idx]
		idx++
		return v
	}
}

// ConvertArrayToReadableStream converts a slice of stream parts into a channel.
// This is the Go equivalent of the TS convertArrayToReadableStream helper.
func ConvertArrayToReadableStream[T any](parts []T) <-chan T {
	ch := make(chan T, len(parts))
	for _, p := range parts {
		ch <- p
	}
	close(ch)
	return ch
}

// ConvertAsyncIterableToArray drains a channel into a slice.
// This is the Go equivalent of the TS convertAsyncIterableToArray helper.
func ConvertAsyncIterableToArray[T any](ch <-chan T) []T {
	var result []T
	for v := range ch {
		result = append(result, v)
	}
	return result
}

// itoa converts an int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
