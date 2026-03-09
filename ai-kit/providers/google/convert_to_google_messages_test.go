// Ported from: packages/google/src/convert-to-google-generative-ai-messages.test.ts
package google

import (
	"reflect"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestConvertToGoogleMessages(t *testing.T) {
	t.Run("system messages", func(t *testing.T) {
		t.Run("should store system message in system instruction", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "Test"},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.SystemInstruction == nil {
				t.Fatal("expected systemInstruction to be set")
			}
			if len(result.SystemInstruction.Parts) != 1 || result.SystemInstruction.Parts[0].Text != "Test" {
				t.Errorf("unexpected systemInstruction: %+v", result.SystemInstruction)
			}
			if len(result.Contents) != 0 {
				t.Errorf("expected empty contents, got %d", len(result.Contents))
			}
		})

		t.Run("should throw error when there was already a user message", func(t *testing.T) {
			_, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.UserMessage{Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Test"},
					}},
					languagemodel.SystemMessage{Content: "Test"},
				},
				nil,
			)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "system messages are only supported at the beginning") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	})

	t.Run("thought signatures", func(t *testing.T) {
		t.Run("should preserve thought signatures in assistant messages", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.TextPart{
								Text:            "Regular text",
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig1"}},
							},
							languagemodel.ReasoningPart{
								Text:            "Reasoning text",
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig2"}},
							},
							languagemodel.ToolCallPart{
								ToolCallID:      "call1",
								ToolName:        "test",
								Input:           map[string]any{"value": "test"},
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig3"}},
							},
						},
					},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Contents) != 1 {
				t.Fatalf("expected 1 content, got %d", len(result.Contents))
			}
			content := result.Contents[0]
			if content.Role != "model" {
				t.Errorf("expected role 'model', got %q", content.Role)
			}
			if len(content.Parts) != 3 {
				t.Fatalf("expected 3 parts, got %d", len(content.Parts))
			}
			// Text part
			if content.Parts[0].Text == nil || *content.Parts[0].Text != "Regular text" {
				t.Errorf("unexpected text part: %+v", content.Parts[0])
			}
			if content.Parts[0].ThoughtSignature == nil || *content.Parts[0].ThoughtSignature != "sig1" {
				t.Errorf("expected thoughtSignature sig1")
			}
			// Reasoning part
			if content.Parts[1].Text == nil || *content.Parts[1].Text != "Reasoning text" {
				t.Errorf("unexpected reasoning part: %+v", content.Parts[1])
			}
			if content.Parts[1].Thought == nil || !*content.Parts[1].Thought {
				t.Error("expected thought=true")
			}
			if content.Parts[1].ThoughtSignature == nil || *content.Parts[1].ThoughtSignature != "sig2" {
				t.Errorf("expected thoughtSignature sig2")
			}
			// Tool call part
			if content.Parts[2].FunctionCall == nil || content.Parts[2].FunctionCall.Name != "test" {
				t.Errorf("unexpected function call: %+v", content.Parts[2])
			}
			if content.Parts[2].ThoughtSignature == nil || *content.Parts[2].ThoughtSignature != "sig3" {
				t.Errorf("expected thoughtSignature sig3")
			}
		})
	})

	t.Run("thought signatures with vertex providerOptionsName", func(t *testing.T) {
		t.Run("should resolve thoughtSignature from google namespace when using vertex providerOptionsName", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.TextPart{
								Text:            "Regular text",
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig1"}},
							},
							languagemodel.ReasoningPart{
								Text:            "Reasoning text",
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig2"}},
							},
							languagemodel.ToolCallPart{
								ToolCallID:      "call1",
								ToolName:        "getWeather",
								Input:           map[string]any{"location": "London"},
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig3"}},
							},
						},
					},
				},
				&ConvertToGoogleMessagesOptions{ProviderOptionsName: "vertex"},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Contents) != 1 || len(result.Contents[0].Parts) != 3 {
				t.Fatal("unexpected structure")
			}
			if result.Contents[0].Parts[0].ThoughtSignature == nil || *result.Contents[0].Parts[0].ThoughtSignature != "sig1" {
				t.Error("expected sig1")
			}
			if result.Contents[0].Parts[1].ThoughtSignature == nil || *result.Contents[0].Parts[1].ThoughtSignature != "sig2" {
				t.Error("expected sig2")
			}
			if result.Contents[0].Parts[2].ThoughtSignature == nil || *result.Contents[0].Parts[2].ThoughtSignature != "sig3" {
				t.Error("expected sig3")
			}
		})

		t.Run("should prefer vertex namespace over google namespace when both are present", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID: "call1",
								ToolName:   "getWeather",
								Input:      map[string]any{"location": "London"},
								ProviderOptions: map[string]map[string]any{
									"vertex": {"thoughtSignature": "vertex_sig"},
									"google": {"thoughtSignature": "google_sig"},
								},
							},
						},
					},
				},
				&ConvertToGoogleMessagesOptions{ProviderOptionsName: "vertex"},
			)
			if err != nil {
				t.Fatal(err)
			}
			part := result.Contents[0].Parts[0]
			if part.ThoughtSignature == nil || *part.ThoughtSignature != "vertex_sig" {
				t.Errorf("expected vertex_sig, got %v", part.ThoughtSignature)
			}
		})

		t.Run("should resolve thoughtSignature from vertex namespace directly", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID: "call1",
								ToolName:   "getWeather",
								Input:      map[string]any{"location": "London"},
								ProviderOptions: map[string]map[string]any{
									"vertex": {"thoughtSignature": "vertex_sig"},
								},
							},
						},
					},
				},
				&ConvertToGoogleMessagesOptions{ProviderOptionsName: "vertex"},
			)
			if err != nil {
				t.Fatal(err)
			}
			part := result.Contents[0].Parts[0]
			if part.ThoughtSignature == nil || *part.ThoughtSignature != "vertex_sig" {
				t.Errorf("expected vertex_sig, got %v", part.ThoughtSignature)
			}
		})
	})

	t.Run("thought signatures with google providerOptionsName (gateway failover)", func(t *testing.T) {
		t.Run("should resolve thoughtSignature from vertex namespace when using google providerOptionsName", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.TextPart{
								Text:            "Regular text",
								ProviderOptions: map[string]map[string]any{"vertex": {"thoughtSignature": "sig1"}},
							},
							languagemodel.ReasoningPart{
								Text:            "Reasoning text",
								ProviderOptions: map[string]map[string]any{"vertex": {"thoughtSignature": "sig2"}},
							},
							languagemodel.ToolCallPart{
								ToolCallID:      "call1",
								ToolName:        "getWeather",
								Input:           map[string]any{"location": "London"},
								ProviderOptions: map[string]map[string]any{"vertex": {"thoughtSignature": "sig3"}},
							},
						},
					},
				},
				&ConvertToGoogleMessagesOptions{ProviderOptionsName: "google"},
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.Contents[0].Parts[0].ThoughtSignature == nil || *result.Contents[0].Parts[0].ThoughtSignature != "sig1" {
				t.Error("expected sig1")
			}
			if result.Contents[0].Parts[1].ThoughtSignature == nil || *result.Contents[0].Parts[1].ThoughtSignature != "sig2" {
				t.Error("expected sig2")
			}
			if result.Contents[0].Parts[2].ThoughtSignature == nil || *result.Contents[0].Parts[2].ThoughtSignature != "sig3" {
				t.Error("expected sig3")
			}
		})

		t.Run("should prefer google namespace over vertex namespace when both are present", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID: "call1",
								ToolName:   "getWeather",
								Input:      map[string]any{"location": "London"},
								ProviderOptions: map[string]map[string]any{
									"google": {"thoughtSignature": "google_sig"},
									"vertex": {"thoughtSignature": "vertex_sig"},
								},
							},
						},
					},
				},
				&ConvertToGoogleMessagesOptions{ProviderOptionsName: "google"},
			)
			if err != nil {
				t.Fatal(err)
			}
			part := result.Contents[0].Parts[0]
			if part.ThoughtSignature == nil || *part.ThoughtSignature != "google_sig" {
				t.Errorf("expected google_sig, got %v", part.ThoughtSignature)
			}
		})

		t.Run("should resolve thoughtSignature from vertex namespace when google namespace is absent (default providerOptionsName)", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID: "call1",
								ToolName:   "getWeather",
								Input:      map[string]any{"location": "London"},
								ProviderOptions: map[string]map[string]any{
									"vertex": {"thoughtSignature": "vertex_sig"},
								},
							},
						},
					},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			part := result.Contents[0].Parts[0]
			if part.ThoughtSignature == nil || *part.ThoughtSignature != "vertex_sig" {
				t.Errorf("expected vertex_sig, got %v", part.ThoughtSignature)
			}
		})
	})

	t.Run("Gemma model system instructions", func(t *testing.T) {
		t.Run("should prepend system instruction to first user message for Gemma models", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "You are a helpful assistant."},
					languagemodel.UserMessage{Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					}},
				},
				&ConvertToGoogleMessagesOptions{IsGemmaModel: true},
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.SystemInstruction != nil {
				t.Error("expected nil systemInstruction for Gemma model")
			}
			if len(result.Contents) != 1 {
				t.Fatalf("expected 1 content, got %d", len(result.Contents))
			}
			if len(result.Contents[0].Parts) != 2 {
				t.Fatalf("expected 2 parts, got %d", len(result.Contents[0].Parts))
			}
			expected := "You are a helpful assistant.\n\n"
			if result.Contents[0].Parts[0].Text == nil || *result.Contents[0].Parts[0].Text != expected {
				t.Errorf("expected prepended system text %q, got %q", expected, *result.Contents[0].Parts[0].Text)
			}
			if result.Contents[0].Parts[1].Text == nil || *result.Contents[0].Parts[1].Text != "Hello" {
				t.Errorf("expected text 'Hello'")
			}
		})

		t.Run("should handle multiple system messages for Gemma models", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "You are helpful."},
					languagemodel.SystemMessage{Content: "Be concise."},
					languagemodel.UserMessage{Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hi"},
					}},
				},
				&ConvertToGoogleMessagesOptions{IsGemmaModel: true},
			)
			if err != nil {
				t.Fatal(err)
			}
			expected := "You are helpful.\n\nBe concise.\n\n"
			if result.Contents[0].Parts[0].Text == nil || *result.Contents[0].Parts[0].Text != expected {
				t.Errorf("expected %q, got %q", expected, *result.Contents[0].Parts[0].Text)
			}
		})

		t.Run("should not affect non-Gemma models", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "You are helpful."},
					languagemodel.UserMessage{Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					}},
				},
				&ConvertToGoogleMessagesOptions{IsGemmaModel: false},
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.SystemInstruction == nil {
				t.Fatal("expected systemInstruction")
			}
			if result.SystemInstruction.Parts[0].Text != "You are helpful." {
				t.Error("unexpected system instruction text")
			}
		})

		t.Run("should handle Gemma model with system instruction but no user messages", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.SystemMessage{Content: "You are helpful."},
				},
				&ConvertToGoogleMessagesOptions{IsGemmaModel: true},
			)
			if err != nil {
				t.Fatal(err)
			}
			if result.SystemInstruction != nil {
				t.Error("expected nil systemInstruction for Gemma model with no user messages")
			}
			if len(result.Contents) != 0 {
				t.Errorf("expected empty contents")
			}
		})
	})

	t.Run("user messages", func(t *testing.T) {
		t.Run("should add image parts for base64 encoded files", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.UserMessage{Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
							MediaType: "image/png",
						},
					}},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Contents) != 1 {
				t.Fatalf("expected 1 content, got %d", len(result.Contents))
			}
			part := result.Contents[0].Parts[0]
			if part.InlineData == nil {
				t.Fatal("expected inlineData")
			}
			if part.InlineData.Data != "AAECAw==" {
				t.Errorf("expected data 'AAECAw==', got %q", part.InlineData.Data)
			}
			if part.InlineData.MimeType != "image/png" {
				t.Errorf("expected mimeType 'image/png', got %q", part.InlineData.MimeType)
			}
		})
	})

	t.Run("tool messages", func(t *testing.T) {
		t.Run("should convert tool result messages to function responses", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.ToolMessage{Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolName:   "testFunction",
							ToolCallID: "testCallId",
							Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"someData": "test result"}},
						},
					}},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Contents) != 1 {
				t.Fatalf("expected 1 content, got %d", len(result.Contents))
			}
			content := result.Contents[0]
			if content.Role != "user" {
				t.Errorf("expected role 'user', got %q", content.Role)
			}
			part := content.Parts[0]
			if part.FunctionResponse == nil {
				t.Fatal("expected functionResponse")
			}
			if part.FunctionResponse.Name != "testFunction" {
				t.Errorf("expected name 'testFunction', got %q", part.FunctionResponse.Name)
			}
		})
	})

	t.Run("assistant messages", func(t *testing.T) {
		t.Run("should add PNG image parts for base64 encoded files", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.FilePart{
								Data:      languagemodel.DataContentString{Value: "AAECAw=="},
								MediaType: "image/png",
							},
						},
					},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			part := result.Contents[0].Parts[0]
			if part.InlineData == nil {
				t.Fatal("expected inlineData")
			}
			if part.InlineData.Data != "AAECAw==" || part.InlineData.MimeType != "image/png" {
				t.Errorf("unexpected inlineData: %+v", part.InlineData)
			}
		})

		t.Run("should throw error for URL file data in assistant messages", func(t *testing.T) {
			_, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.FilePart{
								Data:      languagemodel.DataContentString{Value: "https://example.com/image.png"},
								MediaType: "image/png",
							},
						},
					},
				},
				nil,
			)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "File data URLs in assistant messages are not supported") {
				t.Errorf("unexpected error: %v", err)
			}
		})

		t.Run("should convert tool result messages with content type (multipart with images)", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.ToolMessage{Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolName:   "imageGenerator",
							ToolCallID: "testCallId",
							Output: languagemodel.ToolResultOutputContent{
								Value: []languagemodel.ToolResultContentPart{
									languagemodel.ToolResultContentText{Text: "Here is the generated image:"},
									languagemodel.ToolResultContentImageData{
										Data:      "base64encodedimagedata",
										MediaType: "image/jpeg",
									},
								},
							},
						},
					}},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Contents) != 1 {
				t.Fatalf("expected 1 content, got %d", len(result.Contents))
			}
			parts := result.Contents[0].Parts
			if len(parts) != 3 {
				t.Fatalf("expected 3 parts, got %d", len(parts))
			}
			// FunctionResponse for text
			if parts[0].FunctionResponse == nil || parts[0].FunctionResponse.Name != "imageGenerator" {
				t.Error("expected functionResponse for text")
			}
			// InlineData for image
			if parts[1].InlineData == nil || parts[1].InlineData.MimeType != "image/jpeg" {
				t.Error("expected inlineData for image")
			}
			// Text for image response
			if parts[2].Text == nil || *parts[2].Text != "Tool executed successfully and returned this image as a response" {
				t.Error("expected success text")
			}
		})
	})

	t.Run("parallel tool calls", func(t *testing.T) {
		t.Run("should include thought signature on functionCall when provided", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID:      "call1",
								ToolName:        "checkweather",
								Input:           map[string]any{"city": "paris"},
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig_parallel"}},
							},
							languagemodel.ToolCallPart{
								ToolCallID: "call2",
								ToolName:   "checkweather",
								Input:      map[string]any{"city": "london"},
							},
						},
					},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			parts := result.Contents[0].Parts
			if parts[0].ThoughtSignature == nil || *parts[0].ThoughtSignature != "sig_parallel" {
				t.Error("expected sig_parallel on first part")
			}
			if parts[1].ThoughtSignature != nil {
				t.Error("expected nil thoughtSignature on second part")
			}
		})
	})

	t.Run("tool results with thought signatures", func(t *testing.T) {
		t.Run("should include thought signature on functionCall but not on functionResponse", func(t *testing.T) {
			result, err := ConvertToGoogleMessages(
				languagemodel.Prompt{
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID:      "call1",
								ToolName:        "readdata",
								Input:           map[string]any{"userId": "123"},
								ProviderOptions: map[string]map[string]any{"google": {"thoughtSignature": "sig_original"}},
							},
						},
					},
					languagemodel.ToolMessage{Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call1",
							ToolName:   "readdata",
							Output:     languagemodel.ToolResultOutputText{Value: "file not found"},
						},
					}},
				},
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			// Assistant message
			assistantPart := result.Contents[0].Parts[0]
			if assistantPart.ThoughtSignature == nil || *assistantPart.ThoughtSignature != "sig_original" {
				t.Error("expected sig_original on assistant part")
			}
			// Tool result message
			toolPart := result.Contents[1].Parts[0]
			if toolPart.FunctionResponse == nil {
				t.Fatal("expected functionResponse")
			}
			if toolPart.ThoughtSignature != nil {
				t.Error("expected nil thoughtSignature on tool result")
			}
		})
	})
}

// Verify the types match what we expect - used as compile-time check
var _ = reflect.DeepEqual
