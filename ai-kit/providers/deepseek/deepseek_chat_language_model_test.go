// Ported from: packages/deepseek/src/chat/deepseek-chat-language-model.test.ts
package deepseek

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- Fixture data ---

// deepseekTextFixture is the JSON response for a text completion.
var deepseekTextFixture = `{
  "id": "00f10ecd-60b3-4707-b5db-e4bcadf7aea1",
  "object": "chat.completion",
  "created": 1764656316,
  "model": "deepseek-chat",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello, how can I help you today?"
      },
      "logprobs": null,
      "finish_reason": "length"
    }
  ],
  "usage": {
    "prompt_tokens": 13,
    "completion_tokens": 300,
    "total_tokens": 313,
    "prompt_tokens_details": {
      "cached_tokens": 0
    },
    "prompt_cache_hit_tokens": 0,
    "prompt_cache_miss_tokens": 13
  },
  "system_fingerprint": "fp_eaab8d114b_prod0820_fp8_kvcache"
}`

// deepseekReasoningFixture is the JSON response for a reasoning completion.
var deepseekReasoningFixture = `{
  "id": "945bb10c-9bf3-47ff-a2a2-43bbe9705c72",
  "object": "chat.completion",
  "created": 1764660903,
  "model": "deepseek-reasoner",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "The word \"strawberry\" contains three instances of the letter \"r\".",
        "reasoning_content": "Let me count the r's in strawberry: s-t-r-a-w-b-e-r-r-y. Three r's total."
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 18,
    "completion_tokens": 345,
    "total_tokens": 363,
    "prompt_tokens_details": {
      "cached_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 315
    },
    "prompt_cache_hit_tokens": 0,
    "prompt_cache_miss_tokens": 18
  },
  "system_fingerprint": "fp_eaab8d114b_prod0820_fp8_kvcache"
}`

// deepseekToolCallFixture is the JSON response for a tool call completion.
var deepseekToolCallFixture = `{
  "id": "7a630f5b-b7e6-4878-82f8-d77db164d42b",
  "object": "chat.completion",
  "created": 1764665845,
  "model": "deepseek-reasoner",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "",
        "reasoning_content": "The user is asking for the weather. Let me call the weather function.",
        "tool_calls": [
          {
            "index": 0,
            "id": "call_00_9V0vrf86Pc9aelHCJMZqnJBo",
            "type": "function",
            "function": {
              "name": "weather",
              "arguments": "{\"location\": \"San Francisco\"}"
            }
          }
        ]
      },
      "logprobs": null,
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 339,
    "completion_tokens": 92,
    "total_tokens": 431,
    "prompt_tokens_details": {
      "cached_tokens": 320
    },
    "completion_tokens_details": {
      "reasoning_tokens": 48
    },
    "prompt_cache_hit_tokens": 320,
    "prompt_cache_miss_tokens": 19
  },
  "system_fingerprint": "fp_eaab8d114b_prod0820_fp8_kvcache"
}`

// deepseekJSONFixture is the JSON response for a JSON format completion.
var deepseekJSONFixture = `{
  "id": "f03bc170-b375-4561-9685-35182c8152c5",
  "object": "chat.completion",
  "created": 1764681341,
  "model": "deepseek-reasoner",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "{\"location\": \"San Francisco\", \"condition\": \"cloudy\", \"temperature\": 7}",
        "reasoning_content": "The tool returned weather data. I should output the JSON directly."
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 495,
    "completion_tokens": 144,
    "total_tokens": 639,
    "prompt_tokens_details": {
      "cached_tokens": 320
    },
    "completion_tokens_details": {
      "reasoning_tokens": 118
    },
    "prompt_cache_hit_tokens": 320,
    "prompt_cache_miss_tokens": 175
  },
  "system_fingerprint": "fp_eaab8d114b_prod0820_fp8_kvcache"
}`

// --- Streaming fixture data ---

// deepseekTextChunks simulates streaming text chunks.
var deepseekTextChunks = []string{
	`{"id":"f6117a0b-129d-46fa-b239-78f01c2c5df9","object":"chat.completion.chunk","created":1764657993,"model":"deepseek-chat","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"role":"assistant","content":""},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"f6117a0b-129d-46fa-b239-78f01c2c5df9","object":"chat.completion.chunk","created":1764657993,"model":"deepseek-chat","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"content":"Hello"},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"f6117a0b-129d-46fa-b239-78f01c2c5df9","object":"chat.completion.chunk","created":1764657993,"model":"deepseek-chat","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"content":", world!"},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"f6117a0b-129d-46fa-b239-78f01c2c5df9","object":"chat.completion.chunk","created":1764657993,"model":"deepseek-chat","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_cache_hit_tokens":0,"prompt_cache_miss_tokens":10}}`,
}

// deepseekReasoningChunks simulates streaming reasoning chunks.
var deepseekReasoningChunks = []string{
	`{"id":"cac7192e-e619-40c6-96b0-ed4276bc03ac","object":"chat.completion.chunk","created":1764661832,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"role":"assistant","content":null,"reasoning_content":""},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cac7192e-e619-40c6-96b0-ed4276bc03ac","object":"chat.completion.chunk","created":1764661832,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"content":null,"reasoning_content":"Let me"},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cac7192e-e619-40c6-96b0-ed4276bc03ac","object":"chat.completion.chunk","created":1764661832,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"content":null,"reasoning_content":" think."},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cac7192e-e619-40c6-96b0-ed4276bc03ac","object":"chat.completion.chunk","created":1764661832,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"content":"The answer is 3."},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cac7192e-e619-40c6-96b0-ed4276bc03ac","object":"chat.completion.chunk","created":1764661832,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"stop"}],"usage":{"prompt_tokens":18,"completion_tokens":345,"total_tokens":363,"prompt_cache_hit_tokens":0,"prompt_cache_miss_tokens":18,"completion_tokens_details":{"reasoning_tokens":315}}}`,
}

// deepseekToolCallChunks simulates streaming tool call chunks.
var deepseekToolCallChunks = []string{
	`{"id":"cca85624-4056-401f-b220-d77601d1f70d","object":"chat.completion.chunk","created":1764664568,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"role":"assistant","content":null,"reasoning_content":""},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cca85624-4056-401f-b220-d77601d1f70d","object":"chat.completion.chunk","created":1764664568,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"content":null,"reasoning_content":"I need to call weather."},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cca85624-4056-401f-b220-d77601d1f70d","object":"chat.completion.chunk","created":1764664568,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_00_ioIn7yN9p1ZOMNpDLwd4MgAF","type":"function","function":{"name":"weather","arguments":""}}]},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cca85624-4056-401f-b220-d77601d1f70d","object":"chat.completion.chunk","created":1764664568,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"location\""}}]},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cca85624-4056-401f-b220-d77601d1f70d","object":"chat.completion.chunk","created":1764664568,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":": \"San Francisco\"}"}}]},"logprobs":null,"finish_reason":null}],"usage":null}`,
	`{"id":"cca85624-4056-401f-b220-d77601d1f70d","object":"chat.completion.chunk","created":1764664568,"model":"deepseek-reasoner","system_fingerprint":"fp_test","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":339,"completion_tokens":92,"total_tokens":431,"prompt_cache_hit_tokens":320,"prompt_cache_miss_tokens":19,"completion_tokens_details":{"reasoning_tokens":48}}}`,
}

// --- Test helper types ---

// capturedRequest holds the request body and headers from the mock server.
type capturedRequest struct {
	body    map[string]any
	headers http.Header
}

// createMockServer creates a test server that returns the given JSON response and captures requests.
func createMockServer(t *testing.T, responseJSON string) (*httptest.Server, *[]capturedRequest) {
	t.Helper()
	var captured []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(500)
			return
		}
		var bodyMap map[string]any
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
			w.WriteHeader(500)
			return
		}
		captured = append(captured, capturedRequest{body: bodyMap, headers: r.Header})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(responseJSON))
	}))

	return server, &captured
}

// createStreamingMockServer creates a test server that returns SSE streaming chunks.
func createStreamingMockServer(t *testing.T, chunks []string) (*httptest.Server, *[]capturedRequest) {
	t.Helper()
	var captured []capturedRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(500)
			return
		}
		var bodyMap map[string]any
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			t.Errorf("failed to unmarshal request body: %v", err)
			w.WriteHeader(500)
			return
		}
		captured = append(captured, capturedRequest{body: bodyMap, headers: r.Header})

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(200)

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("response writer does not support flushing")
			return
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))

	return server, &captured
}

// createProviderWithServer creates a DeepSeek provider pointing to the test server.
func createProviderWithServer(serverURL string) *Provider {
	apiKey := "test-api-key"
	return NewProvider(ProviderSettings{
		APIKey:  &apiKey,
		BaseURL: &serverURL,
	})
}

// testPrompt is the standard test prompt.
var testPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

// --- Tests ---

func TestDeepSeekChatLanguageModel_DoGenerate(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		t.Run("should send correct request body", func(t *testing.T) {
			server, captured := createMockServer(t, deepseekTextFixture)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			temp := 0.5
			topP := 0.3
			_, err := provider.ChatModel("deepseek-chat").DoGenerate(languagemodel.CallOptions{
				Ctx: context.Background(),
				Prompt: languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "You are a helpful assistant."},
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: "Hello"},
						},
					},
				},
				Temperature: &temp,
				TopP:        &topP,
			})
			if err != nil {
				t.Fatalf("DoGenerate failed: %v", err)
			}

			if len(*captured) != 1 {
				t.Fatalf("expected 1 request, got %d", len(*captured))
			}
			body := (*captured)[0].body

			if body["model"] != "deepseek-chat" {
				t.Errorf("expected model 'deepseek-chat', got %v", body["model"])
			}
			if body["temperature"] != 0.5 {
				t.Errorf("expected temperature 0.5, got %v", body["temperature"])
			}
			if body["top_p"] != 0.3 {
				t.Errorf("expected top_p 0.3, got %v", body["top_p"])
			}

			messages, ok := body["messages"].([]any)
			if !ok {
				t.Fatalf("expected messages to be array, got %T", body["messages"])
			}
			if len(messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(messages))
			}

			systemMsg, ok := messages[0].(map[string]any)
			if !ok {
				t.Fatalf("expected message to be map, got %T", messages[0])
			}
			if systemMsg["role"] != "system" {
				t.Errorf("expected role 'system', got %v", systemMsg["role"])
			}
			if systemMsg["content"] != "You are a helpful assistant." {
				t.Errorf("expected system content, got %v", systemMsg["content"])
			}

			userMsg, ok := messages[1].(map[string]any)
			if !ok {
				t.Fatalf("expected message to be map, got %T", messages[1])
			}
			if userMsg["role"] != "user" {
				t.Errorf("expected role 'user', got %v", userMsg["role"])
			}
			if userMsg["content"] != "Hello" {
				t.Errorf("expected content 'Hello', got %v", userMsg["content"])
			}
		})

		t.Run("should extract text content", func(t *testing.T) {
			server, _ := createMockServer(t, deepseekTextFixture)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			result, err := provider.ChatModel("deepseek-chat").DoGenerate(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
			})
			if err != nil {
				t.Fatalf("DoGenerate failed: %v", err)
			}

			// Should have text content
			var textContent string
			for _, c := range result.Content {
				if tc, ok := c.(languagemodel.Text); ok {
					textContent += tc.Text
				}
			}
			if textContent == "" {
				t.Error("expected non-empty text content")
			}

			// Check finish reason
			if result.FinishReason.Unified != languagemodel.FinishReasonLength {
				t.Errorf("expected finish reason 'length', got %q", result.FinishReason.Unified)
			}

			// Check usage
			if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 13 {
				t.Errorf("expected 13 input tokens, got %v", result.Usage.InputTokens.Total)
			}
			if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 300 {
				t.Errorf("expected 300 output tokens, got %v", result.Usage.OutputTokens.Total)
			}
		})
	})

	t.Run("reasoning", func(t *testing.T) {
		t.Run("should send correct request body", func(t *testing.T) {
			server, captured := createMockServer(t, deepseekReasoningFixture)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			_, err := provider.ChatModel("deepseek-reasoner").DoGenerate(languagemodel.CallOptions{
				Ctx: context.Background(),
				Prompt: languagemodel.Prompt{
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: `How many "r"s are in the word "strawberry"?`},
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("DoGenerate failed: %v", err)
			}

			if len(*captured) != 1 {
				t.Fatalf("expected 1 request, got %d", len(*captured))
			}
			body := (*captured)[0].body

			if body["model"] != "deepseek-reasoner" {
				t.Errorf("expected model 'deepseek-reasoner', got %v", body["model"])
			}

			messages, ok := body["messages"].([]any)
			if !ok {
				t.Fatalf("expected messages to be array, got %T", body["messages"])
			}
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
		})

		t.Run("should extract text content", func(t *testing.T) {
			server, _ := createMockServer(t, deepseekReasoningFixture)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			result, err := provider.ChatModel("deepseek-chat").DoGenerate(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
			})
			if err != nil {
				t.Fatalf("DoGenerate failed: %v", err)
			}

			// Should have reasoning content and text content
			var hasReasoning, hasText bool
			for _, c := range result.Content {
				switch c.(type) {
				case languagemodel.Reasoning:
					hasReasoning = true
				case languagemodel.Text:
					hasText = true
				}
			}
			if !hasReasoning {
				t.Error("expected reasoning content in response")
			}
			if !hasText {
				t.Error("expected text content in response")
			}

			// Check finish reason
			if result.FinishReason.Unified != languagemodel.FinishReasonStop {
				t.Errorf("expected finish reason 'stop', got %q", result.FinishReason.Unified)
			}
		})
	})

	t.Run("tool call", func(t *testing.T) {
		t.Run("should send correct request body", func(t *testing.T) {
			server, captured := createMockServer(t, deepseekToolCallFixture)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			_, err := provider.ChatModel("deepseek-reasoner").DoGenerate(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
				Tools: []languagemodel.Tool{
					languagemodel.FunctionTool{
						Name: "weather",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location": map[string]any{"type": "string"},
							},
							"required":             []any{"location"},
							"additionalProperties": false,
							"$schema":              "http://json-schema.org/draft-07/schema#",
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("DoGenerate failed: %v", err)
			}

			if len(*captured) != 1 {
				t.Fatalf("expected 1 request, got %d", len(*captured))
			}
			body := (*captured)[0].body

			if body["model"] != "deepseek-reasoner" {
				t.Errorf("expected model 'deepseek-reasoner', got %v", body["model"])
			}

			// Check tools
			tools, ok := body["tools"].([]any)
			if !ok {
				t.Fatalf("expected tools to be array, got %T", body["tools"])
			}
			if len(tools) != 1 {
				t.Fatalf("expected 1 tool, got %d", len(tools))
			}
			tool, ok := tools[0].(map[string]any)
			if !ok {
				t.Fatalf("expected tool to be map, got %T", tools[0])
			}
			if tool["type"] != "function" {
				t.Errorf("expected tool type 'function', got %v", tool["type"])
			}
			fn, ok := tool["function"].(map[string]any)
			if !ok {
				t.Fatalf("expected function to be map, got %T", tool["function"])
			}
			if fn["name"] != "weather" {
				t.Errorf("expected function name 'weather', got %v", fn["name"])
			}
		})

		t.Run("json response format", func(t *testing.T) {
			t.Run("should send correct request body without schema", func(t *testing.T) {
				server, captured := createMockServer(t, deepseekJSONFixture)
				defer server.Close()
				provider := createProviderWithServer(server.URL)

				thinkingType := "enabled"
				_, err := provider.ChatModel("deepseek-reasoner").DoGenerate(languagemodel.CallOptions{
					Ctx:            context.Background(),
					Prompt:         testPrompt,
					ResponseFormat: languagemodel.ResponseFormatJSON{},
					Tools: []languagemodel.Tool{
						languagemodel.FunctionTool{
							Name: "weather",
							InputSchema: map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location": map[string]any{"type": "string"},
								},
								"required":             []any{"location"},
								"additionalProperties": false,
								"$schema":              "http://json-schema.org/draft-07/schema#",
							},
						},
					},
					ProviderOptions: shared.ProviderOptions{
						"deepseek": map[string]any{
							"thinking": map[string]any{
								"type": thinkingType,
							},
						},
					},
				})
				if err != nil {
					t.Fatalf("DoGenerate failed: %v", err)
				}

				if len(*captured) != 1 {
					t.Fatalf("expected 1 request, got %d", len(*captured))
				}
				body := (*captured)[0].body

				// Check response_format
				respFormat, ok := body["response_format"].(map[string]any)
				if !ok {
					t.Fatalf("expected response_format to be map, got %T", body["response_format"])
				}
				if respFormat["type"] != "json_object" {
					t.Errorf("expected response_format type 'json_object', got %v", respFormat["type"])
				}

				// Check messages - should have system message with "Return JSON."
				messages, ok := body["messages"].([]any)
				if !ok {
					t.Fatalf("expected messages to be array, got %T", body["messages"])
				}
				if len(messages) < 2 {
					t.Fatalf("expected at least 2 messages, got %d", len(messages))
				}
				systemMsg, ok := messages[0].(map[string]any)
				if !ok {
					t.Fatalf("expected message to be map, got %T", messages[0])
				}
				if systemMsg["role"] != "system" {
					t.Errorf("expected role 'system', got %v", systemMsg["role"])
				}
				if systemMsg["content"] != "Return JSON." {
					t.Errorf("expected content 'Return JSON.', got %v", systemMsg["content"])
				}
			})

			t.Run("should send correct request body with schema", func(t *testing.T) {
				server, captured := createMockServer(t, deepseekJSONFixture)
				defer server.Close()
				provider := createProviderWithServer(server.URL)

				thinkingType := "enabled"
				schema := map[string]any{
					"type": "object",
					"properties": map[string]any{
						"elements": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location":    map[string]any{"type": "string"},
									"temperature": map[string]any{"type": "number"},
									"condition":   map[string]any{"type": "string"},
								},
								"required":             []any{"location", "temperature", "condition"},
								"additionalProperties": false,
							},
						},
					},
					"required":             []any{"elements"},
					"additionalProperties": false,
					"$schema":              "http://json-schema.org/draft-07/schema#",
				}

				_, err := provider.ChatModel("deepseek-reasoner").DoGenerate(languagemodel.CallOptions{
					Ctx:    context.Background(),
					Prompt: testPrompt,
					ResponseFormat: languagemodel.ResponseFormatJSON{
						Schema: schema,
					},
					Tools: []languagemodel.Tool{
						languagemodel.FunctionTool{
							Name: "weather",
							InputSchema: map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location": map[string]any{"type": "string"},
								},
								"required":             []any{"location"},
								"additionalProperties": false,
								"$schema":              "http://json-schema.org/draft-07/schema#",
							},
						},
					},
					ProviderOptions: shared.ProviderOptions{
						"deepseek": map[string]any{
							"thinking": map[string]any{
								"type": thinkingType,
							},
						},
					},
				})
				if err != nil {
					t.Fatalf("DoGenerate failed: %v", err)
				}

				if len(*captured) != 1 {
					t.Fatalf("expected 1 request, got %d", len(*captured))
				}
				body := (*captured)[0].body

				// Check messages - should have system message with schema
				messages, ok := body["messages"].([]any)
				if !ok {
					t.Fatalf("expected messages to be array, got %T", body["messages"])
				}
				systemMsg, ok := messages[0].(map[string]any)
				if !ok {
					t.Fatalf("expected message to be map, got %T", messages[0])
				}
				content, ok := systemMsg["content"].(string)
				if !ok {
					t.Fatalf("expected content to be string, got %T", systemMsg["content"])
				}
				if !strings.HasPrefix(content, "Return JSON that conforms to the following schema: ") {
					t.Errorf("expected system message to start with schema instruction, got %q", content)
				}
			})

			t.Run("should extract text content", func(t *testing.T) {
				server, _ := createMockServer(t, deepseekJSONFixture)
				defer server.Close()
				provider := createProviderWithServer(server.URL)

				thinkingType := "enabled"
				result, err := provider.ChatModel("deepseek-reasoner").DoGenerate(languagemodel.CallOptions{
					Ctx:            context.Background(),
					Prompt:         testPrompt,
					ResponseFormat: languagemodel.ResponseFormatJSON{},
					Tools: []languagemodel.Tool{
						languagemodel.FunctionTool{
							Name: "weather",
							InputSchema: map[string]any{
								"type": "object",
								"properties": map[string]any{
									"location": map[string]any{"type": "string"},
								},
								"required":             []any{"location"},
								"additionalProperties": false,
								"$schema":              "http://json-schema.org/draft-07/schema#",
							},
						},
					},
					ProviderOptions: shared.ProviderOptions{
						"deepseek": map[string]any{
							"thinking": map[string]any{
								"type": thinkingType,
							},
						},
					},
				})
				if err != nil {
					t.Fatalf("DoGenerate failed: %v", err)
				}

				// Should have text content (JSON response)
				var textContent string
				for _, c := range result.Content {
					if tc, ok := c.(languagemodel.Text); ok {
						textContent += tc.Text
					}
				}
				if textContent == "" {
					t.Error("expected non-empty text content")
				}

				// Verify it's valid JSON
				var parsed map[string]any
				if err := json.Unmarshal([]byte(textContent), &parsed); err != nil {
					t.Errorf("expected valid JSON text content, got error: %v", err)
				}
			})
		})

		t.Run("should extract tool call content", func(t *testing.T) {
			server, _ := createMockServer(t, deepseekToolCallFixture)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			thinkingType := "enabled"
			result, err := provider.ChatModel("deepseek-reasoner").DoGenerate(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
				Tools: []languagemodel.Tool{
					languagemodel.FunctionTool{
						Name: "weather",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location": map[string]any{"type": "string"},
							},
							"required":             []any{"location"},
							"additionalProperties": false,
							"$schema":              "http://json-schema.org/draft-07/schema#",
						},
					},
				},
				ProviderOptions: shared.ProviderOptions{
					"deepseek": map[string]any{
						"thinking": map[string]any{
							"type": thinkingType,
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("DoGenerate failed: %v", err)
			}

			// Should have reasoning content and tool call
			var hasReasoning, hasToolCall bool
			for _, c := range result.Content {
				switch tc := c.(type) {
				case languagemodel.Reasoning:
					hasReasoning = true
				case languagemodel.ToolCall:
					hasToolCall = true
					if tc.ToolName != "weather" {
						t.Errorf("expected tool name 'weather', got %q", tc.ToolName)
					}
					if tc.ToolCallID != "call_00_9V0vrf86Pc9aelHCJMZqnJBo" {
						t.Errorf("expected tool call ID 'call_00_9V0vrf86Pc9aelHCJMZqnJBo', got %q", tc.ToolCallID)
					}
					// Verify the input is valid JSON containing location
					var inputMap map[string]any
					if err := json.Unmarshal([]byte(tc.Input), &inputMap); err != nil {
						t.Errorf("expected valid JSON input, got error: %v", err)
					}
					if inputMap["location"] != "San Francisco" {
						t.Errorf("expected location 'San Francisco', got %v", inputMap["location"])
					}
				}
			}
			if !hasReasoning {
				t.Error("expected reasoning content in response")
			}
			if !hasToolCall {
				t.Error("expected tool call content in response")
			}

			// Check finish reason
			if result.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
				t.Errorf("expected finish reason 'tool-calls', got %q", result.FinishReason.Unified)
			}
		})
	})
}

func TestDeepSeekChatLanguageModel_DoStream(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		t.Run("should send model id, settings, and input", func(t *testing.T) {
			server, captured := createStreamingMockServer(t, deepseekTextChunks)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			temp := 0.5
			topP := 0.3
			_, err := provider.ChatModel("deepseek-chat").DoStream(languagemodel.CallOptions{
				Ctx: context.Background(),
				Prompt: languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "You are a helpful assistant."},
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: "Hello"},
						},
					},
				},
				Temperature: &temp,
				TopP:        &topP,
			})
			if err != nil {
				t.Fatalf("DoStream failed: %v", err)
			}

			if len(*captured) != 1 {
				t.Fatalf("expected 1 request, got %d", len(*captured))
			}
			body := (*captured)[0].body

			if body["model"] != "deepseek-chat" {
				t.Errorf("expected model 'deepseek-chat', got %v", body["model"])
			}
			if body["stream"] != true {
				t.Errorf("expected stream true, got %v", body["stream"])
			}
			streamOpts, ok := body["stream_options"].(map[string]any)
			if !ok {
				t.Fatalf("expected stream_options to be map, got %T", body["stream_options"])
			}
			if streamOpts["include_usage"] != true {
				t.Errorf("expected include_usage true, got %v", streamOpts["include_usage"])
			}
			if body["temperature"] != 0.5 {
				t.Errorf("expected temperature 0.5, got %v", body["temperature"])
			}
			if body["top_p"] != 0.3 {
				t.Errorf("expected top_p 0.3, got %v", body["top_p"])
			}

			messages, ok := body["messages"].([]any)
			if !ok {
				t.Fatalf("expected messages to be array, got %T", body["messages"])
			}
			if len(messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(messages))
			}
		})

		t.Run("should stream text", func(t *testing.T) {
			server, _ := createStreamingMockServer(t, deepseekTextChunks)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			result, err := provider.ChatModel("deepseek-chat").DoStream(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
			})
			if err != nil {
				t.Fatalf("DoStream failed: %v", err)
			}

			// Collect all stream parts
			var parts []languagemodel.StreamPart
			for part := range result.Stream {
				parts = append(parts, part)
			}

			// Verify stream starts
			if len(parts) == 0 {
				t.Fatal("expected stream parts")
			}
			if _, ok := parts[0].(languagemodel.StreamPartStreamStart); !ok {
				t.Errorf("expected first part to be StreamPartStreamStart, got %T", parts[0])
			}

			// Collect text deltas
			var textDeltas []string
			for _, part := range parts {
				if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
					textDeltas = append(textDeltas, delta.Delta)
				}
			}
			fullText := strings.Join(textDeltas, "")
			if fullText != "Hello, world!" {
				t.Errorf("expected text 'Hello, world!', got %q", fullText)
			}

			// Check finish part
			var finishPart *languagemodel.StreamPartFinish
			for _, part := range parts {
				if fp, ok := part.(languagemodel.StreamPartFinish); ok {
					finishPart = &fp
				}
			}
			if finishPart == nil {
				t.Fatal("expected finish part in stream")
			}
			if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
				t.Errorf("expected finish reason 'stop', got %q", finishPart.FinishReason.Unified)
			}
		})
	})

	t.Run("reasoning", func(t *testing.T) {
		t.Run("should stream reasoning", func(t *testing.T) {
			server, _ := createStreamingMockServer(t, deepseekReasoningChunks)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			result, err := provider.ChatModel("deepseek-reasoning").DoStream(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
			})
			if err != nil {
				t.Fatalf("DoStream failed: %v", err)
			}

			// Collect all stream parts
			var parts []languagemodel.StreamPart
			for part := range result.Stream {
				parts = append(parts, part)
			}

			// Check for reasoning start
			var hasReasoningStart, hasReasoningEnd bool
			var reasoningDeltas []string
			var textDeltas []string
			for _, part := range parts {
				switch p := part.(type) {
				case languagemodel.StreamPartReasoningStart:
					hasReasoningStart = true
				case languagemodel.StreamPartReasoningEnd:
					hasReasoningEnd = true
				case languagemodel.StreamPartReasoningDelta:
					reasoningDeltas = append(reasoningDeltas, p.Delta)
				case languagemodel.StreamPartTextDelta:
					textDeltas = append(textDeltas, p.Delta)
				}
			}

			if !hasReasoningStart {
				t.Error("expected reasoning start in stream")
			}
			if !hasReasoningEnd {
				t.Error("expected reasoning end in stream")
			}
			fullReasoning := strings.Join(reasoningDeltas, "")
			if fullReasoning != "Let me think." {
				t.Errorf("expected reasoning 'Let me think.', got %q", fullReasoning)
			}
			fullText := strings.Join(textDeltas, "")
			if fullText != "The answer is 3." {
				t.Errorf("expected text 'The answer is 3.', got %q", fullText)
			}
		})
	})

	t.Run("tool call", func(t *testing.T) {
		t.Run("should stream tool call", func(t *testing.T) {
			server, _ := createStreamingMockServer(t, deepseekToolCallChunks)
			defer server.Close()
			provider := createProviderWithServer(server.URL)

			thinkingType := "enabled"
			result, err := provider.ChatModel("deepseek-reasoner").DoStream(languagemodel.CallOptions{
				Ctx:    context.Background(),
				Prompt: testPrompt,
				Tools: []languagemodel.Tool{
					languagemodel.FunctionTool{
						Name: "weather",
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"location": map[string]any{"type": "string"},
							},
							"required":             []any{"location"},
							"additionalProperties": false,
							"$schema":              "http://json-schema.org/draft-07/schema#",
						},
					},
				},
				ProviderOptions: shared.ProviderOptions{
					"deepseek": map[string]any{
						"thinking": map[string]any{
							"type": thinkingType,
						},
					},
				},
			})
			if err != nil {
				t.Fatalf("DoStream failed: %v", err)
			}

			// Collect all stream parts
			var parts []languagemodel.StreamPart
			for part := range result.Stream {
				parts = append(parts, part)
			}

			// Check for reasoning
			var hasReasoningStart bool
			var reasoningDeltas []string
			for _, part := range parts {
				switch p := part.(type) {
				case languagemodel.StreamPartReasoningStart:
					hasReasoningStart = true
				case languagemodel.StreamPartReasoningDelta:
					reasoningDeltas = append(reasoningDeltas, p.Delta)
				}
			}
			if !hasReasoningStart {
				t.Error("expected reasoning start in stream")
			}
			if len(reasoningDeltas) == 0 {
				t.Error("expected reasoning deltas in stream")
			}

			// Check for tool call
			var toolCalls []languagemodel.ToolCall
			for _, part := range parts {
				if tc, ok := part.(languagemodel.ToolCall); ok {
					toolCalls = append(toolCalls, tc)
				}
			}
			if len(toolCalls) != 1 {
				t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
			}
			if toolCalls[0].ToolName != "weather" {
				t.Errorf("expected tool name 'weather', got %q", toolCalls[0].ToolName)
			}
			if toolCalls[0].ToolCallID != "call_00_ioIn7yN9p1ZOMNpDLwd4MgAF" {
				t.Errorf("expected tool call ID 'call_00_ioIn7yN9p1ZOMNpDLwd4MgAF', got %q", toolCalls[0].ToolCallID)
			}

			// Verify tool call input
			var inputMap map[string]any
			if err := json.Unmarshal([]byte(toolCalls[0].Input), &inputMap); err != nil {
				t.Errorf("expected valid JSON tool input, got error: %v", err)
			}
			if inputMap["location"] != "San Francisco" {
				t.Errorf("expected location 'San Francisco', got %v", inputMap["location"])
			}

			// Check finish part
			var finishPart *languagemodel.StreamPartFinish
			for _, part := range parts {
				if fp, ok := part.(languagemodel.StreamPartFinish); ok {
					finishPart = &fp
				}
			}
			if finishPart == nil {
				t.Fatal("expected finish part in stream")
			}
			if finishPart.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
				t.Errorf("expected finish reason 'tool-calls', got %q", finishPart.FinishReason.Unified)
			}
		})
	})
}
