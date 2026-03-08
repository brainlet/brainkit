// Ported from: packages/ai/src/middleware/extract-json-middleware.test.ts
package middleware

import (
	"strings"
	"testing"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// --- wrapGenerate tests ---

func TestExtractJson_WrapGenerate_StripJsonFence(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "```json\n{\"value\": \"test\"}\n```"},
				},
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := result.Content[0].(lm.Text).Text
	if text != `{"value": "test"}` {
		t.Errorf("expected stripped text, got %q", text)
	}
}

func TestExtractJson_WrapGenerate_StripPlainFence(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "```\n{\"value\": \"test\"}\n```"},
				},
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := result.Content[0].(lm.Text).Text
	if text != `{"value": "test"}` {
		t.Errorf("expected stripped text, got %q", text)
	}
}

func TestExtractJson_WrapGenerate_LeaveUnfencedUnchanged(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: `{"value": "test"}`},
				},
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := result.Content[0].(lm.Text).Text
	if text != `{"value": "test"}` {
		t.Errorf("expected unchanged text, got %q", text)
	}
}

func TestExtractJson_WrapGenerate_CustomTransform(t *testing.T) {
	m := ExtractJsonMiddleware(&ExtractJsonMiddlewareOptions{
		Transform: func(text string) string {
			text = strings.Replace(text, "PREFIX", "", 1)
			text = strings.Replace(text, "SUFFIX", "", 1)
			return text
		},
	})

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "PREFIX{\"value\": \"test\"}SUFFIX"},
				},
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := result.Content[0].(lm.Text).Text
	if text != `{"value": "test"}` {
		t.Errorf("expected custom transformed text, got %q", text)
	}
}

func TestExtractJson_WrapGenerate_PreserveNonTextContent(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "```json\n{\"value\": \"test\"}\n```"},
					lm.Reasoning{Text: "some reasoning"},
				},
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(result.Content))
	}
	text := result.Content[0].(lm.Text).Text
	if text != `{"value": "test"}` {
		t.Errorf("expected stripped text, got %q", text)
	}
	reasoning := result.Content[1].(lm.Reasoning).Text
	if reasoning != "some reasoning" {
		t.Error("expected reasoning to be preserved")
	}
}

// --- wrapStream tests ---

func TestExtractJson_WrapStream_StripFence(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "```json\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: `{"value": "test"}`},
					lm.StreamPartTextDelta{ID: "1", Delta: "\n```"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != `{"value": "test"}` {
		t.Errorf("expected stripped text, got %q", text)
	}
}

func TestExtractJson_WrapStream_StripPlainFence(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "```\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: `{"value": "test"}`},
					lm.StreamPartTextDelta{ID: "1", Delta: "\n```"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != `{"value": "test"}` {
		t.Errorf("expected stripped text, got %q", text)
	}
}

func TestExtractJson_WrapStream_LeaveUnfencedUnchanged(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: `{"value": "test"}`},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != `{"value": "test"}` {
		t.Errorf("expected unchanged text, got %q", text)
	}
}

func TestExtractJson_WrapStream_FenceSplitAcrossDeltas(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "`"},
					lm.StreamPartTextDelta{ID: "1", Delta: "``"},
					lm.StreamPartTextDelta{ID: "1", Delta: "json\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: `{"value": "test"}`},
					lm.StreamPartTextDelta{ID: "1", Delta: "\n`"},
					lm.StreamPartTextDelta{ID: "1", Delta: "``"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != `{"value": "test"}` {
		t.Errorf("expected stripped text, got %q", text)
	}
}

func TestExtractJson_WrapStream_NotAFence(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "`code`"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != "`code`" {
		t.Errorf("expected `code`, got %q", text)
	}
}

func TestExtractJson_WrapStream_PassThroughNonTextChunks(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "```json\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: `{"value": "test"}`},
					lm.StreamPartTextDelta{ID: "1", Delta: "\n```"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartToolInputStart{ID: "tool-1", ToolName: "testTool"},
					lm.StreamPartToolInputDelta{ID: "tool-1", Delta: `{"arg": "value"}`},
					lm.StreamPartToolInputEnd{ID: "tool-1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)

	var hasToolInputStart bool
	for _, p := range parts {
		if _, ok := p.(lm.StreamPartToolInputStart); ok {
			hasToolInputStart = true
		}
	}
	if !hasToolInputStart {
		t.Error("expected ToolInputStart to be passed through")
	}
}

func TestExtractJson_WrapStream_ContentWithoutFenceStarts(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "{"},
					lm.StreamPartTextDelta{ID: "1", Delta: `"value": "test"`},
					lm.StreamPartTextDelta{ID: "1", Delta: "}"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != `{"value": "test"}` {
		t.Errorf("expected unchanged JSON text, got %q", text)
	}
}

func TestExtractJson_WrapStream_LargeContent(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	largeJSON := `{"data":"` + strings.Repeat("x", 100) + `"}`

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "```json\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: largeJSON},
					lm.StreamPartTextDelta{ID: "1", Delta: "\n```"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != largeJSON {
		t.Errorf("expected large JSON, got text of length %d", len(text))
	}
}

func TestExtractJson_WrapStream_EmptyBetweenFences(t *testing.T) {
	m := ExtractJsonMiddleware(nil)

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "```json\n```"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
}

func TestExtractJson_WrapStream_CustomTransform(t *testing.T) {
	m := ExtractJsonMiddleware(&ExtractJsonMiddlewareOptions{
		Transform: func(text string) string {
			text = strings.Replace(text, "PREFIX", "", 1)
			text = strings.Replace(text, "SUFFIX", "", 1)
			return text
		},
	})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "PREFIX"},
					lm.StreamPartTextDelta{ID: "1", Delta: `{"value": "test"}`},
					lm.StreamPartTextDelta{ID: "1", Delta: "SUFFIX"},
					lm.StreamPartTextEnd{ID: "1"},
					lm.StreamPartFinish{FinishReason: lm.FinishReason{Unified: "stop"}},
				}),
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)
	text := collectTextDeltas(parts)
	if text != `{"value": "test"}` {
		t.Errorf("expected custom transformed text, got %q", text)
	}
}

// Helper to collect all text deltas from stream parts.
func collectTextDeltas(parts []lm.StreamPart) string {
	var sb strings.Builder
	for _, p := range parts {
		if td, ok := p.(lm.StreamPartTextDelta); ok {
			sb.WriteString(td.Delta)
		}
	}
	return sb.String()
}
