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

	"github.com/brainlet/brainkit/jsbridge"
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

	exports := []string{"generateText", "streamText", "generateObject", "streamObject", "embed", "embedMany", "tool", "createOpenAI"}
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
