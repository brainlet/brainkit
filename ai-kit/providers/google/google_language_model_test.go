// Ported from: packages/google/src/google-generative-ai-language-model.test.ts
package google

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

var testPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

var safetyRatings = []map[string]any{
	{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "probability": "NEGLIGIBLE"},
	{"category": "HARM_CATEGORY_HATE_SPEECH", "probability": "NEGLIGIBLE"},
	{"category": "HARM_CATEGORY_HARASSMENT", "probability": "NEGLIGIBLE"},
	{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "probability": "NEGLIGIBLE"},
}

type langRequestCapture struct {
	Body    []byte
	Headers http.Header
	URL     string
}

func (rc *langRequestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

func createLangTestServer(fixture map[string]any, headers map[string]string) (*httptest.Server, *langRequestCapture) {
	capture := &langRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		capture.URL = r.URL.String()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fixture)
	}))
	return server, capture
}

func createSSETestServer(chunks []string, headers map[string]string) (*httptest.Server, *langRequestCapture) {
	capture := &langRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		capture.URL = r.URL.String()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	return server, capture
}

func createTestLangModel(baseURL string) *GoogleLanguageModel {
	return NewGoogleLanguageModel("gemini-pro", GoogleLanguageModelConfig{
		Provider: "google.generative-ai",
		BaseURL:  baseURL,
		Headers: func() map[string]string {
			return map[string]string{
				"x-goog-api-key": "test-api-key",
			}
		},
		GenerateID: func() string { return "test-id" },
	})
}

// Standard text response fixture
func googleTextFixture() map[string]any {
	text := "There are **3** r's in strawberry.\n\nHere is the breakdown: st**r**awbe**rr**y."
	return map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{
							"text":             text,
							"thoughtSignature": "EtoFCtcFAb4+9vtfe4MXRxQjw48U1WKrR/7lYsgFkVi/bepqsSPjY0VU7HEzkeCBIfy1fu5t9aUZ4IZ65aWagqbBrV45fc97olcg",
						},
					},
					"role": "model",
				},
				"finishReason": "STOP",
				"index":        float64(0),
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     float64(9),
			"candidatesTokenCount": float64(28),
			"totalTokenCount":      float64(281),
			"thoughtsTokenCount":   float64(244),
		},
		"modelVersion": "gemini-3-pro-preview",
	}
}

// Tool call fixture
func googleToolCallFixture() map[string]any {
	return map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{
							"functionCall": map[string]any{
								"name": "weather",
								"args": map[string]any{"location": "San Francisco"},
							},
							"thoughtSignature": "EskgCsYgAb4+9vtF7/499YQS2bjZs3xcQI",
						},
					},
					"role": "model",
				},
				"finishReason": "STOP",
				"index":        float64(0),
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     float64(29),
			"candidatesTokenCount": float64(15),
			"totalTokenCount":      float64(937),
			"thoughtsTokenCount":   float64(893),
		},
	}
}

func TestGoogleLanguageModel_DoGenerate_Text(t *testing.T) {
	t.Run("should extract text response", func(t *testing.T) {
		server, _ := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		found := false
		for _, c := range result.Content {
			if textPart, ok := c.(languagemodel.Text); ok {
				if strings.Contains(textPart.Text, "strawberry") {
					found = true
				}
			}
		}
		if !found {
			t.Error("expected text content containing 'strawberry'")
		}
	})

	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 9 {
			t.Errorf("expected inputTokens.Total 9, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 28+244 {
			t.Errorf("expected outputTokens.Total %d, got %v", 28+244, result.Usage.OutputTokens.Total)
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_MalformedFunctionCall(t *testing.T) {
	t.Run("should handle MALFORMED_FUNCTION_CALL finish reason and empty content", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content":      map[string]any{},
					"finishReason": "MALFORMED_FUNCTION_CALL",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount": float64(9056),
				"totalTokenCount":  float64(9056),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Content) != 0 {
			t.Errorf("expected 0 content, got %d", len(result.Content))
		}
		if result.FinishReason.Raw == nil || *result.FinishReason.Raw != "MALFORMED_FUNCTION_CALL" {
			t.Errorf("expected raw finish reason 'MALFORMED_FUNCTION_CALL', got %v", result.FinishReason.Raw)
		}
		if result.FinishReason.Unified != languagemodel.FinishReasonError {
			t.Errorf("expected unified finish reason 'error', got %q", result.FinishReason.Unified)
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ToolCall(t *testing.T) {
	t.Run("should extract tool calls", func(t *testing.T) {
		server, _ := createLangTestServer(googleToolCallFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "weather",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"location": map[string]any{"type": "string"}},
						"required":   []any{"location"},
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		foundToolCall := false
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.ToolCall); ok {
				if tc.ToolName == "weather" {
					foundToolCall = true
					if tc.ToolCallID != "test-id" {
						t.Errorf("expected toolCallId 'test-id', got %q", tc.ToolCallID)
					}
					if !strings.Contains(tc.Input, "San Francisco") {
						t.Errorf("expected input to contain 'San Francisco', got %q", tc.Input)
					}
				}
			}
		}
		if !foundToolCall {
			t.Error("expected to find a tool call for 'weather'")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ReasoningGemini3(t *testing.T) {
	t.Run("should extract reasoning with thoughtSignature", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text":             "There are **3** \"r\"s in strawberry.",
								"thoughtSignature": "EswFCskFAb4",
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(9),
				"candidatesTokenCount": float64(29),
				"totalTokenCount":      float64(296),
				"thoughtsTokenCount":   float64(258),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		foundText := false
		for _, c := range result.Content {
			if tp, ok := c.(languagemodel.Text); ok {
				if strings.Contains(tp.Text, "strawberry") {
					foundText = true
					// Verify thoughtSignature in provider metadata
					if tp.ProviderMetadata != nil {
						googleMeta, ok := tp.ProviderMetadata["google"]
						if ok {
							if googleMeta["thoughtSignature"] == nil {
								t.Error("expected thoughtSignature in provider metadata")
							}
						}
					}
				}
			}
		}
		if !foundText {
			t.Error("expected text content")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createLangTestServer(googleTextFixture(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if result.Response == nil {
			t.Fatal("expected response")
		}
		if result.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header, got headers: %v", result.Response.Headers)
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_RequestBody(t *testing.T) {
	t.Run("should pass the model, messages, and options", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		temp := float64(0.5)
		seed := 123
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "test system instruction"},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
			},
			Seed:        &seed,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()

		// Check system instruction
		sysInst, ok := body["systemInstruction"].(map[string]any)
		if !ok {
			t.Fatal("expected systemInstruction")
		}
		sysParts, ok := sysInst["parts"].([]any)
		if !ok || len(sysParts) == 0 {
			t.Fatal("expected systemInstruction parts")
		}
		sp := sysParts[0].(map[string]any)
		if sp["text"] != "test system instruction" {
			t.Errorf("expected system instruction text, got %v", sp["text"])
		}

		// Check generation config
		genConfig, ok := body["generationConfig"].(map[string]any)
		if !ok {
			t.Fatal("expected generationConfig")
		}
		if genConfig["temperature"] != float64(0.5) {
			t.Errorf("expected temperature 0.5, got %v", genConfig["temperature"])
		}
		if genConfig["seed"] != float64(123) {
			t.Errorf("expected seed 123, got %v", genConfig["seed"])
		}
	})

	t.Run("should send request body", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		contents, ok := body["contents"].([]any)
		if !ok || len(contents) == 0 {
			t.Fatal("expected contents")
		}
		c0 := contents[0].(map[string]any)
		if c0["role"] != "user" {
			t.Errorf("expected role 'user', got %v", c0["role"])
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should only pass valid provider options", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "test system instruction"},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
			},
			Seed: intPtr(123),
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{
					"foo":                "bar",
					"responseModalities": []any{"TEXT", "IMAGE"},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		genConfig := body["generationConfig"].(map[string]any)
		modalities, ok := genConfig["responseModalities"].([]any)
		if !ok {
			t.Fatal("expected responseModalities in generationConfig")
		}
		if len(modalities) != 2 || modalities[0] != "TEXT" || modalities[1] != "IMAGE" {
			t.Errorf("expected ['TEXT', 'IMAGE'], got %v", modalities)
		}
	})

	t.Run("should pass thinkingConfig in provider options", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{
					"thinkingConfig": map[string]any{
						"thinkingLevel": "high",
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		genConfig := body["generationConfig"].(map[string]any)
		tc, ok := genConfig["thinkingConfig"].(map[string]any)
		if !ok {
			t.Fatal("expected thinkingConfig")
		}
		if tc["thinkingLevel"] != "high" {
			t.Errorf("expected thinkingLevel 'high', got %v", tc["thinkingLevel"])
		}
	})

	t.Run("should pass mediaResolution in provider options", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{
					"mediaResolution": "MEDIA_RESOLUTION_LOW",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		genConfig := body["generationConfig"].(map[string]any)
		if genConfig["mediaResolution"] != "MEDIA_RESOLUTION_LOW" {
			t.Errorf("expected mediaResolution 'MEDIA_RESOLUTION_LOW', got %v", genConfig["mediaResolution"])
		}
	})

	t.Run("should pass imageConfig in provider options", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{
					"imageConfig": map[string]any{
						"aspectRatio": "16:9",
						"imageSize":   "2K",
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		genConfig := body["generationConfig"].(map[string]any)
		imgConfig, ok := genConfig["imageConfig"].(map[string]any)
		if !ok {
			t.Fatal("expected imageConfig")
		}
		if imgConfig["aspectRatio"] != "16:9" {
			t.Errorf("expected aspectRatio '16:9', got %v", imgConfig["aspectRatio"])
		}
		if imgConfig["imageSize"] != "2K" {
			t.Errorf("expected imageSize '2K', got %v", imgConfig["imageSize"])
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ToolsAndToolChoice(t *testing.T) {
	t.Run("should pass tools and toolChoice", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"value": map[string]any{"type": "string"}},
						"required":   []any{"value"},
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "test-tool"},
			Prompt:     testPrompt,
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()

		// Check tools
		tools, ok := body["tools"].([]any)
		if !ok || len(tools) == 0 {
			t.Fatal("expected tools")
		}
		tool0 := tools[0].(map[string]any)
		funcDecls, ok := tool0["functionDeclarations"].([]any)
		if !ok || len(funcDecls) == 0 {
			t.Fatal("expected functionDeclarations")
		}
		decl := funcDecls[0].(map[string]any)
		if decl["name"] != "test-tool" {
			t.Errorf("expected tool name 'test-tool', got %v", decl["name"])
		}

		// Check toolConfig
		toolConfig, ok := body["toolConfig"].(map[string]any)
		if !ok {
			t.Fatal("expected toolConfig")
		}
		fcc, ok := toolConfig["functionCallingConfig"].(map[string]any)
		if !ok {
			t.Fatal("expected functionCallingConfig")
		}
		if fcc["mode"] != "ANY" {
			t.Errorf("expected mode 'ANY', got %v", fcc["mode"])
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ResponseFormat(t *testing.T) {
	t.Run("should set response mime type with responseFormat", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"location": map[string]any{"type": "string"}},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		genConfig := body["generationConfig"].(map[string]any)
		if genConfig["responseMimeType"] != "application/json" {
			t.Errorf("expected responseMimeType 'application/json', got %v", genConfig["responseMimeType"])
		}
		schema, ok := genConfig["responseSchema"].(map[string]any)
		if !ok {
			t.Fatal("expected responseSchema")
		}
		if schema["type"] != "object" {
			t.Errorf("expected type 'object', got %v", schema["type"])
		}
	})

	t.Run("should not pass schema with structuredOutputs false", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{
					"structuredOutputs": false,
				},
			},
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"property1": map[string]any{"type": "string"}},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		genConfig := body["generationConfig"].(map[string]any)
		if genConfig["responseMimeType"] != "application/json" {
			t.Errorf("expected responseMimeType 'application/json', got %v", genConfig["responseMimeType"])
		}
		if genConfig["responseSchema"] != nil {
			t.Error("expected responseSchema to be nil when structuredOutputs is false")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_Headers(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createLangTestServer(googleTextFixture(), nil)
		defer server.Close()

		model := NewGoogleLanguageModel("gemini-pro", GoogleLanguageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{
					"x-goog-api-key":        "test-api-key",
					"Custom-Provider-Header": "provider-header-value",
				}
			},
			GenerateID: func() string { return "test-id" },
		})

		reqHeaderVal := "request-header-value"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Headers: map[string]*string{
				"Custom-Request-Header": &reqHeaderVal,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Error("expected provider header")
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Error("expected request header")
		}
		if capture.Headers.Get("X-Goog-Api-Key") != "test-api-key" {
			t.Error("expected api key header")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_Sources(t *testing.T) {
	t.Run("should extract sources from grounding metadata", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "test response"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"groundingChunks": []any{
							map[string]any{
								"web": map[string]any{
									"uri":   "https://source.example.com",
									"title": "Source Title",
								},
							},
						},
					},
				},
			},
			"promptFeedback": map[string]any{"safetyRatings": safetyRatings},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(1),
				"candidatesTokenCount": float64(2),
				"totalTokenCount":      float64(3),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		foundSource := false
		for _, c := range result.Content {
			if src, ok := c.(languagemodel.SourceURL); ok {
				if src.URL == "https://source.example.com" && src.Title != nil && *src.Title == "Source Title" {
					foundSource = true
				}
			}
		}
		if !foundSource {
			t.Error("expected source from grounding metadata")
		}
	})

	t.Run("should extract sources from maps grounding metadata", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "test response with Maps"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"groundingChunks": []any{
							map[string]any{
								"maps": map[string]any{
									"uri":     "https://maps.google.com/maps?cid=12345",
									"title":   "Best Italian Restaurant",
									"placeId": "ChIJ12345",
								},
							},
						},
					},
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(1),
				"candidatesTokenCount": float64(2),
				"totalTokenCount":      float64(3),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		foundSource := false
		for _, c := range result.Content {
			if src, ok := c.(languagemodel.SourceURL); ok {
				if src.URL == "https://maps.google.com/maps?cid=12345" {
					foundSource = true
				}
			}
		}
		if !foundSource {
			t.Error("expected source from maps grounding metadata")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_SafetyRatings(t *testing.T) {
	t.Run("should expose safety ratings in provider metadata", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "test response"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": []any{
						map[string]any{
							"category":         "HARM_CATEGORY_DANGEROUS_CONTENT",
							"probability":      "NEGLIGIBLE",
							"probabilityScore": 0.1,
							"severity":         "LOW",
							"severityScore":    0.2,
							"blocked":          false,
						},
					},
				},
			},
			"promptFeedback": map[string]any{"safetyRatings": safetyRatings},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		googleMeta, ok := result.ProviderMetadata["google"]
		if !ok {
			t.Fatal("expected 'google' in providerMetadata")
		}
		ratings := googleMeta["safetyRatings"]
		if ratings == nil {
			t.Error("expected safetyRatings in provider metadata")
		}
	})

	t.Run("should expose PromptFeedback in provider metadata", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "No"}},
						"role":  "model",
					},
					"finishReason": "SAFETY",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"promptFeedback": map[string]any{
				"blockReason":  "SAFETY",
				"safetyRatings": safetyRatings,
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		googleMeta := result.ProviderMetadata["google"]
		pf := googleMeta["promptFeedback"]
		if pf == nil {
			t.Error("expected promptFeedback in provider metadata")
		}
	})

	t.Run("should expose grounding metadata in provider metadata", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "test response"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"webSearchQueries": []any{"What's the weather in Chicago this weekend?"},
						"groundingChunks": []any{
							map[string]any{
								"web": map[string]any{
									"uri":   "https://example.com/weather",
									"title": "Chicago Weather Forecast",
								},
							},
						},
					},
				},
			},
			"promptFeedback": map[string]any{"safetyRatings": safetyRatings},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(1),
				"candidatesTokenCount": float64(2),
				"totalTokenCount":      float64(3),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		googleMeta := result.ProviderMetadata["google"]
		gm := googleMeta["groundingMetadata"]
		if gm == nil {
			t.Error("expected groundingMetadata in provider metadata")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_CodeExecution(t *testing.T) {
	t.Run("should handle code execution tool calls", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"executableCode": map[string]any{
									"language": "PYTHON",
									"code":     "print(1+1)",
								},
							},
							map[string]any{
								"codeExecutionResult": map[string]any{
									"outcome": "OUTCOME_OK",
									"output":  "2",
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.code_execution",
					Name: "code_execution",
					Args: map[string]any{},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		foundToolCall := false
		foundToolResult := false
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.ToolCall); ok {
				if tc.ToolName == "code_execution" {
					foundToolCall = true
					if tc.ProviderExecuted == nil || !*tc.ProviderExecuted {
						t.Error("expected providerExecuted to be true")
					}
					if !strings.Contains(tc.Input, "PYTHON") {
						t.Errorf("expected input to contain 'PYTHON', got %q", tc.Input)
					}
				}
			}
			if tr, ok := c.(languagemodel.ToolResult); ok {
				if tr.ToolName == "code_execution" {
					foundToolResult = true
					resultMap, ok := tr.Result.(map[string]any)
					if !ok {
						t.Fatalf("expected map result, got %T", tr.Result)
					}
					if resultMap["outcome"] != "OUTCOME_OK" {
						t.Errorf("expected outcome OUTCOME_OK, got %v", resultMap["outcome"])
					}
				}
			}
		}
		if !foundToolCall {
			t.Error("expected code_execution tool call")
		}
		if !foundToolResult {
			t.Error("expected code_execution tool result")
		}
	})

	t.Run("should return stop finish reason for code execution (provider-executed tool)", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"executableCode": map[string]any{
									"language": "PYTHON",
									"code":     "print(1+1)",
								},
							},
							map[string]any{
								"codeExecutionResult": map[string]any{
									"outcome": "OUTCOME_OK",
									"output":  "2",
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.code_execution",
					Name: "code_execution",
					Args: map[string]any{},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Provider-executed tools should not trigger 'tool-calls' finish reason
		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected unified finish reason 'stop', got %q", result.FinishReason.Unified)
		}
	})

	t.Run("should return tool-calls finish reason when code execution is combined with function tools", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"executableCode": map[string]any{
									"language": "PYTHON",
									"code":     "print(1+1)",
								},
							},
							map[string]any{
								"codeExecutionResult": map[string]any{
									"outcome": "OUTCOME_OK",
									"output":  "2",
								},
							},
							map[string]any{
								"functionCall": map[string]any{
									"name": "test-tool",
									"args": map[string]any{"value": "test"},
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
				},
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.code_execution",
					Name: "code_execution",
					Args: map[string]any{},
				},
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"value": map[string]any{"type": "string"}},
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected unified finish reason 'tool-calls', got %q", result.FinishReason.Unified)
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ImageOutputs(t *testing.T) {
	t.Run("should extract image file outputs", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "Here is an image:"},
							map[string]any{
								"inlineData": map[string]any{
									"mimeType": "image/jpeg",
									"data":     "base64encodedimagedata",
								},
							},
							map[string]any{"text": "And another image:"},
							map[string]any{
								"inlineData": map[string]any{
									"mimeType": "image/png",
									"data":     "anotherbase64encodedimagedata",
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"promptFeedback": map[string]any{"safetyRatings": safetyRatings},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
				"totalTokenCount":      float64(30),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		fileCount := 0
		textCount := 0
		for _, c := range result.Content {
			if _, ok := c.(languagemodel.File); ok {
				fileCount++
			}
			if _, ok := c.(languagemodel.Text); ok {
				textCount++
			}
		}
		if fileCount != 2 {
			t.Errorf("expected 2 file parts, got %d", fileCount)
		}
		if textCount != 2 {
			t.Errorf("expected 2 text parts, got %d", textCount)
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_Reasoning(t *testing.T) {
	t.Run("should correctly parse and separate reasoning parts from text output", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "Visible text part 1. "},
							map[string]any{"text": "This is a thought process.", "thought": true},
							map[string]any{"text": "Visible text part 2."},
							map[string]any{"text": "Another internal thought.", "thought": true},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
				"totalTokenCount":      float64(30),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Content) < 4 {
			t.Fatalf("expected at least 4 content parts, got %d", len(result.Content))
		}

		// Part 0: text
		if tp, ok := result.Content[0].(languagemodel.Text); !ok || tp.Text != "Visible text part 1. " {
			t.Errorf("expected text 'Visible text part 1. ', got %v", result.Content[0])
		}
		// Part 1: reasoning
		if rp, ok := result.Content[1].(languagemodel.Reasoning); !ok || rp.Text != "This is a thought process." {
			t.Errorf("expected reasoning 'This is a thought process.', got %v", result.Content[1])
		}
		// Part 2: text
		if tp, ok := result.Content[2].(languagemodel.Text); !ok || tp.Text != "Visible text part 2." {
			t.Errorf("expected text 'Visible text part 2.', got %v", result.Content[2])
		}
		// Part 3: reasoning
		if rp, ok := result.Content[3].(languagemodel.Reasoning); !ok || rp.Text != "Another internal thought." {
			t.Errorf("expected reasoning 'Another internal thought.', got %v", result.Content[3])
		}
	})

	t.Run("should correctly parse thought signatures with reasoning parts", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "Visible text part 1. ", "thoughtSignature": "sig1"},
							map[string]any{"text": "This is a thought process.", "thought": true, "thoughtSignature": "sig2"},
							map[string]any{"text": "Visible text part 2.", "thoughtSignature": "sig3"},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
				"totalTokenCount":      float64(30),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Check that thought signatures are in provider metadata
		for _, c := range result.Content {
			switch part := c.(type) {
			case languagemodel.Text:
				if part.ProviderMetadata != nil {
					googleMeta, ok := part.ProviderMetadata["google"]
					if ok {
						if googleMeta["thoughtSignature"] == nil {
							t.Error("expected thoughtSignature in text provider metadata")
						}
					}
				}
			case languagemodel.Reasoning:
				if part.ProviderMetadata != nil {
					googleMeta, ok := part.ProviderMetadata["google"]
					if ok {
						if googleMeta["thoughtSignature"] == nil {
							t.Error("expected thoughtSignature in reasoning provider metadata")
						}
					}
				}
			}
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_ProviderMetadataKey(t *testing.T) {
	t.Run("should use 'vertex' as providerMetadata key when provider includes 'vertex'", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Hello!"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"webSearchQueries": []any{"test query"},
					},
				},
			},
			"promptFeedback": map[string]any{"safetyRatings": safetyRatings},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(1),
				"candidatesTokenCount": float64(2),
				"totalTokenCount":      float64(3),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := NewGoogleLanguageModel("gemini-pro", GoogleLanguageModelConfig{
			Provider: "google.vertex.chat",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{"x-goog-api-key": "test-api-key"}
			},
			GenerateID: func() string { return "test-id" },
		})

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := result.ProviderMetadata["vertex"]; !ok {
			t.Error("expected 'vertex' key in providerMetadata")
		}
		if _, ok := result.ProviderMetadata["google"]; ok {
			t.Error("did not expect 'google' key when provider includes 'vertex'")
		}
	})

	t.Run("should use 'google' as providerMetadata key when provider does not include 'vertex'", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Hello!"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"webSearchQueries": []any{"test query"},
					},
				},
			},
			"promptFeedback": map[string]any{"safetyRatings": safetyRatings},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(1),
				"candidatesTokenCount": float64(2),
				"totalTokenCount":      float64(3),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		if _, ok := result.ProviderMetadata["google"]; !ok {
			t.Error("expected 'google' key in providerMetadata")
		}
		if _, ok := result.ProviderMetadata["vertex"]; ok {
			t.Error("did not expect 'vertex' key for non-vertex provider")
		}
	})
}

func TestGoogleLanguageModel_DoGenerate_IncludeThoughts(t *testing.T) {
	t.Run("should support includeThoughts with google generative ai provider", func(t *testing.T) {
		fixture := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text":             "let me think about this problem",
								"thought":          true,
								"thoughtSignature": "reasoning_sig",
							},
							map[string]any{"text": "the answer is 42"},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"safetyRatings": safetyRatings,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(15),
				"totalTokenCount":      float64(25),
				"thoughtsTokenCount":   float64(8),
			},
		}

		server, _ := createLangTestServer(fixture, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{
					"thinkingConfig": map[string]any{
						"includeThoughts": true,
						"thinkingBudget":  float64(1024),
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Check that we have a reasoning part and a text part
		hasReasoning := false
		hasText := false
		for _, c := range result.Content {
			if rp, ok := c.(languagemodel.Reasoning); ok && rp.Text == "let me think about this problem" {
				hasReasoning = true
			}
			if tp, ok := c.(languagemodel.Text); ok && tp.Text == "the answer is 42" {
				hasText = true
			}
		}
		if !hasReasoning {
			t.Error("expected reasoning content part")
		}
		if !hasText {
			t.Error("expected text content part")
		}

		// Check usage includes reasoning tokens
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 8 {
			t.Errorf("expected outputTokens.Reasoning 8, got %v", result.Usage.OutputTokens.Reasoning)
		}
	})
}

// ---- DoStream tests ----

func collectStreamParts(ch <-chan languagemodel.StreamPart) []languagemodel.StreamPart {
	var parts []languagemodel.StreamPart
	for p := range ch {
		parts = append(parts, p)
	}
	return parts
}

func TestGoogleLanguageModel_DoStream_Text(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Hello"}},
						"role":  "model",
					},
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": " world"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(5),
				"totalTokenCount":      float64(15),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk1), string(chunk2)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)

		// Check for stream-start
		foundStreamStart := false
		foundTextDelta := false
		foundFinish := false
		for _, p := range parts {
			switch p.(type) {
			case languagemodel.StreamPartStreamStart:
				foundStreamStart = true
			case languagemodel.StreamPartTextDelta:
				foundTextDelta = true
			case languagemodel.StreamPartFinish:
				foundFinish = true
			}
		}
		if !foundStreamStart {
			t.Error("expected stream-start event")
		}
		if !foundTextDelta {
			t.Error("expected text-delta event")
		}
		if !foundFinish {
			t.Error("expected finish event")
		}
	})

	t.Run("should expose the raw response headers", func(t *testing.T) {
		chunk, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "hello"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
		})

		server, _ := createSSETestServer([]string{string(chunk)}, map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		// Consume stream
		collectStreamParts(result.Stream)

		if result.Response == nil {
			t.Fatal("expected response")
		}
		if result.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header, got headers: %v", result.Response.Headers)
		}
	})
}

func TestGoogleLanguageModel_DoStream_ToolCall(t *testing.T) {
	t.Run("should stream tool call", func(t *testing.T) {
		chunk, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"functionCall": map[string]any{
									"name": "weather",
									"args": map[string]any{"location": "San Francisco"},
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(15),
				"totalTokenCount":      float64(25),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "weather",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"location": map[string]any{"type": "string"}},
						"required":   []any{"location"},
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		foundToolCall := false
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				if tc.ToolName == "weather" {
					foundToolCall = true
				}
			}
		}
		if !foundToolCall {
			t.Error("expected tool-call stream event")
		}
	})
}

func TestGoogleLanguageModel_DoStream_Sources(t *testing.T) {
	t.Run("should stream source events", func(t *testing.T) {
		chunk, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Some initial text"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"groundingChunks": []any{
							map[string]any{
								"web": map[string]any{
									"uri":   "https://source.example.com",
									"title": "Source Title",
								},
							},
						},
					},
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(294),
				"candidatesTokenCount": float64(233),
				"totalTokenCount":      float64(527),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		foundSource := false
		for _, p := range parts {
			if src, ok := p.(languagemodel.SourceURL); ok {
				if src.URL == "https://source.example.com" {
					foundSource = true
				}
			}
		}
		if !foundSource {
			t.Error("expected source stream event")
		}
	})

	t.Run("should preserve grounding metadata when it arrives before the finishReason chunk", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "hello"}},
						"role":  "model",
					},
					"index": float64(0),
					"groundingMetadata": map[string]any{
						"webSearchQueries": []any{"super bowl 2026 halftime show"},
						"groundingChunks": []any{
							map[string]any{
								"web": map[string]any{
									"uri":   "https://example.com/superbowl",
									"title": "Super Bowl 2026",
								},
							},
						},
					},
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": " world"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(38),
				"candidatesTokenCount": float64(1335),
				"totalTokenCount":      float64(1890),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk1), string(chunk2)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)

		// Check that grounding metadata is preserved in finish event
		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.ProviderMetadata != nil {
					googleMeta, ok := f.ProviderMetadata["google"]
					if ok {
						if googleMeta["groundingMetadata"] == nil {
							t.Error("expected groundingMetadata in finish event provider metadata")
						}
					}
				}
			}
		}

		// Check that source events were emitted
		sourceCount := 0
		for _, p := range parts {
			if _, ok := p.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		if sourceCount != 1 {
			t.Errorf("expected 1 source event, got %d", sourceCount)
		}
	})

	t.Run("should deduplicate sources across chunks", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "first chunk"}},
						"role":  "model",
					},
					"index":        float64(0),
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"groundingChunks": []any{
							map[string]any{"web": map[string]any{"uri": "https://example.com", "title": "Example"}},
							map[string]any{"web": map[string]any{"uri": "https://unique.com", "title": "Unique"}},
						},
					},
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "second chunk"}},
						"role":  "model",
					},
					"index":        float64(0),
					"safetyRatings": safetyRatings,
					"groundingMetadata": map[string]any{
						"groundingChunks": []any{
							map[string]any{"web": map[string]any{"uri": "https://example.com", "title": "Example Duplicate"}},
							map[string]any{"web": map[string]any{"uri": "https://another.com", "title": "Another"}},
						},
					},
				},
			},
		})
		chunk3, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "final chunk"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
		})

		server, _ := createSSETestServer([]string{string(chunk1), string(chunk2), string(chunk3)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		sourceCount := 0
		for _, p := range parts {
			if _, ok := p.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		// Should deduplicate: example.com appears twice but should only emit once
		if sourceCount != 3 {
			t.Errorf("expected 3 deduplicated source events, got %d", sourceCount)
		}
	})
}

func TestGoogleLanguageModel_DoStream_FinishReason(t *testing.T) {
	t.Run("should set finishReason to tool-calls when chunk contains functionCall", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "Initial text response"}},
						"role":  "model",
					},
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"functionCall": map[string]any{
									"name": "test-tool",
									"args": map[string]any{"value": "example value"},
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
				"totalTokenCount":      float64(30),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk1), string(chunk2)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"value": map[string]any{"type": "string"}},
						"required":   []any{"value"},
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
					t.Errorf("expected unified finish reason 'tool-calls', got %q", f.FinishReason.Unified)
				}
			}
		}
	})
}

func TestGoogleLanguageModel_DoStream_CodeExecution(t *testing.T) {
	t.Run("should stream code execution tool calls and results", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"executableCode": map[string]any{
									"language": "PYTHON",
									"code":     `print("hello")`,
								},
							},
						},
					},
				},
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"codeExecutionResult": map[string]any{
									"outcome": "OUTCOME_OK",
									"output":  "hello\n",
								},
							},
						},
					},
					"finishReason": "STOP",
				},
			},
		})

		server, _ := createSSETestServer([]string{string(chunk1), string(chunk2)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.code_execution",
					Name: "code_execution",
					Args: map[string]any{},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		foundToolCall := false
		foundToolResult := false
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				if tc.ToolName == "code_execution" {
					foundToolCall = true
				}
			}
			if tr, ok := p.(languagemodel.ToolResult); ok {
				if tr.ToolName == "code_execution" {
					foundToolResult = true
				}
			}
		}
		if !foundToolCall {
			t.Error("expected code_execution tool call in stream")
		}
		if !foundToolResult {
			t.Error("expected code_execution tool result in stream")
		}
	})
}

func TestGoogleLanguageModel_DoStream_Files(t *testing.T) {
	t.Run("should stream files", func(t *testing.T) {
		chunk, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"inlineData": map[string]any{
									"data":     "test",
									"mimeType": "text/plain",
								},
							},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": safetyRatings,
				},
			},
		})
		usageChunk, _ := json.Marshal(map[string]any{
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(294),
				"candidatesTokenCount": float64(233),
				"totalTokenCount":      float64(527),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk), string(usageChunk)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		foundFile := false
		for _, p := range parts {
			if f, ok := p.(languagemodel.File); ok {
				if f.MediaType == "text/plain" {
					foundFile = true
				}
			}
		}
		if !foundFile {
			t.Error("expected file stream event")
		}
	})
}

func TestGoogleLanguageModel_DoStream_Reasoning(t *testing.T) {
	t.Run("should stream reasoning parts separately from text parts", func(t *testing.T) {
		chunk1, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"text":    "I need to think about this carefully.",
								"thought": true,
							},
						},
						"role": "model",
					},
					"index": float64(0),
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":   float64(14),
				"totalTokenCount":    float64(84),
				"thoughtsTokenCount": float64(70),
			},
		})
		chunk2, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{"text": "Here is a simple explanation."},
						},
						"role": "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(14),
				"candidatesTokenCount": float64(8),
				"totalTokenCount":      float64(164),
				"thoughtsTokenCount":   float64(142),
			},
		})

		server, _ := createSSETestServer([]string{string(chunk1), string(chunk2)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		foundReasoning := false
		foundText := false
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartReasoningDelta); ok {
				foundReasoning = true
			}
			if _, ok := p.(languagemodel.StreamPartTextDelta); ok {
				foundText = true
			}
		}
		if !foundReasoning {
			t.Error("expected reasoning stream event")
		}
		if !foundText {
			t.Error("expected text stream event")
		}
	})
}

func TestGoogleLanguageModel_DoStream_SafetyRatings(t *testing.T) {
	t.Run("should expose safety ratings in provider metadata on finish", func(t *testing.T) {
		chunk, _ := json.Marshal(map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{map[string]any{"text": "test"}},
						"role":  "model",
					},
					"finishReason": "STOP",
					"index":        float64(0),
					"safetyRatings": []any{
						map[string]any{
							"category":         "HARM_CATEGORY_DANGEROUS_CONTENT",
							"probability":      "NEGLIGIBLE",
							"probabilityScore": 0.1,
							"severity":         "LOW",
							"severityScore":    0.2,
							"blocked":          false,
						},
					},
				},
			},
		})

		server, _ := createSSETestServer([]string{string(chunk)}, nil)
		defer server.Close()

		model := createTestLangModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}

		parts := collectStreamParts(result.Stream)
		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.ProviderMetadata != nil {
					googleMeta, ok := f.ProviderMetadata["google"]
					if ok {
						if googleMeta["safetyRatings"] == nil {
							t.Error("expected safetyRatings in finish event provider metadata")
						}
					}
				}
			}
		}
	})
}

