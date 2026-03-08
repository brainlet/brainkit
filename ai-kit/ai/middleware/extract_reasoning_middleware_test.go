// Ported from: packages/ai/src/middleware/extract-reasoning-middleware.test.ts
package middleware

import (
	"strings"
	"testing"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// --- wrapGenerate tests ---

func TestExtractReasoning_WrapGenerate_ExtractThinkTags(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "<think>analyzing the request</think>Here is the response"},
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

	reasoning, ok := result.Content[0].(lm.Reasoning)
	if !ok {
		t.Fatalf("expected Reasoning, got %T", result.Content[0])
	}
	if reasoning.Text != "analyzing the request" {
		t.Errorf("expected reasoning text, got %q", reasoning.Text)
	}

	text, ok := result.Content[1].(lm.Text)
	if !ok {
		t.Fatalf("expected Text, got %T", result.Content[1])
	}
	if text.Text != "Here is the response" {
		t.Errorf("expected text, got %q", text.Text)
	}
}

func TestExtractReasoning_WrapGenerate_ThinkTagsNoText(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "<think>analyzing the request\n</think>"},
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

	reasoning := result.Content[0].(lm.Reasoning)
	if reasoning.Text != "analyzing the request\n" {
		t.Errorf("expected reasoning text with newline, got %q", reasoning.Text)
	}

	text := result.Content[1].(lm.Text)
	if text.Text != "" {
		t.Errorf("expected empty text, got %q", text.Text)
	}
}

func TestExtractReasoning_WrapGenerate_MultipleThinkTags(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "<think>analyzing the request</think>Here is the response<think>thinking about the response</think>more"},
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

	reasoning := result.Content[0].(lm.Reasoning)
	if !strings.Contains(reasoning.Text, "analyzing the request") {
		t.Error("expected first reasoning")
	}
	if !strings.Contains(reasoning.Text, "thinking about the response") {
		t.Error("expected second reasoning")
	}

	text := result.Content[1].(lm.Text)
	if !strings.Contains(text.Text, "Here is the response") {
		t.Error("expected first text part")
	}
	if !strings.Contains(text.Text, "more") {
		t.Error("expected second text part")
	}
}

func TestExtractReasoning_WrapGenerate_StartWithReasoning(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{
		TagName:            "think",
		StartWithReasoning: true,
	})

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "analyzing the request</think>Here is the response"},
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

	reasoning := result.Content[0].(lm.Reasoning)
	if reasoning.Text != "analyzing the request" {
		t.Errorf("expected reasoning text, got %q", reasoning.Text)
	}

	text := result.Content[1].(lm.Text)
	if text.Text != "Here is the response" {
		t.Errorf("expected text, got %q", text.Text)
	}
}

func TestExtractReasoning_WrapGenerate_NoThinkTag(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapGenerate(mw.WrapGenerateOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "analyzing the request</think>Here is the response"},
				},
			}, nil
		},
		Params: lm.CallOptions{},
		Model:  &mockLanguageModel{},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Without startWithReasoning, no extraction should happen
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(result.Content))
	}

	text := result.Content[0].(lm.Text)
	if text.Text != "analyzing the request</think>Here is the response" {
		t.Errorf("expected unchanged text, got %q", text.Text)
	}
}

// --- wrapStream tests ---

func TestExtractReasoning_WrapStream_ExtractFromSplitTags(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "<think>"},
					lm.StreamPartTextDelta{ID: "1", Delta: "ana"},
					lm.StreamPartTextDelta{ID: "1", Delta: "lyzing the request"},
					lm.StreamPartTextDelta{ID: "1", Delta: "</think>"},
					lm.StreamPartTextDelta{ID: "1", Delta: "Here"},
					lm.StreamPartTextDelta{ID: "1", Delta: " is the response"},
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

	var reasoningTexts []string
	var textDeltas []string
	var hasReasoningStart, hasReasoningEnd bool
	for _, p := range parts {
		switch v := p.(type) {
		case lm.StreamPartReasoningStart:
			hasReasoningStart = true
		case lm.StreamPartReasoningDelta:
			reasoningTexts = append(reasoningTexts, v.Delta)
		case lm.StreamPartReasoningEnd:
			hasReasoningEnd = true
		case lm.StreamPartTextDelta:
			textDeltas = append(textDeltas, v.Delta)
		}
	}

	if !hasReasoningStart {
		t.Error("expected ReasoningStart")
	}
	if !hasReasoningEnd {
		t.Error("expected ReasoningEnd")
	}

	fullReasoning := strings.Join(reasoningTexts, "")
	if fullReasoning != "analyzing the request" {
		t.Errorf("expected reasoning, got %q", fullReasoning)
	}

	fullText := strings.Join(textDeltas, "")
	if fullText != "Here is the response" {
		t.Errorf("expected text, got %q", fullText)
	}
}

func TestExtractReasoning_WrapStream_NoThinkTag(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "this is the response"},
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

	var textDeltas []string
	for _, p := range parts {
		if td, ok := p.(lm.StreamPartTextDelta); ok {
			textDeltas = append(textDeltas, td.Delta)
		}
	}

	fullText := strings.Join(textDeltas, "")
	if fullText != "this is the response" {
		t.Errorf("expected text, got %q", fullText)
	}

	// Should not have any reasoning parts
	for _, p := range parts {
		if _, ok := p.(lm.StreamPartReasoningStart); ok {
			t.Error("did not expect ReasoningStart")
		}
	}
}

func TestExtractReasoning_WrapStream_StartWithReasoning(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{
		TagName:            "think",
		StartWithReasoning: true,
	})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "ana"},
					lm.StreamPartTextDelta{ID: "1", Delta: "lyzing the request\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: "</think>"},
					lm.StreamPartTextDelta{ID: "1", Delta: "this is the response"},
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

	var reasoningTexts []string
	var textDeltas []string
	for _, p := range parts {
		switch v := p.(type) {
		case lm.StreamPartReasoningDelta:
			reasoningTexts = append(reasoningTexts, v.Delta)
		case lm.StreamPartTextDelta:
			textDeltas = append(textDeltas, v.Delta)
		}
	}

	fullReasoning := strings.Join(reasoningTexts, "")
	if fullReasoning != "analyzing the request\n" {
		t.Errorf("expected reasoning, got %q", fullReasoning)
	}

	fullText := strings.Join(textDeltas, "")
	if fullText != "this is the response" {
		t.Errorf("expected text, got %q", fullText)
	}
}

func TestExtractReasoning_WrapStream_EmptyThinkTags(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "<think></think>"},
					lm.StreamPartTextDelta{ID: "1", Delta: " This is the answer."},
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

	// Find reasoning events
	var hasReasoningStart, hasReasoningEnd bool
	for _, p := range parts {
		switch p.(type) {
		case lm.StreamPartReasoningStart:
			hasReasoningStart = true
		case lm.StreamPartReasoningEnd:
			hasReasoningEnd = true
		}
	}

	if !hasReasoningStart {
		t.Error("expected ReasoningStart for empty think tags")
	}
	if !hasReasoningEnd {
		t.Error("expected ReasoningEnd for empty think tags")
	}
}

func TestExtractReasoning_WrapStream_NoTextAfterThink(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "<think>"},
					lm.StreamPartTextDelta{ID: "1", Delta: "ana"},
					lm.StreamPartTextDelta{ID: "1", Delta: "lyzing the request\n"},
					lm.StreamPartTextDelta{ID: "1", Delta: "</think>"},
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

	var reasoningTexts []string
	for _, p := range parts {
		if rd, ok := p.(lm.StreamPartReasoningDelta); ok {
			reasoningTexts = append(reasoningTexts, rd.Delta)
		}
	}

	fullReasoning := strings.Join(reasoningTexts, "")
	if fullReasoning != "analyzing the request\n" {
		t.Errorf("expected reasoning, got %q", fullReasoning)
	}
}

func TestExtractReasoning_WrapStream_MultipleThinkTags(t *testing.T) {
	m := ExtractReasoningMiddleware(ExtractReasoningMiddlewareOptions{TagName: "think"})

	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoStream: func() (lm.StreamResult, error) {
			return lm.StreamResult{
				Stream: streamFromParts([]lm.StreamPart{
					lm.StreamPartTextStart{ID: "1"},
					lm.StreamPartTextDelta{ID: "1", Delta: "<think>analyzing the request</think>Here is the response<think>thinking about the response</think>more"},
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

	var reasoningTexts []string
	var textDeltas []string
	for _, p := range parts {
		switch v := p.(type) {
		case lm.StreamPartReasoningDelta:
			reasoningTexts = append(reasoningTexts, v.Delta)
		case lm.StreamPartTextDelta:
			textDeltas = append(textDeltas, v.Delta)
		}
	}

	fullReasoning := strings.Join(reasoningTexts, "")
	if !strings.Contains(fullReasoning, "analyzing the request") {
		t.Error("expected first reasoning")
	}
	if !strings.Contains(fullReasoning, "thinking about the response") {
		t.Error("expected second reasoning")
	}

	fullText := strings.Join(textDeltas, "")
	if !strings.Contains(fullText, "Here is the response") {
		t.Error("expected first text part")
	}
	if !strings.Contains(fullText, "more") {
		t.Error("expected second text part")
	}
}
