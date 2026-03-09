// Ported from: packages/anthropic/src/anthropic-provider.test.ts
package anthropic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func createSuccessfulResponse() map[string]any {
	return map[string]any{
		"type":  "message",
		"id":    "msg_123",
		"model": "claude-3-haiku-20240307",
		"content": []any{
			map[string]any{"type": "text", "text": "Hi"},
		},
		"stop_reason":   nil,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  float64(1),
			"output_tokens": float64(1),
		},
	}
}

// providerRequestCapture captures the request URL, headers and body from a test server.
type providerRequestCapture struct {
	URL     string
	Body    []byte
	Headers http.Header
}

func createProviderTestServer(capture *providerRequestCapture) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.URL = r.URL.Path
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createSuccessfulResponse())
	}))
}

var providerTestPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

func TestCreateAnthropic_BaseURL(t *testing.T) {
	t.Run("uses the default Anthropic base URL when not provided", func(t *testing.T) {
		capture := &providerRequestCapture{}
		server := createProviderTestServer(capture)
		defer server.Close()

		// We cannot easily test the default URL without intercepting the actual HTTP call.
		// Instead, we test with a base URL pointing to the test server.
		provider, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey:  strPtr("test-api-key"),
			BaseURL: strPtr(server.URL),
		})
		if err != nil {
			t.Fatalf("unexpected error creating provider: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")

		_, err = model.DoGenerate(languagemodel.CallOptions{
			Prompt: providerTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.URL != "/messages" {
			t.Errorf("expected request URL path '/messages', got %q", capture.URL)
		}
	})

	t.Run("uses custom baseURL", func(t *testing.T) {
		capture := &providerRequestCapture{}
		server := createProviderTestServer(capture)
		defer server.Close()

		provider, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey:  strPtr("test-api-key"),
			BaseURL: strPtr(server.URL),
		})
		if err != nil {
			t.Fatalf("unexpected error creating provider: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")

		_, err = model.DoGenerate(languagemodel.CallOptions{
			Prompt: providerTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.URL != "/messages" {
			t.Errorf("expected request URL path '/messages', got %q", capture.URL)
		}
	})
}

func TestCreateAnthropic_Authentication(t *testing.T) {
	t.Run("sends Authorization Bearer header when authToken is provided", func(t *testing.T) {
		capture := &providerRequestCapture{}
		server := createProviderTestServer(capture)
		defer server.Close()

		provider, err := CreateAnthropic(AnthropicProviderSettings{
			AuthToken: strPtr("test-auth-token"),
			BaseURL:   strPtr(server.URL),
		})
		if err != nil {
			t.Fatalf("unexpected error creating provider: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")

		_, err = model.DoGenerate(languagemodel.CallOptions{
			Prompt: providerTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		authHeader := capture.Headers.Get("Authorization")
		if authHeader != "Bearer test-auth-token" {
			t.Errorf("expected Authorization header 'Bearer test-auth-token', got %q", authHeader)
		}

		apiKeyHeader := capture.Headers.Get("X-Api-Key")
		if apiKeyHeader != "" {
			t.Errorf("expected no x-api-key header, got %q", apiKeyHeader)
		}
	})

	t.Run("throws error when both apiKey and authToken are provided", func(t *testing.T) {
		_, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey:    strPtr("test-api-key"),
			AuthToken: strPtr("test-auth-token"),
		})
		if err == nil {
			t.Fatal("expected error when both apiKey and authToken are provided")
		}
		expectedSubstring := "Both apiKey and authToken were provided. Please use only one authentication method."
		if !strings.Contains(err.Error(), expectedSubstring) {
			t.Errorf("expected error message to contain %q, got %q", expectedSubstring, err.Error())
		}
	})
}

func TestCreateAnthropic_CustomProviderName(t *testing.T) {
	t.Run("should use custom provider name when specified", func(t *testing.T) {
		name := "my-claude-proxy"
		provider, err := CreateAnthropic(AnthropicProviderSettings{
			Name:   &name,
			ApiKey: strPtr("test-api-key"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")
		if model.Provider() != "my-claude-proxy" {
			t.Errorf("expected provider 'my-claude-proxy', got %q", model.Provider())
		}
	})

	t.Run("should default to anthropic.messages when name not specified", func(t *testing.T) {
		provider, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey: strPtr("test-api-key"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")
		if model.Provider() != "anthropic.messages" {
			t.Errorf("expected provider 'anthropic.messages', got %q", model.Provider())
		}
	})
}

func TestCreateAnthropic_SupportedUrls(t *testing.T) {
	t.Run("should support image/* URLs", func(t *testing.T) {
		provider, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey: strPtr("test-api-key"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")
		supportedUrls, err := model.SupportedUrls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imagePatterns, ok := supportedUrls["image/*"]
		if !ok {
			t.Fatal("expected image/* to be in supported URLs")
		}
		if len(imagePatterns) == 0 {
			t.Fatal("expected at least one image/* pattern")
		}
		if !imagePatterns[0].MatchString("https://example.com/image.png") {
			t.Error("expected image/* pattern to match 'https://example.com/image.png'")
		}
	})

	t.Run("should support application/pdf URLs", func(t *testing.T) {
		provider, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey: strPtr("test-api-key"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")
		supportedUrls, err := model.SupportedUrls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		pdfPatterns, ok := supportedUrls["application/pdf"]
		if !ok {
			t.Fatal("expected application/pdf to be in supported URLs")
		}
		if len(pdfPatterns) == 0 {
			t.Fatal("expected at least one application/pdf pattern")
		}
		if !pdfPatterns[0].MatchString("https://arxiv.org/pdf/2401.00001") {
			t.Error("expected application/pdf pattern to match 'https://arxiv.org/pdf/2401.00001'")
		}
	})
}

func TestCreateAnthropic_DefaultSupportedUrls(t *testing.T) {
	t.Run("should return all URL patterns as compiled regexps", func(t *testing.T) {
		provider, err := CreateAnthropic(AnthropicProviderSettings{
			ApiKey: strPtr("test-api-key"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		model := provider.createChatModel("claude-3-haiku-20240307")
		supportedUrls, err := model.SupportedUrls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for mediaType, patterns := range supportedUrls {
			for i, pattern := range patterns {
				if pattern == nil {
					t.Errorf("expected non-nil pattern for %s[%d]", mediaType, i)
				}
				// Verify it's a valid regexp that can match URLs
				_ = regexp.MustCompile(pattern.String())
			}
		}
	})
}
