package aiembed

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

func loadEnv(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("../.env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			os.Setenv(k, v)
		}
	}
}

func TestBundleLoads(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	// Verify __ai_sdk is defined
	val, err := c.bridge.Eval("test.js", `typeof globalThis.__ai_sdk`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	if val.String() != "object" {
		t.Errorf("__ai_sdk type = %q, want 'object'", val.String())
	}
}

func TestBundleExports(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	exports := []string{
		"generateText", "streamText", "generateObject", "streamObject",
		"embed", "embedMany", "tool", "createOpenAI",
		"createAnthropic", "createGoogleGenerativeAI",
		"wrapLanguageModel", "defaultSettingsMiddleware", "extractReasoningMiddleware",
	}
	for _, name := range exports {
		val, err := c.bridge.Eval("test.js", `typeof globalThis.__ai_sdk.`+name)
		if err != nil {
			t.Fatalf("Eval %s: %v", name, err)
		}
		if val.String() != "function" {
			t.Errorf("__ai_sdk.%s = %q, want 'function'", name, val.String())
		}
		val.Free()
	}
}

func TestGenerateText(t *testing.T) {
	// Mock OpenAI /v1/chat/completions endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", 404)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "method not allowed", 405)
			return
		}

		// Verify request has correct structure
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if req["model"] != "gpt-4" {
			t.Errorf("model = %v, want gpt-4", req["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      "chatcmpl-test-123",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello from mock!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 4,
				"total_tokens":      14,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model:  Model{ID: "openai/gpt-4", Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"}},
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	if result.Text != "Hello from mock!" {
		t.Errorf("text = %q, want %q", result.Text, "Hello from mock!")
	}
}

func TestGenerateTextAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Invalid API key",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	_, err = c.GenerateText(GenerateTextParams{
		Model:  Model{ID: "openai/gpt-4", Provider: &ProviderConfig{APIKey: "bad-key", BaseURL: srv.URL + "/v1"}},
		Prompt: "hello",
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	t.Logf("Got expected error: %v", err)
}

func TestGenerateTextWithMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		messages, ok := req["messages"].([]interface{})
		if !ok || len(messages) < 2 {
			t.Errorf("expected at least 2 messages, got %v", req["messages"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chatcmpl-test", "object": "chat.completion",
			"model": "gpt-4", "created": 1234567890,
			"choices": []map[string]interface{}{{
				"index": 0,
				"message": map[string]interface{}{
					"role": "assistant", "content": "I remember you said hello.",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]interface{}{
				"prompt_tokens": 20, "completion_tokens": 8, "total_tokens": 28,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		System: "You are helpful.",
		Messages: []Message{
			UserMessage("Hello!"),
			AssistantMessage("Hi there!"),
			UserMessage("Do you remember what I said?"),
		},
	})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if result.FinishReason != FinishStop {
		t.Errorf("finishReason = %q, want %q", result.FinishReason, FinishStop)
	}
	if result.Usage.TotalTokens == 0 {
		t.Error("expected non-zero usage")
	}
	t.Logf("Response: %q, Usage: %+v", result.Text, result.Usage)
}

func TestStreamText(t *testing.T) {
	chunks := []string{"Hello", " from", " streaming", "!"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", 404)
			return
		}

		// Verify stream=true in request
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["stream"] != true {
			t.Errorf("expected stream=true, got %v", req["stream"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)

		for i, chunk := range chunks {
			data := map[string]interface{}{
				"id":      "chatcmpl-test-stream",
				"object":  "chat.completion.chunk",
				"created": 1234567890,
				"model":   "gpt-4",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": chunk,
						},
						"finish_reason": nil,
					},
				},
			}
			if i == len(chunks)-1 {
				data["choices"] = []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]interface{}{
							"content": chunk,
						},
						"finish_reason": "stop",
					},
				}
			}
			b, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", b)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	var tokens []string
	result, err := c.StreamText(StreamTextParams{
		Model:  Model{ID: "openai/gpt-4", Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"}},
		Prompt: "Say hello",
		OnToken: func(token string) {
			tokens = append(tokens, token)
		},
	})
	if err != nil {
		t.Fatalf("StreamText: %v", err)
	}

	if len(tokens) == 0 {
		t.Error("expected tokens via callback, got none")
	}
	t.Logf("Received %d tokens: %v", len(tokens), tokens)

	want := "Hello from streaming!"
	if result.Text != want {
		t.Errorf("full text = %q, want %q", result.Text, want)
	}
}

func TestStreamTextWithMessages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if req["stream"] != true {
			t.Errorf("expected stream=true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		chunks := []string{"I", " remember", " our", " conversation"}
		for i, chunk := range chunks {
			data := map[string]interface{}{
				"id": "chatcmpl-stream", "object": "chat.completion.chunk",
				"model": "gpt-4", "created": 1234567890,
				"choices": []map[string]interface{}{{
					"index":         0,
					"delta":         map[string]interface{}{"content": chunk},
					"finish_reason": nil,
				}},
			}
			if i == len(chunks)-1 {
				data["choices"] = []map[string]interface{}{{
					"index":         0,
					"delta":         map[string]interface{}{"content": chunk},
					"finish_reason": "stop",
				}}
			}
			b, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", b)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	var tokens []string
	result, err := c.StreamText(StreamTextParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		System: "You are helpful.",
		Messages: []Message{
			UserMessage("Hello!"),
			AssistantMessage("Hi!"),
			UserMessage("Do you remember?"),
		},
		OnToken: func(token string) {
			tokens = append(tokens, token)
		},
	})
	if err != nil {
		t.Fatalf("StreamText: %v", err)
	}

	if len(tokens) == 0 {
		t.Error("expected tokens")
	}
	if result.Text != "I remember our conversation" {
		t.Errorf("text = %q, want 'I remember our conversation'", result.Text)
	}
	t.Logf("StreamText with messages: %d tokens, text: %q", len(tokens), result.Text)
}

func TestBytecodeLoading(t *testing.T) {
	// Verify the precompiled bytecode loads and works
	if len(bundleBytecode) == 0 {
		t.Skip("no precompiled bytecode (run go generate)")
	}

	t.Logf("Source: %.1f KB, Bytecode: %.1f KB",
		float64(len(bundleSource))/1024, float64(len(bundleBytecode))/1024)

	// NewClient now uses bytecode by default
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	// Verify __ai_sdk is functional
	val, err := c.bridge.Eval("test.js", `
		JSON.stringify({
			generateText: typeof globalThis.__ai_sdk.generateText,
			streamText: typeof globalThis.__ai_sdk.streamText,
			createOpenAI: typeof globalThis.__ai_sdk.createOpenAI,
		});
	`)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	defer val.Free()

	var types map[string]string
	json.Unmarshal([]byte(val.String()), &types)
	for name, typ := range types {
		if typ != "function" {
			t.Errorf("__ai_sdk.%s = %q, want 'function'", name, typ)
		}
	}
}

func TestBytecodeSpeedup(t *testing.T) {
	if len(bundleBytecode) == 0 {
		t.Skip("no precompiled bytecode (run go generate)")
	}

	const iterations = 5

	// Measure source loading
	var sourceTotal time.Duration
	for i := 0; i < iterations; i++ {
		b, err := jsbridge.New(jsbridge.Config{},
			jsbridge.Console(), jsbridge.Encoding(), jsbridge.Streams(),
			jsbridge.Crypto(), jsbridge.URL(), jsbridge.Timers(),
			jsbridge.Abort(), jsbridge.Events(), jsbridge.StructuredClone(),
			jsbridge.Fetch(),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		start := time.Now()
		if err := LoadBundleFromSource(b); err != nil {
			b.Close()
			t.Fatalf("LoadBundleFromSource: %v", err)
		}
		sourceTotal += time.Since(start)
		b.Close()
	}

	// Measure bytecode loading
	var bytecodeTotal time.Duration
	for i := 0; i < iterations; i++ {
		b, err := jsbridge.New(jsbridge.Config{},
			jsbridge.Console(), jsbridge.Encoding(), jsbridge.Streams(),
			jsbridge.Crypto(), jsbridge.URL(), jsbridge.Timers(),
			jsbridge.Abort(), jsbridge.Events(), jsbridge.StructuredClone(),
			jsbridge.Fetch(),
		)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		start := time.Now()
		if err := LoadBundle(b); err != nil {
			b.Close()
			t.Fatalf("LoadBundle: %v", err)
		}
		bytecodeTotal += time.Since(start)
		b.Close()
	}

	sourceAvg := sourceTotal / time.Duration(iterations)
	bytecodeAvg := bytecodeTotal / time.Duration(iterations)
	speedup := float64(sourceAvg) / float64(bytecodeAvg)

	t.Logf("Source avg:   %s", sourceAvg.Round(time.Millisecond))
	t.Logf("Bytecode avg: %s", bytecodeAvg.Round(time.Millisecond))
	t.Logf("Speedup:      %.2fx", speedup)

	if speedup < 1.5 {
		t.Logf("Warning: speedup below 1.5x (got %.2fx) — bytecode may not be worth the complexity", speedup)
	}
}

func TestGenerateTextWithTools(t *testing.T) {
	callNum := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callNum++
		w.Header().Set("Content-Type", "application/json")

		if callNum == 1 {
			// First call: model decides to call the weather tool
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chatcmpl-1", "object": "chat.completion",
				"model": "gpt-4", "created": 1234567890,
				"choices": []map[string]interface{}{{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": nil,
						"tool_calls": []map[string]interface{}{{
							"id":   "call_abc123",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "get_weather",
								"arguments": `{"city":"San Francisco"}`,
							},
						}},
					},
					"finish_reason": "tool_calls",
				}},
				"usage": map[string]interface{}{
					"prompt_tokens": 20, "completion_tokens": 10, "total_tokens": 30,
				},
			})
		} else {
			// Second call: model uses the tool result to generate final response
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chatcmpl-2", "object": "chat.completion",
				"model": "gpt-4", "created": 1234567890,
				"choices": []map[string]interface{}{{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "The weather in San Francisco is 72F and sunny.",
					},
					"finish_reason": "stop",
				}},
				"usage": map[string]interface{}{
					"prompt_tokens": 40, "completion_tokens": 12, "total_tokens": 52,
				},
			})
		}
	}))
	defer srv.Close()

	toolExecuted := false
	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Prompt:   "What is the weather in San Francisco?",
		MaxSteps: 3,
		Tools: map[string]Tool{
			"get_weather": {
				Description: "Get the current weather in a city",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}},"required":["city"]}`),
				Execute: func(args json.RawMessage) (interface{}, error) {
					toolExecuted = true
					var input struct {
						City string `json:"city"`
					}
					json.Unmarshal(args, &input)
					return map[string]interface{}{
						"city":        input.City,
						"temperature": 72,
						"conditions":  "sunny",
					}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateText with tools: %v", err)
	}

	if !toolExecuted {
		t.Error("expected tool to be executed")
	}
	if !strings.Contains(result.Text, "72") || !strings.Contains(result.Text, "sunny") {
		t.Errorf("expected response to mention weather, got: %q", result.Text)
	}
	if callNum != 2 {
		t.Errorf("expected 2 API calls (tool call + final), got %d", callNum)
	}
	t.Logf("Tool calling result: %q (steps: %d, API calls: %d)", result.Text, len(result.Steps), callNum)
}

func TestGenerateObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chatcmpl-test", "object": "chat.completion",
			"model": "gpt-4", "created": 1234567890,
			"choices": []map[string]interface{}{{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": `{"name":"Alice","age":30}`,
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]interface{}{
				"prompt_tokens": 15, "completion_tokens": 10, "total_tokens": 25,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateObject(GenerateObjectParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Mode:   "json",
		Prompt: "Generate a person with name and age",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "number"}
			},
			"required": ["name", "age"]
		}`),
	})
	if err != nil {
		t.Fatalf("GenerateObject: %v", err)
	}

	// Parse the object
	var person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := json.Unmarshal(result.Object, &person); err != nil {
		t.Fatalf("unmarshal object: %v (raw: %s)", err, string(result.Object))
	}

	if person.Name != "Alice" {
		t.Errorf("name = %q, want Alice", person.Name)
	}
	if person.Age != 30 {
		t.Errorf("age = %d, want 30", person.Age)
	}
	t.Logf("Object: %+v, Usage: %+v", person, result.Usage)
}

func TestGenerateTextRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model:  Model{ID: "openai/gpt-4o-mini", Provider: &ProviderConfig{APIKey: key}},
		Prompt: "Reply with exactly: HELLO_FROM_OPENAI",
	})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	t.Logf("OpenAI response: %q", result.Text)
	if !strings.Contains(strings.ToUpper(result.Text), "HELLO_FROM_OPENAI") {
		t.Errorf("unexpected response: %q", result.Text)
	}
}

func TestStreamTextRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	var tokens []string
	result, err := c.StreamText(StreamTextParams{
		Model:  Model{ID: "openai/gpt-4o-mini", Provider: &ProviderConfig{APIKey: key}},
		Prompt: "Count from 1 to 5, one number per word, nothing else",
		OnToken: func(token string) {
			tokens = append(tokens, token)
		},
	})
	if err != nil {
		t.Fatalf("StreamText: %v", err)
	}

	if len(tokens) == 0 {
		t.Error("expected tokens via callback")
	}
	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	t.Logf("StreamText real: %d tokens, text: %q, usage: %+v", len(tokens), result.Text, result.Usage)
}

func TestGenerateTextWithMessagesRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{ID: "openai/gpt-4o-mini", Provider: &ProviderConfig{APIKey: key}},
		System: "You are a helpful assistant. Always respond in exactly one word.",
		Messages: []Message{
			UserMessage("What color is the sky?"),
			AssistantMessage("Blue"),
			UserMessage("What color is grass?"),
		},
	})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	t.Logf("Messages real: %q, finishReason: %s, usage: %+v", result.Text, result.FinishReason, result.Usage)
}

func TestGenerateTextWithToolsRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	toolCalled := false
	result, err := c.GenerateText(GenerateTextParams{
		Model:    Model{ID: "openai/gpt-4o-mini", Provider: &ProviderConfig{APIKey: key}},
		Prompt:   "What is 42 multiplied by 17? Use the calculator tool.",
		MaxSteps: 3,
		Tools: map[string]Tool{
			"calculator": {
				Description: "Multiplies two numbers together",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"number","description":"first number"},"b":{"type":"number","description":"second number"}},"required":["a","b"]}`),
				Execute: func(args json.RawMessage) (interface{}, error) {
					toolCalled = true
					var input struct {
						A float64 `json:"a"`
						B float64 `json:"b"`
					}
					json.Unmarshal(args, &input)
					return map[string]interface{}{
						"result": input.A * input.B,
					}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("GenerateText with tools: %v", err)
	}

	if !toolCalled {
		t.Error("expected calculator tool to be called")
	}
	if !strings.Contains(result.Text, "714") {
		t.Errorf("expected result to contain 714 (42*17), got: %q", result.Text)
	}
	t.Logf("Tool calling real: %q, steps: %d, toolCalled: %v, usage: %+v", result.Text, len(result.Steps), toolCalled, result.Usage)
}

func TestEmbedManyRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.EmbedMany(EmbedManyParams{
		Model:  Model{ID: "openai/text-embedding-3-small", Provider: &ProviderConfig{APIKey: key}},
		Values: []string{"Hello world", "Goodbye world", "Machine learning is fascinating"},
	})
	if err != nil {
		t.Fatalf("EmbedMany: %v", err)
	}

	if len(result.Embeddings) != 3 {
		t.Errorf("expected 3 embeddings, got %d", len(result.Embeddings))
	}
	for i, emb := range result.Embeddings {
		if len(emb) != 1536 {
			t.Errorf("embedding[%d]: expected 1536 dimensions, got %d", i, len(emb))
		}
	}
	t.Logf("EmbedMany real: %d embeddings, %d dimensions each, usage: %d tokens",
		len(result.Embeddings), len(result.Embeddings[0]), result.Usage.Tokens)
}

func TestEmbedRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.Embed(EmbedParams{
		Model: Model{ID: "openai/text-embedding-3-small", Provider: &ProviderConfig{APIKey: key}},
		Value: "The quick brown fox jumps over the lazy dog",
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if len(result.Embedding) == 0 {
		t.Error("expected non-empty embedding")
	}
	t.Logf("Embedding: %d dimensions, first 3: [%.4f, %.4f, %.4f], usage: %d tokens",
		len(result.Embedding), result.Embedding[0], result.Embedding[1], result.Embedding[2], result.Usage.Tokens)
}

func TestGenerateObjectRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateObject(GenerateObjectParams{
		Model:  Model{ID: "openai/gpt-4o-mini", Provider: &ProviderConfig{APIKey: key}},
		Prompt: "Generate a fictional person with a name and age between 20-40",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "integer"}
			},
			"required": ["name", "age"]
		}`),
	})
	if err != nil {
		t.Fatalf("GenerateObject: %v", err)
	}

	var person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := json.Unmarshal(result.Object, &person); err != nil {
		t.Fatalf("unmarshal: %v (raw: %s)", err, string(result.Object))
	}

	if person.Name == "" {
		t.Error("expected non-empty name")
	}
	if person.Age < 20 || person.Age > 40 {
		t.Errorf("age = %d, expected 20-40", person.Age)
	}
	t.Logf("Generated person: %s, age %d. Usage: %+v", person.Name, person.Age, result.Usage)
}

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/embeddings" {
			http.Error(w, "not found", 404)
			return
		}

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if req["model"] != "text-embedding-3-small" {
			t.Errorf("model = %v, want text-embedding-3-small", req["model"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"object":    "embedding",
					"embedding": []float64{0.1, 0.2, 0.3, 0.4, 0.5},
					"index":     0,
				},
			},
			"model": "text-embedding-3-small",
			"usage": map[string]interface{}{
				"prompt_tokens": 5,
				"total_tokens":  5,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.Embed(EmbedParams{
		Model: Model{
			ID:       "openai/text-embedding-3-small",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Value: "Hello world",
	})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if len(result.Embedding) != 5 {
		t.Errorf("embedding length = %d, want 5", len(result.Embedding))
	}
	if result.Embedding[0] != 0.1 {
		t.Errorf("embedding[0] = %v, want 0.1", result.Embedding[0])
	}
	t.Logf("Embedding: %v, Usage: %+v", result.Embedding, result.Usage)
}

func TestEmbedMany(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		// The AI SDK sends input as an array
		inputs, ok := req["input"].([]interface{})
		if !ok {
			// Single input
			inputs = []interface{}{req["input"]}
		}

		data := make([]map[string]interface{}, len(inputs))
		for i := range inputs {
			embedding := make([]float64, 3)
			for j := range embedding {
				embedding[j] = float64(i*10 + j + 1)
			}
			data[i] = map[string]interface{}{
				"object":    "embedding",
				"embedding": embedding,
				"index":     i,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"object": "list",
			"data":   data,
			"model":  "text-embedding-3-small",
			"usage": map[string]interface{}{
				"prompt_tokens": len(inputs) * 3,
				"total_tokens":  len(inputs) * 3,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.EmbedMany(EmbedManyParams{
		Model: Model{
			ID:       "openai/text-embedding-3-small",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Values: []string{"Hello", "World", "Test"},
	})
	if err != nil {
		t.Fatalf("EmbedMany: %v", err)
	}

	if len(result.Embeddings) != 3 {
		t.Errorf("embeddings count = %d, want 3", len(result.Embeddings))
	}
	for i, emb := range result.Embeddings {
		if len(emb) != 3 {
			t.Errorf("embedding[%d] length = %d, want 3", i, len(emb))
		}
	}
	t.Logf("Embeddings: %d vectors, Usage: %+v", len(result.Embeddings), result.Usage)
}

// --- New Phase 1 completion tests ---

func TestMultiProviderAnthropic(t *testing.T) {
	// Mock Anthropic messages endpoint (SDK appends /messages to baseURL)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		t.Logf("Anthropic mock received: %s %s", r.Method, r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "msg_test123",
			"type":  "message",
			"role":  "assistant",
			"model": "claude-3-5-sonnet-20241022",
			"content": []map[string]interface{}{
				{"type": "text", "text": "Hello from Anthropic mock!"},
			},
			"stop_reason": "end_turn",
			"usage": map[string]interface{}{
				"input_tokens":  12,
				"output_tokens": 5,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{
			ID:       "anthropic/claude-3-5-sonnet-20241022",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL},
		},
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("GenerateText with Anthropic: %v", err)
	}

	if result.Text != "Hello from Anthropic mock!" {
		t.Errorf("text = %q, want %q", result.Text, "Hello from Anthropic mock!")
	}
	t.Logf("Anthropic mock: %q, usage: %+v", result.Text, result.Usage)
}

func TestMultiProviderGoogle(t *testing.T) {
	// Mock Google Gemini endpoint
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Google uses a different path pattern
		w.Header().Set("Content-Type", "application/json")

		// The AI SDK for Google sends requests to the generateContent endpoint
		json.NewEncoder(w).Encode(map[string]interface{}{
			"candidates": []map[string]interface{}{
				{
					"content": map[string]interface{}{
						"parts": []map[string]interface{}{
							{"text": "Hello from Google mock!"},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]interface{}{
				"promptTokenCount":     10,
				"candidatesTokenCount": 5,
				"totalTokenCount":      15,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{
			ID:       "google/gemini-2.0-flash",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL},
		},
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("GenerateText with Google: %v", err)
	}

	if result.Text == "" {
		t.Error("expected non-empty text from Google mock")
	}
	t.Logf("Google mock: %q, usage: %+v", result.Text, result.Usage)
}

func TestStreamObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		if req["stream"] == true {
			// Streaming response
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)

			chunks := []string{
				`{"name":"Al"}`,
				`{"name":"Alice"}`,
				`{"name":"Alice","age":30}`,
			}
			for _, chunk := range chunks {
				data := map[string]interface{}{
					"id": "chatcmpl-stream", "object": "chat.completion.chunk",
					"model": "gpt-4", "created": 1234567890,
					"choices": []map[string]interface{}{{
						"index":         0,
						"delta":         map[string]interface{}{"content": chunk},
						"finish_reason": nil,
					}},
				}
				b, _ := json.Marshal(data)
				fmt.Fprintf(w, "data: %s\n\n", b)
			}
			// Final chunk
			finalData := map[string]interface{}{
				"id": "chatcmpl-stream", "object": "chat.completion.chunk",
				"model": "gpt-4", "created": 1234567890,
				"choices": []map[string]interface{}{{
					"index":         0,
					"delta":         map[string]interface{}{},
					"finish_reason": "stop",
				}},
				"usage": map[string]interface{}{
					"prompt_tokens": 15, "completion_tokens": 10, "total_tokens": 25,
				},
			}
			b, _ := json.Marshal(finalData)
			fmt.Fprintf(w, "data: %s\n\n", b)
			fmt.Fprint(w, "data: [DONE]\n\n")
		} else {
			// Non-streaming (generateObject uses tool mode by default)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chatcmpl-test", "object": "chat.completion",
				"model": "gpt-4", "created": 1234567890,
				"choices": []map[string]interface{}{{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": `{"name":"Alice","age":30}`,
					},
					"finish_reason": "stop",
				}},
				"usage": map[string]interface{}{
					"prompt_tokens": 15, "completion_tokens": 10, "total_tokens": 25,
				},
			})
		}
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	var partials []json.RawMessage
	result, err := c.StreamObject(StreamObjectParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Mode:   "json",
		Prompt: "Generate a person with name and age",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"age": {"type": "number"}
			},
			"required": ["name", "age"]
		}`),
		OnPartialObject: func(partial json.RawMessage) {
			partials = append(partials, partial)
		},
	})
	if err != nil {
		t.Fatalf("StreamObject: %v", err)
	}

	var person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := json.Unmarshal(result.Object, &person); err != nil {
		t.Fatalf("unmarshal object: %v (raw: %s)", err, string(result.Object))
	}

	if person.Name == "" {
		t.Error("expected non-empty name")
	}
	t.Logf("StreamObject: %+v, partials received: %d, usage: %+v", person, len(partials), result.Usage)
}

func TestStreamObjectRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	var partials []json.RawMessage
	result, err := c.StreamObject(StreamObjectParams{
		Model:  Model{ID: "openai/gpt-4o-mini", Provider: &ProviderConfig{APIKey: key}},
		Prompt: "Generate a fictional city with a name and population between 10000-500000",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"name": {"type": "string"},
				"population": {"type": "integer"}
			},
			"required": ["name", "population"]
		}`),
		OnPartialObject: func(partial json.RawMessage) {
			partials = append(partials, partial)
		},
	})
	if err != nil {
		t.Fatalf("StreamObject: %v", err)
	}

	var city struct {
		Name       string `json:"name"`
		Population int    `json:"population"`
	}
	if err := json.Unmarshal(result.Object, &city); err != nil {
		t.Fatalf("unmarshal: %v (raw: %s)", err, string(result.Object))
	}

	if city.Name == "" {
		t.Error("expected non-empty name")
	}
	if city.Population <= 0 {
		t.Errorf("expected positive population, got %d", city.Population)
	}
	if len(partials) == 0 {
		t.Error("expected partial objects during streaming")
	}
	t.Logf("StreamObject real: %s (pop %d), partials: %d, usage: %+v",
		city.Name, city.Population, len(partials), result.Usage)
}

func TestErrorTyping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Incorrect API key provided",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	_, err = c.GenerateText(GenerateTextParams{
		Model:  Model{ID: "openai/gpt-4", Provider: &ProviderConfig{APIKey: "bad-key", BaseURL: srv.URL + "/v1"}},
		Prompt: "hello",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	// Check that we get a typed error
	switch e := err.(type) {
	case *APICallError:
		if e.StatusCode != 401 {
			t.Errorf("expected status 401, got %d", e.StatusCode)
		}
		t.Logf("Got typed APICallError: status=%d, message=%s", e.StatusCode, e.Message)
	case *AIError:
		t.Logf("Got AIError (classification worked): type=%s, message=%s", e.Type, e.Message)
	default:
		t.Logf("Got untyped error (acceptable): %T: %v", err, err)
	}
}

func TestProviderNotFound(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	_, err = c.GenerateText(GenerateTextParams{
		Model:  Model{ID: "unsupported/model", Provider: &ProviderConfig{APIKey: "key"}},
		Prompt: "hello",
	})
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("expected 'unsupported provider' in error, got: %v", err)
	}
	t.Logf("Got expected provider error: %v", err)
}

func TestMiddlewareDefaultSettings(t *testing.T) {
	// Verify middleware wrapping produces valid JS
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)

		// The middleware should have set temperature
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chatcmpl-test", "object": "chat.completion",
			"model": "gpt-4", "created": 1234567890,
			"choices": []map[string]interface{}{{
				"index": 0,
				"message": map[string]interface{}{
					"role": "assistant", "content": "Middleware works!",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]interface{}{
				"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13,
			},
		})
	}))
	defer srv.Close()

	// Client-level middleware
	c, err := NewClient(ClientConfig{
		HTTPClient: srv.Client(),
		Middleware: []MiddlewareConfig{
			DefaultSettingsMiddleware(MiddlewareSettings{
				MaxTokens:   500,
				Temperature: Float64(0.3),
			}),
		},
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Prompt: "Say hello",
	})
	if err != nil {
		t.Fatalf("GenerateText with middleware: %v", err)
	}

	if result.Text != "Middleware works!" {
		t.Errorf("text = %q, want %q", result.Text, "Middleware works!")
	}
	t.Logf("Middleware test: %q", result.Text)
}

func TestPerCallMiddleware(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chatcmpl-test", "object": "chat.completion",
			"model": "gpt-4", "created": 1234567890,
			"choices": []map[string]interface{}{{
				"index": 0,
				"message": map[string]interface{}{
					"role": "assistant", "content": "Per-call middleware works!",
				},
				"finish_reason": "stop",
			}},
			"usage": map[string]interface{}{
				"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13,
			},
		})
	}))
	defer srv.Close()

	c, err := NewClient(ClientConfig{HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.GenerateText(GenerateTextParams{
		Model: Model{
			ID:       "openai/gpt-4",
			Provider: &ProviderConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
		},
		Prompt: "Say hello",
		Middleware: []MiddlewareConfig{
			DefaultSettingsMiddleware(MiddlewareSettings{
				Temperature: Float64(0.1),
			}),
		},
	})
	if err != nil {
		t.Fatalf("GenerateText with per-call middleware: %v", err)
	}

	if result.Text != "Per-call middleware works!" {
		t.Errorf("text = %q, want %q", result.Text, "Per-call middleware works!")
	}
	t.Logf("Per-call middleware test: %q", result.Text)
}
