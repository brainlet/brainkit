package aiembed

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fastschema/qjs"
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
	val, err := c.bridge.Eval("test.js", qjs.Code(`typeof globalThis.__ai_sdk`))
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	if val.String() != "object" {
		t.Errorf("__ai_sdk type = %q, want 'object'", val.String())
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
		BaseURL: srv.URL + "/v1",
		APIKey:  "test-key",
		Model:   "gpt-4",
		Prompt:  "Say hello",
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
		BaseURL: srv.URL + "/v1",
		APIKey:  "bad-key",
		Model:   "gpt-4",
		Prompt:  "hello",
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	t.Logf("Got expected error: %v", err)
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
		BaseURL: "https://api.openai.com/v1",
		APIKey:  key,
		Model:   "gpt-4o-mini",
		Prompt:  "Reply with exactly: HELLO_FROM_OPENAI",
	})
	if err != nil {
		t.Fatalf("GenerateText: %v", err)
	}

	t.Logf("OpenAI response: %q", result.Text)
	if !strings.Contains(strings.ToUpper(result.Text), "HELLO_FROM_OPENAI") {
		t.Errorf("unexpected response: %q", result.Text)
	}
}
