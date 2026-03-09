// Ported from: packages/perplexity/src/perplexity-language-model.test.ts
package perplexity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// testPrompt is the standard test prompt used across tests.
var testPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

const modelID = "sonar"

// mockIDCounter creates a deterministic ID generator for tests.
func mockIDCounter() func() string {
	counter := 0
	return func() string {
		counter++
		return fmt.Sprintf("id-%d", counter)
	}
}

// perplexityTextFixture returns the JSON fixture for perplexity-text responses.
func perplexityTextFixture() map[string]any {
	return map[string]any{
		"id":      "aec30d94-c6a5-4d30-935e-97dbe8de9f85",
		"model":   "sonar",
		"created": float64(1770768220),
		"usage": map[string]any{
			"prompt_tokens":     float64(11),
			"completion_tokens": float64(392),
			"total_tokens":      float64(403),
		},
		"citations": []any{
			"https://www.modernchristmastrees.com/innovative-approaches-holiday-decor-modern-interior-design/",
			"https://camillestyles.com/design/modern-christmas-decorating-ideas/",
			"https://www.elledecor.com/design-decorate/interior-designers/advice/g2833/christmas-decorating-ideas/",
			"https://www.youtube.com/watch?v=GVxMMHAwzIU",
			"https://www.jjonesdesignco.com/blogs/modern-industrial-christmas-decor-joshuas-must-have-holiday-picks",
			"https://www.bhsusa.com/blog/5-decorations-for-a-mid-century-modern-inspired-holiday",
		},
		"object": "chat.completion",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": "**EcoVista Day** is a new annual holiday celebrated on the first Saturday of October...",
				},
				"finish_reason": "stop",
			},
		},
	}
}

// perplexityCitationsFixture returns the JSON fixture for perplexity-citations responses.
func perplexityCitationsFixture() map[string]any {
	return map[string]any{
		"id":      "702738a1-c1e0-4a7f-b9ab-0f6fe1b13514",
		"model":   "sonar",
		"created": float64(1770768226),
		"usage": map[string]any{
			"prompt_tokens":     float64(10),
			"completion_tokens": float64(251),
			"total_tokens":      float64(261),
		},
		"citations": []any{
			"https://populationstat.com/united-states/san-francisco",
			"https://en.wikipedia.org/wiki/San_Francisco",
			"https://www.california-demographics.com/cities_by_population",
			"https://wfin.com/fox-sports/pat-mcafee-says-san-francisco-wasnt-the-s-hole-he-thought-it-may-be-during-super-bowl-week/",
			"https://fred.stlouisfed.org/series/CASANF0POP",
			"https://worldpopulationreview.com/us-cities/california/san-francisco",
			"https://worldpopulationreview.com/us-counties/california/san-francisco-county",
		},
		"object": "chat.completion",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": "The most recent estimates for San Francisco's city population...",
				},
				"finish_reason": "stop",
			},
		},
	}
}

// createTestServer creates an httptest server that serves the given JSON body.
func createTestServer(body any, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		// Set response headers
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")

		// Write response
		json.NewEncoder(w).Encode(body)
	}))
	return server, capture
}

// createSSETestServer creates an httptest server that serves SSE chunks.
func createSSETestServer(chunks []string, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		for k, v := range headers {
			w.Header().Set(k, v)
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprint(w, chunk)
			flusher.Flush()
		}
	}))
	return server, capture
}

type requestCapture struct {
	Body    []byte
	Headers http.Header
}

func (rc *requestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

// createModel creates a PerplexityLanguageModel that targets a test server.
func createModel(baseURL string) *PerplexityLanguageModel {
	return NewPerplexityLanguageModel(modelID, PerplexityChatConfig{
		BaseURL: baseURL,
		Headers: func() map[string]string {
			return map[string]string{
				"authorization": "Bearer test-token",
				"content-type":  "application/json",
			}
		},
		GenerateID: mockIDCounter(),
	})
}

// createModelWithHeaders creates a PerplexityLanguageModel with custom headers.
func createModelWithHeaders(baseURL string, headers map[string]string) *PerplexityLanguageModel {
	return NewPerplexityLanguageModel(modelID, PerplexityChatConfig{
		BaseURL: baseURL,
		Headers: func() map[string]string {
			return headers
		},
		GenerateID: mockIDCounter(),
	})
}

// --- perplexity text stream chunks ---
func perplexityTextStreamChunks() []string {
	chunks := []string{
		`data: {"id":"a3d55d44-63f9-4704-bb26-e17be1ddab3a","model":"sonar","created":1770768233,"usage":{"prompt_tokens":11,"completion_tokens":1,"total_tokens":12},"citations":["https://www.modernchristmastrees.com/innovative-approaches-holiday-decor-modern-interior-design/","https://camillestyles.com/design/modern-christmas-decorating-ideas/","https://www.elledecor.com/design-decorate/interior-designers/advice/g2833/christmas-decorating-ideas/","https://www.youtube.com/watch?v=GVxMMHAwzIU","https://www.jjonesdesignco.com/blogs/modern-industrial-christmas-decor-joshuas-must-have-holiday-picks"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"**"}}]}` + "\n\n",
		`data: {"id":"a3d55d44-63f9-4704-bb26-e17be1ddab3a","model":"sonar","created":1770768233,"usage":{"prompt_tokens":11,"completion_tokens":3,"total_tokens":14},"citations":["https://www.modernchristmastrees.com/innovative-approaches-holiday-decor-modern-interior-design/","https://camillestyles.com/design/modern-christmas-decorating-ideas/","https://www.elledecor.com/design-decorate/interior-designers/advice/g2833/christmas-decorating-ideas/","https://www.youtube.com/watch?v=GVxMMHAwzIU","https://www.jjonesdesignco.com/blogs/modern-industrial-christmas-decor-joshuas-must-have-holiday-picks"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Eco"}}]}` + "\n\n",
		`data: {"id":"a3d55d44-63f9-4704-bb26-e17be1ddab3a","model":"sonar","created":1770768233,"usage":{"prompt_tokens":11,"completion_tokens":5,"total_tokens":16},"citations":["https://www.modernchristmastrees.com/innovative-approaches-holiday-decor-modern-interior-design/","https://camillestyles.com/design/modern-christmas-decorating-ideas/","https://www.elledecor.com/design-decorate/interior-designers/advice/g2833/christmas-decorating-ideas/","https://www.youtube.com/watch?v=GVxMMHAwzIU","https://www.jjonesdesignco.com/blogs/modern-industrial-christmas-decor-joshuas-must-have-holiday-picks"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Vista"}}]}` + "\n\n",
		`data: {"id":"a3d55d44-63f9-4704-bb26-e17be1ddab3a","model":"sonar","created":1770768237,"usage":{"prompt_tokens":11,"completion_tokens":434,"total_tokens":445},"citations":["https://www.modernchristmastrees.com/innovative-approaches-holiday-decor-modern-interior-design/","https://camillestyles.com/design/modern-christmas-decorating-ideas/","https://www.elledecor.com/design-decorate/interior-designers/advice/g2833/christmas-decorating-ideas/","https://www.youtube.com/watch?v=GVxMMHAwzIU","https://www.jjonesdesignco.com/blogs/modern-industrial-christmas-decor-joshuas-must-have-holiday-picks"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":"stop"}]}` + "\n\n",
		"data: [DONE]\n\n",
	}
	return chunks
}

func perplexityCitationsStreamChunks() []string {
	chunks := []string{
		`data: {"id":"58cb9740-f356-49e9-b71e-a02a1376c1b9","model":"sonar","created":1770768240,"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11},"citations":["https://populationstat.com/united-states/san-francisco","https://en.wikipedia.org/wiki/San_Francisco","https://fred.stlouisfed.org/graph/?g=4K5j","https://www.california-demographics.com/cities_by_population","https://worldpopulationreview.com/us-cities/california/san-francisco","https://worldpopulationreview.com/us-counties/california/san-francisco-county","https://www.worldometers.info/world-population/us-population/"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"The"}}]}` + "\n\n",
		`data: {"id":"58cb9740-f356-49e9-b71e-a02a1376c1b9","model":"sonar","created":1770768240,"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12},"citations":["https://populationstat.com/united-states/san-francisco","https://en.wikipedia.org/wiki/San_Francisco","https://fred.stlouisfed.org/graph/?g=4K5j","https://www.california-demographics.com/cities_by_population","https://worldpopulationreview.com/us-cities/california/san-francisco","https://worldpopulationreview.com/us-counties/california/san-francisco-county","https://www.worldometers.info/world-population/us-population/"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":" current"}}]}` + "\n\n",
		`data: {"id":"58cb9740-f356-49e9-b71e-a02a1376c1b9","model":"sonar","created":1770768244,"usage":{"prompt_tokens":10,"completion_tokens":336,"total_tokens":346},"citations":["https://populationstat.com/united-states/san-francisco","https://en.wikipedia.org/wiki/San_Francisco","https://fred.stlouisfed.org/graph/?g=4K5j","https://www.california-demographics.com/cities_by_population","https://worldpopulationreview.com/us-cities/california/san-francisco","https://worldpopulationreview.com/us-counties/california/san-francisco-county","https://www.worldometers.info/world-population/us-population/"],"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":"stop"}]}` + "\n\n",
		"data: [DONE]\n\n",
	}
	return chunks
}

// collectStreamParts drains a stream channel into a slice.
func collectStreamParts(stream <-chan languagemodel.StreamPart) []languagemodel.StreamPart {
	var parts []languagemodel.StreamPart
	for part := range stream {
		parts = append(parts, part)
	}
	return parts
}

// intPtr is a helper to create a *int.
func intPtr(v int) *int { return &v }

// boolPtr is a helper to create a *bool.
func boolPtr(v bool) *bool { return &v }

// strPtr is a helper to create a *string.
func strPtr(v string) *string { return &v }

// ===== DoGenerate tests =====

func TestDoGenerate_Text(t *testing.T) {
	t.Run("should extract text content", func(t *testing.T) {
		server, _ := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have text content
		var text string
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.Text); ok {
				text += tc.Text
			}
		}
		if text == "" {
			t.Fatal("expected non-empty text content")
		}
		if !strings.Contains(text, "EcoVista Day") {
			t.Errorf("expected text to contain 'EcoVista Day', got: %s", text)
		}

		// Should have citations as sources
		var sourceCount int
		for _, c := range result.Content {
			if _, ok := c.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		if sourceCount != 6 {
			t.Errorf("expected 6 source URLs, got %d", sourceCount)
		}
	})
}

func TestDoGenerate_Citations(t *testing.T) {
	t.Run("should extract citation content", func(t *testing.T) {
		server, _ := createTestServer(perplexityCitationsFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have text content
		var text string
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.Text); ok {
				text += tc.Text
			}
		}
		if text == "" {
			t.Fatal("expected non-empty text content")
		}

		// Should have 7 citations as sources
		var sourceCount int
		for _, c := range result.Content {
			if _, ok := c.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		if sourceCount != 7 {
			t.Errorf("expected 7 source URLs, got %d", sourceCount)
		}
	})
}

func TestDoGenerate_RequestBody(t *testing.T) {
	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "sonar" {
			t.Errorf("expected model 'sonar', got %v", body["model"])
		}

		messages, ok := body["messages"].([]any)
		if !ok {
			t.Fatalf("expected messages to be []any, got %T", body["messages"])
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
		if msg["content"] != "Hello" {
			t.Errorf("expected content 'Hello', got %v", msg["content"])
		}
	})
}

func TestDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should pass through perplexity provider options", func(t *testing.T) {
		server, capture := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"perplexity": {
					"search_recency_filter": "month",
					"return_images":         true,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["search_recency_filter"] != "month" {
			t.Errorf("expected search_recency_filter 'month', got %v", body["search_recency_filter"])
		}
		if body["return_images"] != true {
			t.Errorf("expected return_images true, got %v", body["return_images"])
		}
	})
}

func TestDoGenerate_Headers(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModelWithHeaders(server.URL, map[string]string{
			"authorization":        "Bearer test-api-key",
			"Custom-Provider-Header": "provider-header-value",
		})

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check that the provider header was sent
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		// Check that the request header was sent
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}
		// Check authorization header
		if capture.Headers.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q",
				capture.Headers.Get("Authorization"))
		}
	})
}

func TestDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createTestServer(perplexityTextFixture(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if result.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header 'test-value', got %q", result.Response.Headers["Test-Header"])
		}
	})
}

func TestDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check input tokens
		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 11 {
			t.Errorf("expected input total tokens 11, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.InputTokens.NoCache == nil || *result.Usage.InputTokens.NoCache != 11 {
			t.Errorf("expected input noCache tokens 11, got %v", result.Usage.InputTokens.NoCache)
		}

		// Check output tokens
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 392 {
			t.Errorf("expected output total tokens 392, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 392 {
			t.Errorf("expected output text tokens 392, got %v", result.Usage.OutputTokens.Text)
		}
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 0 {
			t.Errorf("expected output reasoning tokens 0, got %v", result.Usage.OutputTokens.Reasoning)
		}
	})
}

func TestDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should send additional response information", func(t *testing.T) {
		server, _ := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if result.Response.ID == nil || *result.Response.ID != "aec30d94-c6a5-4d30-935e-97dbe8de9f85" {
			t.Errorf("expected ID 'aec30d94-c6a5-4d30-935e-97dbe8de9f85', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "sonar" {
			t.Errorf("expected ModelID 'sonar', got %v", result.Response.ModelID)
		}
		expectedTime := time.Unix(1770768220, 0)
		if result.Response.Timestamp == nil || !result.Response.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, result.Response.Timestamp)
		}
	})
}

func TestDoGenerate_PDFBase64(t *testing.T) {
	t.Run("should handle PDF files with base64 encoding", func(t *testing.T) {
		server, capture := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		prompt := languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Analyze this PDF"},
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "mock-pdf-data"},
						Filename:  strPtr("test.pdf"),
					},
				},
			},
		}

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: prompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		messages := body["messages"].([]any)
		msg := messages[0].(map[string]any)
		contentParts := msg["content"].([]any)

		if len(contentParts) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(contentParts))
		}

		// First part: text
		textPart := contentParts[0].(map[string]any)
		if textPart["type"] != "text" {
			t.Errorf("expected type 'text', got %v", textPart["type"])
		}
		if textPart["text"] != "Analyze this PDF" {
			t.Errorf("expected text 'Analyze this PDF', got %v", textPart["text"])
		}

		// Second part: file_url
		filePart := contentParts[1].(map[string]any)
		if filePart["type"] != "file_url" {
			t.Errorf("expected type 'file_url', got %v", filePart["type"])
		}
		if filePart["file_name"] != "test.pdf" {
			t.Errorf("expected file_name 'test.pdf', got %v", filePart["file_name"])
		}
		fileURL := filePart["file_url"].(map[string]any)
		if fileURL["url"] != "mock-pdf-data" {
			t.Errorf("expected url 'mock-pdf-data', got %v", fileURL["url"])
		}
	})
}

func TestDoGenerate_PDFURLs(t *testing.T) {
	t.Run("should handle PDF files with URLs", func(t *testing.T) {
		server, capture := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		prompt := languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Analyze this PDF"},
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "https://example.com/test.pdf"},
						Filename:  strPtr("test.pdf"),
					},
				},
			},
		}

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: prompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		messages := body["messages"].([]any)
		msg := messages[0].(map[string]any)
		contentParts := msg["content"].([]any)

		if len(contentParts) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(contentParts))
		}

		// Second part: file_url
		filePart := contentParts[1].(map[string]any)
		if filePart["type"] != "file_url" {
			t.Errorf("expected type 'file_url', got %v", filePart["type"])
		}
		if filePart["file_name"] != "test.pdf" {
			t.Errorf("expected file_name 'test.pdf', got %v", filePart["file_name"])
		}
		fileURL := filePart["file_url"].(map[string]any)
		if fileURL["url"] != "https://example.com/test.pdf" {
			t.Errorf("expected url 'https://example.com/test.pdf', got %v", fileURL["url"])
		}
	})
}

func TestDoGenerate_Images(t *testing.T) {
	t.Run("should extract images", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "test-id",
			"created": float64(1680000000),
			"model":   modelID,
			"choices": []any{
				map[string]any{
					"message":       map[string]any{"role": "assistant", "content": ""},
					"finish_reason": "stop",
				},
			},
			"images": []any{
				map[string]any{
					"image_url":  "https://example.com/image.jpg",
					"origin_url": "https://example.com/image.jpg",
					"height":     float64(100),
					"width":      float64(100),
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(10),
				"completion_tokens": float64(20),
				"total_tokens":      float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		perplexityMeta := result.ProviderMetadata["perplexity"]
		images := perplexityMeta["images"].([]map[string]any)
		if len(images) != 1 {
			t.Fatalf("expected 1 image, got %d", len(images))
		}
		if images[0]["imageUrl"] != "https://example.com/image.jpg" {
			t.Errorf("expected imageUrl, got %v", images[0]["imageUrl"])
		}
		if images[0]["originUrl"] != "https://example.com/image.jpg" {
			t.Errorf("expected originUrl, got %v", images[0]["originUrl"])
		}
		if images[0]["height"] != 100 {
			t.Errorf("expected height 100, got %v", images[0]["height"])
		}
		if images[0]["width"] != 100 {
			t.Errorf("expected width 100, got %v", images[0]["width"])
		}

		// Check usage metadata
		usageMeta := perplexityMeta["usage"].(map[string]any)
		if usageMeta["citationTokens"] != nil {
			t.Errorf("expected citationTokens nil, got %v", usageMeta["citationTokens"])
		}
		if usageMeta["numSearchQueries"] != nil {
			t.Errorf("expected numSearchQueries nil, got %v", usageMeta["numSearchQueries"])
		}
	})
}

func TestDoGenerate_ExtendedUsage(t *testing.T) {
	t.Run("should extract extended usage", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "test-id",
			"created": float64(1680000000),
			"model":   modelID,
			"choices": []any{
				map[string]any{
					"message":       map[string]any{"role": "assistant", "content": ""},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":      float64(10),
				"completion_tokens":  float64(20),
				"total_tokens":       float64(30),
				"citation_tokens":    float64(30),
				"num_search_queries": float64(40),
				"reasoning_tokens":   float64(50),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check usage
		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 10 {
			t.Errorf("expected input total 10, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 20 {
			t.Errorf("expected output total 20, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 50 {
			t.Errorf("expected output reasoning 50, got %v", result.Usage.OutputTokens.Reasoning)
		}
		// text = completion_tokens - reasoning_tokens = 20 - 50 = -30
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != -30 {
			t.Errorf("expected output text -30, got %v", result.Usage.OutputTokens.Text)
		}

		// Check provider metadata
		perplexityMeta := result.ProviderMetadata["perplexity"]
		usageMeta := perplexityMeta["usage"].(map[string]any)
		if usageMeta["citationTokens"] != 30 {
			t.Errorf("expected citationTokens 30, got %v", usageMeta["citationTokens"])
		}
		if usageMeta["numSearchQueries"] != 40 {
			t.Errorf("expected numSearchQueries 40, got %v", usageMeta["numSearchQueries"])
		}
	})
}

// ===== DoStream tests =====

func TestDoStream_Text(t *testing.T) {
	t.Run("should stream text", func(t *testing.T) {
		server, _ := createSSETestServer(perplexityTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have text deltas
		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if fullText == "" {
			t.Fatal("expected non-empty streamed text")
		}
		if !strings.Contains(fullText, "**") {
			t.Errorf("expected text to contain '**', got: %s", fullText)
		}
		if !strings.Contains(fullText, "EcoVista") {
			t.Errorf("expected text to contain 'EcoVista', got: %s", fullText)
		}

		// Should have a finish part
		var gotFinish bool
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartFinish); ok {
				gotFinish = true
			}
		}
		if !gotFinish {
			t.Error("expected finish part in stream")
		}
	})
}

func TestDoStream_Citations(t *testing.T) {
	t.Run("should stream citations", func(t *testing.T) {
		server, _ := createSSETestServer(perplexityCitationsStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have source URLs from the first chunk
		var sourceCount int
		for _, part := range parts {
			if _, ok := part.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		if sourceCount != 7 {
			t.Errorf("expected 7 source URLs, got %d", sourceCount)
		}

		// Should have text deltas
		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if fullText == "" {
			t.Fatal("expected non-empty streamed text")
		}
	})
}

func TestDoStream_RequestBody(t *testing.T) {
	t.Run("should send correct streaming request body", func(t *testing.T) {
		server, capture := createSSETestServer(perplexityTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		body := capture.BodyJSON()
		if body["model"] != "sonar" {
			t.Errorf("expected model 'sonar', got %v", body["model"])
		}
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}

		messages := body["messages"].([]any)
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
		if msg["content"] != "Hello" {
			t.Errorf("expected content 'Hello', got %v", msg["content"])
		}
	})
}

func TestDoStream_Headers(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createSSETestServer(perplexityTextStreamChunks(), nil)
		defer server.Close()
		model := createModelWithHeaders(server.URL, map[string]string{
			"authorization":        "Bearer test-api-key",
			"Custom-Provider-Header": "provider-header-value",
		})

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		// Check headers
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}
		if capture.Headers.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q",
				capture.Headers.Get("Authorization"))
		}
	})
}

func TestDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createSSETestServer(perplexityTextStreamChunks(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		if streamResult.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if streamResult.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header 'test-value', got %q",
				streamResult.Response.Headers["Test-Header"])
		}
	})
}

func TestDoStream_Images(t *testing.T) {
	t.Run("should stream images", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"id":      "stream-id",
			"created": 1680003600,
			"model":   modelID,
			"images": []any{
				map[string]any{
					"image_url":  "https://example.com/image.jpg",
					"origin_url": "https://example.com/image.jpg",
					"height":     100,
					"width":      100,
				},
			},
			"choices": []any{
				map[string]any{
					"delta":         map[string]any{"role": "assistant", "content": "Hello"},
					"finish_reason": nil,
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"id":      "stream-id",
			"created": 1680003600,
			"model":   modelID,
			"choices": []any{
				map[string]any{
					"delta":         map[string]any{"role": "assistant", "content": ""},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		})

		chunks := []string{
			fmt.Sprintf("data: %s\n\n", chunk1),
			fmt.Sprintf("data: %s\n\n", chunk2),
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Find finish part and check provider metadata
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}

		perplexityMeta := finishPart.ProviderMetadata["perplexity"]
		images := perplexityMeta["images"].([]map[string]any)
		if len(images) != 1 {
			t.Fatalf("expected 1 image, got %d", len(images))
		}
		if images[0]["imageUrl"] != "https://example.com/image.jpg" {
			t.Errorf("expected imageUrl, got %v", images[0]["imageUrl"])
		}
		if images[0]["originUrl"] != "https://example.com/image.jpg" {
			t.Errorf("expected originUrl, got %v", images[0]["originUrl"])
		}
	})
}

func TestDoStream_ExtendedUsage(t *testing.T) {
	t.Run("should stream extended usage", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"id":      "stream-id",
			"created": 1680003600,
			"model":   modelID,
			"choices": []any{
				map[string]any{
					"delta":         map[string]any{"role": "assistant", "content": "Hello"},
					"finish_reason": nil,
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"id":      "stream-id",
			"created": 1680003600,
			"model":   modelID,
			"choices": []any{
				map[string]any{
					"delta":         map[string]any{"role": "assistant", "content": ""},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":      11,
				"completion_tokens":  21,
				"total_tokens":       32,
				"citation_tokens":    30,
				"num_search_queries": 40,
				"reasoning_tokens":   50,
			},
		})

		chunks := []string{
			fmt.Sprintf("data: %s\n\n", chunk1),
			fmt.Sprintf("data: %s\n\n", chunk2),
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Find finish part
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}

		// Check usage
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 11 {
			t.Errorf("expected input total 11, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 21 {
			t.Errorf("expected output total 21, got %v", finishPart.Usage.OutputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Reasoning == nil || *finishPart.Usage.OutputTokens.Reasoning != 50 {
			t.Errorf("expected output reasoning 50, got %v", finishPart.Usage.OutputTokens.Reasoning)
		}
		// text = 21 - 50 = -29
		if finishPart.Usage.OutputTokens.Text == nil || *finishPart.Usage.OutputTokens.Text != -29 {
			t.Errorf("expected output text -29, got %v", finishPart.Usage.OutputTokens.Text)
		}

		// Check provider metadata
		perplexityMeta := finishPart.ProviderMetadata["perplexity"]
		usageMeta := perplexityMeta["usage"].(map[string]any)
		// citation_tokens and num_search_queries should be present
		if usageMeta["citationTokens"] == nil {
			t.Error("expected citationTokens to be non-nil")
		}
		if usageMeta["numSearchQueries"] == nil {
			t.Error("expected numSearchQueries to be non-nil")
		}
	})
}

func TestDoStream_FinishReason(t *testing.T) {
	t.Run("should extract finish reason from stream", func(t *testing.T) {
		server, _ := createSSETestServer(perplexityTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

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
			t.Errorf("expected unified finish reason 'stop', got %q", finishPart.FinishReason.Unified)
		}
		if finishPart.FinishReason.Raw == nil || *finishPart.FinishReason.Raw != "stop" {
			t.Errorf("expected raw finish reason 'stop', got %v", finishPart.FinishReason.Raw)
		}
	})
}

func TestDoStream_ResponseMetadata(t *testing.T) {
	t.Run("should stream response metadata", func(t *testing.T) {
		server, _ := createSSETestServer(perplexityTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Find response metadata part
		var metadataPart *languagemodel.StreamPartResponseMetadata
		for _, part := range parts {
			if mp, ok := part.(languagemodel.StreamPartResponseMetadata); ok {
				metadataPart = &mp
			}
		}
		if metadataPart == nil {
			t.Fatal("expected response metadata part in stream")
		}
		if metadataPart.ID == nil || *metadataPart.ID != "a3d55d44-63f9-4704-bb26-e17be1ddab3a" {
			t.Errorf("expected ID 'a3d55d44-63f9-4704-bb26-e17be1ddab3a', got %v", metadataPart.ID)
		}
		if metadataPart.ModelID == nil || *metadataPart.ModelID != "sonar" {
			t.Errorf("expected ModelID 'sonar', got %v", metadataPart.ModelID)
		}
	})
}

func TestDoGenerate_FinishReason(t *testing.T) {
	t.Run("should extract finish reason", func(t *testing.T) {
		server, _ := createTestServer(perplexityTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected unified finish reason 'stop', got %q", result.FinishReason.Unified)
		}
		if result.FinishReason.Raw == nil || *result.FinishReason.Raw != "stop" {
			t.Errorf("expected raw finish reason 'stop', got %v", result.FinishReason.Raw)
		}
	})
}
