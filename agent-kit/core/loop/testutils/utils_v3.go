// Ported from: packages/core/src/loop/test-utils/utils-v3.ts
package testutils

import (
	"time"
)

// ---------------------------------------------------------------------------
// V3 Usage types
// ---------------------------------------------------------------------------

// UsageV3InputTokens holds V3 input token breakdown.
type UsageV3InputTokens struct {
	Total      int  `json:"total"`
	NoCache    int  `json:"noCache"`
	CacheRead  *int `json:"cacheRead,omitempty"`
	CacheWrite *int `json:"cacheWrite,omitempty"`
}

// UsageV3OutputTokens holds V3 output token breakdown.
type UsageV3OutputTokens struct {
	Total     int  `json:"total"`
	Text      int  `json:"text"`
	Reasoning *int `json:"reasoning,omitempty"`
}

// UsageV3 represents token usage in V3 format.
type UsageV3 struct {
	InputTokens  UsageV3InputTokens  `json:"inputTokens"`
	OutputTokens UsageV3OutputTokens `json:"outputTokens"`
}

// ---------------------------------------------------------------------------
// Test usage constants (V3)
// ---------------------------------------------------------------------------

// TestUsageV3 is the standard V3 test usage.
var TestUsageV3 = UsageV3{
	InputTokens: UsageV3InputTokens{
		Total:      3,
		NoCache:    3,
		CacheRead:  nil,
		CacheWrite: nil,
	},
	OutputTokens: UsageV3OutputTokens{
		Total:     10,
		Text:      10,
		Reasoning: nil,
	},
}

// TestUsageV3_2 is the second V3 test usage with cached/reasoning tokens.
var TestUsageV3_2 = UsageV3{
	InputTokens: UsageV3InputTokens{
		Total:     3,
		NoCache:   0,
		CacheRead: intPtr(3),
	},
	OutputTokens: UsageV3OutputTokens{
		Total:     10,
		Text:      0,
		Reasoning: intPtr(10),
	},
}

// ---------------------------------------------------------------------------
// DefaultSettingsV3
// ---------------------------------------------------------------------------

// DefaultSettingsV3 returns the default V3 test settings matching the TS defaultSettings().
func DefaultSettingsV3() DefaultSettingsResult {
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
// CreateTestModelsV3
// ---------------------------------------------------------------------------

// CreateTestModelsV3Options configures the V3 mock models created by CreateTestModelsV3.
type CreateTestModelsV3Options struct {
	Warnings []SharedV3Warning
	Stream   <-chan LanguageModelV3StreamPart
	Request  *RequestBody
	Response *ResponseHeaders
}

// CreateTestModelsV3 creates a slice of ModelManagerModelConfig with a V3 mock model.
// If no stream is provided, a default stream producing "Hello, world!" is used.
func CreateTestModelsV3(opts ...CreateTestModelsV3Options) []ModelManagerModelConfig {
	var opt CreateTestModelsV3Options
	if len(opts) > 0 {
		opt = opts[0]
	}

	var stream <-chan LanguageModelV3StreamPart
	if opt.Stream != nil {
		stream = opt.Stream
	} else {
		stream = DefaultV3Stream(opt.Warnings)
	}

	mock := NewMastraLanguageModelV3Mock(MastraLanguageModelV3MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResultV3, error) {
			return &DoStreamResultV3{
				Stream:   stream,
				Request:  opt.Request,
				Response: opt.Response,
				Warnings: opt.Warnings,
			}, nil
		},
		DoGenerate: func(options map[string]any) (*DoGenerateResultV3, error) {
			return &DoGenerateResultV3{
				Content: []map[string]any{
					{"type": "text", "text": "Hello, world!"},
				},
				FinishReason: FinishReasonV3{Unified: "stop", Raw: "stop"},
				Usage:        TestUsageV3,
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

// DefaultV3Stream creates the default V3 stream producing "Hello, world!".
func DefaultV3Stream(warnings []SharedV3Warning) <-chan LanguageModelV3StreamPart {
	parts := []LanguageModelV3StreamPart{
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
			"finishReason": FinishReasonV3{Unified: "stop", Raw: "stop"},
			"usage":        TestUsageV3,
			"providerMetadata": map[string]any{
				"testProvider": map[string]any{"testKey": "testValue"},
			},
		},
	}
	return ConvertArrayToReadableStream(parts)
}

// ---------------------------------------------------------------------------
// Pre-built V3 mock models
// ---------------------------------------------------------------------------

// ModelWithSourcesV3 is a pre-built V3 mock model that emits sources.
var ModelWithSourcesV3 = NewMastraLanguageModelV3Mock(MastraLanguageModelV3MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResultV3, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV3StreamPart{
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
				"finishReason": FinishReasonV3{Unified: "stop", Raw: "stop"},
				"usage":        TestUsageV3,
			},
		})
		return &DoStreamResultV3{Stream: stream}, nil
	},
	DoGenerate: func(options map[string]any) (*DoGenerateResultV3, error) {
		return &DoGenerateResultV3{
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
			FinishReason: FinishReasonV3{Unified: "stop", Raw: "stop"},
			Usage:        TestUsageV3,
			Warnings:     nil,
		}, nil
	},
})

// ModelWithFilesV3 is a pre-built V3 mock model that emits files.
var ModelWithFilesV3 = NewMastraLanguageModelV3Mock(MastraLanguageModelV3MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResultV3, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV3StreamPart{
			{"type": "file", "data": "Hello World", "mediaType": "text/plain"},
			{"type": "text-start", "id": "text-1"},
			{"type": "text-delta", "id": "text-1", "delta": "Hello!"},
			{"type": "text-end", "id": "text-1"},
			{"type": "file", "data": "QkFVRw==", "mediaType": "image/jpeg"},
			{
				"type":         "finish",
				"finishReason": FinishReasonV3{Unified: "stop", Raw: "stop"},
				"usage":        TestUsageV3,
			},
		})
		return &DoStreamResultV3{Stream: stream}, nil
	},
	DoGenerate: func(options map[string]any) (*DoGenerateResultV3, error) {
		return &DoGenerateResultV3{
			Content: []map[string]any{
				{"type": "file", "data": "Hello World", "mediaType": "text/plain"},
				{"type": "text", "text": "Hello!"},
				{"type": "file", "data": "QkFVRw==", "mediaType": "image/jpeg"},
			},
			FinishReason: FinishReasonV3{Unified: "stop", Raw: "stop"},
			Usage:        TestUsageV3,
			Warnings:     nil,
		}, nil
	},
})

// ModelWithReasoningV3 is a pre-built V3 mock model that emits reasoning.
var ModelWithReasoningV3 = NewMastraLanguageModelV3Mock(MastraLanguageModelV3MockConfig{
	DoStream: func(options map[string]any) (*DoStreamResultV3, error) {
		stream := ConvertArrayToReadableStream([]LanguageModelV3StreamPart{
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
				"providerMetadata": map[string]any{
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
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{
				"type": "reasoning-start",
				"id":   "4",
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{"type": "reasoning-delta", "id": "4", "delta": " I need to think about"},
			{"type": "reasoning-delta", "id": "4", "delta": " this problem carefully."},
			{
				"type": "reasoning-end",
				"id":   "4",
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"signature": "0987654321"},
				},
			},
			{
				"type": "reasoning-start",
				"id":   "5",
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"signature": "1234567890"},
				},
			},
			{"type": "reasoning-delta", "id": "5", "delta": " The best solution"},
			{"type": "reasoning-delta", "id": "5", "delta": " requires careful"},
			{"type": "reasoning-delta", "id": "5", "delta": " consideration of all factors."},
			{
				"type": "reasoning-end",
				"id":   "5",
				"providerMetadata": map[string]any{
					"testProvider": map[string]any{"signature": "0987654321"},
				},
			},
			{"type": "text-start", "id": "text-1"},
			{"type": "text-delta", "id": "text-1", "delta": "Hi"},
			{"type": "text-delta", "id": "text-1", "delta": " there!"},
			{"type": "text-end", "id": "text-1"},
			{
				"type":         "finish",
				"finishReason": FinishReasonV3{Unified: "stop", Raw: "stop"},
				"usage":        TestUsageV3,
			},
		})
		return &DoStreamResultV3{Stream: stream}, nil
	},
	DoGenerate: func(options map[string]any) (*DoGenerateResultV3, error) {
		return &DoGenerateResultV3{
			Content: []map[string]any{
				{
					"type": "reasoning",
					"text": "I will open the conversation with witty banter. Once the user has relaxed, I will pry for valuable information. I need to think about this problem carefully. The best solution requires careful consideration of all factors.",
				},
				{"type": "text", "text": "Hi there!"},
			},
			FinishReason: FinishReasonV3{Unified: "stop", Raw: "stop"},
			Usage:        TestUsageV3,
			Warnings:     nil,
			Response: &DoGenerateResponseMeta{
				ID:        "id-0",
				ModelID:   "mock-model-id",
				Timestamp: time.Unix(0, 0),
			},
		}, nil
	},
})
