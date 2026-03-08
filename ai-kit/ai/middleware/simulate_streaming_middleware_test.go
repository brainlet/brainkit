// Ported from: packages/ai/src/middleware/simulate-streaming-middleware.test.ts
package middleware

import (
	"testing"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

func TestSimulateStreaming_TextResponse(t *testing.T) {
	mock := &mockLanguageModel{
		doGenerateFn: func(opts lm.CallOptions) (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{lm.Text{Text: "This is a test response"}},
				FinishReason: lm.FinishReason{
					Unified: "stop",
					Raw:     ptrStr("stop"),
				},
			}, nil
		},
	}

	m := SimulateStreamingMiddleware()
	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return mock.DoGenerate(lm.CallOptions{})
		},
		DoStream: func() (lm.StreamResult, error) {
			return mock.DoStream(lm.CallOptions{})
		},
		Params: lm.CallOptions{},
		Model:  mock,
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)

	// Should have: StreamStart, TextStart, TextDelta, TextEnd, Finish
	var hasTextStart, hasTextDelta, hasTextEnd, hasFinish bool
	var textContent string
	for _, p := range parts {
		switch v := p.(type) {
		case lm.StreamPartTextStart:
			hasTextStart = true
		case lm.StreamPartTextDelta:
			hasTextDelta = true
			textContent += v.Delta
		case lm.StreamPartTextEnd:
			hasTextEnd = true
		case lm.StreamPartFinish:
			hasFinish = true
			if v.FinishReason.Unified != "stop" {
				t.Errorf("expected stop finish reason, got %s", v.FinishReason.Unified)
			}
		}
	}

	if !hasTextStart {
		t.Error("expected TextStart")
	}
	if !hasTextDelta {
		t.Error("expected TextDelta")
	}
	if textContent != "This is a test response" {
		t.Errorf("expected 'This is a test response', got '%s'", textContent)
	}
	if !hasTextEnd {
		t.Error("expected TextEnd")
	}
	if !hasFinish {
		t.Error("expected Finish")
	}
}

func TestSimulateStreaming_ReasoningResponse(t *testing.T) {
	mock := &mockLanguageModel{
		doGenerateFn: func(opts lm.CallOptions) (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Reasoning{Text: "This is the reasoning process"},
					lm.Text{Text: "This is a test response"},
				},
				FinishReason: lm.FinishReason{Unified: "stop", Raw: ptrStr("stop")},
			}, nil
		},
	}

	m := SimulateStreamingMiddleware()
	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return mock.DoGenerate(lm.CallOptions{})
		},
		DoStream: func() (lm.StreamResult, error) {
			return mock.DoStream(lm.CallOptions{})
		},
		Params: lm.CallOptions{},
		Model:  mock,
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)

	var reasoningText, textContent string
	var hasReasoningStart, hasReasoningEnd bool
	for _, p := range parts {
		switch v := p.(type) {
		case lm.StreamPartReasoningStart:
			hasReasoningStart = true
		case lm.StreamPartReasoningDelta:
			reasoningText += v.Delta
		case lm.StreamPartReasoningEnd:
			hasReasoningEnd = true
		case lm.StreamPartTextDelta:
			textContent += v.Delta
		}
	}

	if !hasReasoningStart {
		t.Error("expected ReasoningStart")
	}
	if reasoningText != "This is the reasoning process" {
		t.Errorf("expected reasoning text, got '%s'", reasoningText)
	}
	if !hasReasoningEnd {
		t.Error("expected ReasoningEnd")
	}
	if textContent != "This is a test response" {
		t.Errorf("expected text content, got '%s'", textContent)
	}
}

func TestSimulateStreaming_MultipleReasoningParts(t *testing.T) {
	mock := &mockLanguageModel{
		doGenerateFn: func(opts lm.CallOptions) (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{
					lm.Text{Text: "This is a test response"},
					lm.Reasoning{Text: "First reasoning step"},
					lm.Reasoning{Text: "Second reasoning step"},
				},
				FinishReason: lm.FinishReason{Unified: "stop", Raw: ptrStr("stop")},
			}, nil
		},
	}

	m := SimulateStreamingMiddleware()
	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return mock.DoGenerate(lm.CallOptions{})
		},
		DoStream: func() (lm.StreamResult, error) {
			return mock.DoStream(lm.CallOptions{})
		},
		Params: lm.CallOptions{},
		Model:  mock,
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)

	var reasoningStartCount, reasoningEndCount int
	var reasoningTexts []string
	for _, p := range parts {
		switch v := p.(type) {
		case lm.StreamPartReasoningStart:
			reasoningStartCount++
		case lm.StreamPartReasoningDelta:
			reasoningTexts = append(reasoningTexts, v.Delta)
		case lm.StreamPartReasoningEnd:
			reasoningEndCount++
		}
	}

	if reasoningStartCount != 2 {
		t.Errorf("expected 2 reasoning starts, got %d", reasoningStartCount)
	}
	if reasoningEndCount != 2 {
		t.Errorf("expected 2 reasoning ends, got %d", reasoningEndCount)
	}
	if len(reasoningTexts) != 2 {
		t.Fatalf("expected 2 reasoning texts, got %d", len(reasoningTexts))
	}
	if reasoningTexts[0] != "First reasoning step" {
		t.Errorf("expected first reasoning, got '%s'", reasoningTexts[0])
	}
	if reasoningTexts[1] != "Second reasoning step" {
		t.Errorf("expected second reasoning, got '%s'", reasoningTexts[1])
	}
}

func TestSimulateStreaming_PreservesMetadata(t *testing.T) {
	mock := &mockLanguageModel{
		doGenerateFn: func(opts lm.CallOptions) (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{lm.Text{Text: "This is a test response"}},
				FinishReason: lm.FinishReason{Unified: "stop", Raw: ptrStr("stop")},
				ProviderMetadata: map[string]map[string]any{
					"custom": {"key": "value"},
				},
			}, nil
		},
	}

	m := SimulateStreamingMiddleware()
	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return mock.DoGenerate(lm.CallOptions{})
		},
		DoStream: func() (lm.StreamResult, error) {
			return mock.DoStream(lm.CallOptions{})
		},
		Params: lm.CallOptions{},
		Model:  mock,
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)

	var finishPart *lm.StreamPartFinish
	for _, p := range parts {
		if f, ok := p.(lm.StreamPartFinish); ok {
			finishPart = &f
		}
	}

	if finishPart == nil {
		t.Fatal("expected Finish part")
	}
	if finishPart.ProviderMetadata == nil {
		t.Fatal("expected provider metadata")
	}
	custom, ok := finishPart.ProviderMetadata["custom"]
	if !ok {
		t.Fatal("expected custom key in provider metadata")
	}
	if custom["key"] != "value" {
		t.Errorf("expected key=value, got %v", custom["key"])
	}
}

func TestSimulateStreaming_EmptyTextResponse(t *testing.T) {
	mock := &mockLanguageModel{
		doGenerateFn: func(opts lm.CallOptions) (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content:      []lm.Content{lm.Text{Text: ""}},
				FinishReason: lm.FinishReason{Unified: "stop", Raw: ptrStr("stop")},
			}, nil
		},
	}

	m := SimulateStreamingMiddleware()
	result, err := m.WrapStream(mw.WrapStreamOptions{
		DoGenerate: func() (lm.GenerateResult, error) {
			return mock.DoGenerate(lm.CallOptions{})
		},
		DoStream: func() (lm.StreamResult, error) {
			return mock.DoStream(lm.CallOptions{})
		},
		Params: lm.CallOptions{},
		Model:  mock,
	})
	if err != nil {
		t.Fatal(err)
	}

	parts := collectStreamParts(result.Stream)

	// Empty text should not produce TextStart/TextDelta/TextEnd
	for _, p := range parts {
		if _, ok := p.(lm.StreamPartTextStart); ok {
			t.Error("did not expect TextStart for empty text")
		}
	}

	// But should still have Finish
	var hasFinish bool
	for _, p := range parts {
		if _, ok := p.(lm.StreamPartFinish); ok {
			hasFinish = true
		}
	}
	if !hasFinish {
		t.Error("expected Finish part")
	}
}
