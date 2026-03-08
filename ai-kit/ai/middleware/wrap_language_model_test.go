// Ported from: packages/ai/src/middleware/wrap-language-model.test.ts
package middleware

import (
	"regexp"
	"testing"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

func testPrompt() lm.Prompt {
	return lm.Prompt{
		lm.UserMessage{Content: []lm.UserMessagePart{lm.TextPart{Text: "Hello"}}},
	}
}

func TestWrapLanguageModel_ModelIDPassThrough(t *testing.T) {
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: &mockLanguageModel{modelIDVal: "test-model"},
		Middleware: []mw.LanguageModelMiddleware{{}},
	})
	if wrapped.ModelID() != "test-model" {
		t.Errorf("expected test-model, got %s", wrapped.ModelID())
	}
}

func TestWrapLanguageModel_OverrideModelID(t *testing.T) {
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: &mockLanguageModel{modelIDVal: "test-model"},
		Middleware: []mw.LanguageModelMiddleware{{
			OverrideModelID: func(model lm.LanguageModel) string {
				return "override-model"
			},
		}},
	})
	if wrapped.ModelID() != "override-model" {
		t.Errorf("expected override-model, got %s", wrapped.ModelID())
	}
}

func TestWrapLanguageModel_ModelIDParameter(t *testing.T) {
	mid := "override-model"
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model:      &mockLanguageModel{modelIDVal: "test-model"},
		Middleware: []mw.LanguageModelMiddleware{{}},
		ModelID:    &mid,
	})
	if wrapped.ModelID() != "override-model" {
		t.Errorf("expected override-model, got %s", wrapped.ModelID())
	}
}

func TestWrapLanguageModel_ProviderPassThrough(t *testing.T) {
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: &mockLanguageModel{providerVal: "test-provider"},
		Middleware: []mw.LanguageModelMiddleware{{}},
	})
	if wrapped.Provider() != "test-provider" {
		t.Errorf("expected test-provider, got %s", wrapped.Provider())
	}
}

func TestWrapLanguageModel_OverrideProvider(t *testing.T) {
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: &mockLanguageModel{providerVal: "test-provider"},
		Middleware: []mw.LanguageModelMiddleware{{
			OverrideProvider: func(model lm.LanguageModel) string {
				return "override-provider"
			},
		}},
	})
	if wrapped.Provider() != "override-provider" {
		t.Errorf("expected override-provider, got %s", wrapped.Provider())
	}
}

func TestWrapLanguageModel_ProviderIDParameter(t *testing.T) {
	pid := "override-provider"
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model:      &mockLanguageModel{providerVal: "test-provider"},
		Middleware: []mw.LanguageModelMiddleware{{}},
		ProviderID: &pid,
	})
	if wrapped.Provider() != "override-provider" {
		t.Errorf("expected override-provider, got %s", wrapped.Provider())
	}
}

func TestWrapLanguageModel_SupportedUrlsPassThrough(t *testing.T) {
	expected := map[string][]*regexp.Regexp{
		"original/*": {regexp.MustCompile(`^https://.*$`)},
	}
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: &mockLanguageModel{supportedUrlsVal: expected},
		Middleware: []mw.LanguageModelMiddleware{{}},
	})
	urls, err := wrapped.SupportedUrls()
	if err != nil {
		t.Fatal(err)
	}
	if len(urls["original/*"]) != 1 {
		t.Errorf("expected 1 url pattern, got %d", len(urls["original/*"]))
	}
}

func TestWrapLanguageModel_OverrideSupportedUrls(t *testing.T) {
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: &mockLanguageModel{
			supportedUrlsVal: map[string][]*regexp.Regexp{
				"original/*": {regexp.MustCompile(`^https://.*$`)},
			},
		},
		Middleware: []mw.LanguageModelMiddleware{{
			OverrideSupportedUrls: func(model lm.LanguageModel) (map[string][]*regexp.Regexp, error) {
				return map[string][]*regexp.Regexp{
					"override/*": {regexp.MustCompile(`^https://.*$`)},
				}, nil
			},
		}},
	})
	urls, err := wrapped.SupportedUrls()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := urls["override/*"]; !ok {
		t.Error("expected override/* key")
	}
}

func TestWrapLanguageModel_TransformParamsForGenerate(t *testing.T) {
	mock := &mockLanguageModel{}
	var capturedType string
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: mock,
		Middleware: []mw.LanguageModelMiddleware{{
			TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
				capturedType = opts.Type
				opts.Params.Temperature = ptrFloat64(0.5)
				return opts.Params, nil
			},
		}},
	})

	params := lm.CallOptions{Prompt: testPrompt()}
	_, err := wrapped.DoGenerate(params)
	if err != nil {
		t.Fatal(err)
	}
	if capturedType != "generate" {
		t.Errorf("expected type=generate, got %s", capturedType)
	}
	if len(mock.DoGenerateCalls) != 1 {
		t.Fatalf("expected 1 doGenerate call, got %d", len(mock.DoGenerateCalls))
	}
	if mock.DoGenerateCalls[0].Temperature == nil || *mock.DoGenerateCalls[0].Temperature != 0.5 {
		t.Error("expected transformed temperature 0.5")
	}
}

func TestWrapLanguageModel_WrapGenerate(t *testing.T) {
	mock := &mockLanguageModel{
		doGenerateFn: func(opts lm.CallOptions) (lm.GenerateResult, error) {
			return lm.GenerateResult{
				Content: []lm.Content{lm.Text{Text: "result"}},
			}, nil
		},
	}
	var wrapCalled bool
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: mock,
		Middleware: []mw.LanguageModelMiddleware{{
			WrapGenerate: func(opts mw.WrapGenerateOptions) (lm.GenerateResult, error) {
				wrapCalled = true
				return opts.DoGenerate()
			},
		}},
	})

	params := lm.CallOptions{Prompt: testPrompt()}
	result, err := wrapped.DoGenerate(params)
	if err != nil {
		t.Fatal(err)
	}
	if !wrapCalled {
		t.Error("expected wrapGenerate to be called")
	}
	if len(result.Content) == 0 {
		t.Error("expected content in result")
	}
}

func TestWrapLanguageModel_TransformParamsForStream(t *testing.T) {
	mock := &mockLanguageModel{}
	var capturedType string
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: mock,
		Middleware: []mw.LanguageModelMiddleware{{
			TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
				capturedType = opts.Type
				opts.Params.Temperature = ptrFloat64(0.8)
				return opts.Params, nil
			},
		}},
	})

	params := lm.CallOptions{Prompt: testPrompt()}
	_, err := wrapped.DoStream(params)
	if err != nil {
		t.Fatal(err)
	}
	if capturedType != "stream" {
		t.Errorf("expected type=stream, got %s", capturedType)
	}
	if len(mock.DoStreamCalls) != 1 {
		t.Fatalf("expected 1 doStream call, got %d", len(mock.DoStreamCalls))
	}
	if mock.DoStreamCalls[0].Temperature == nil || *mock.DoStreamCalls[0].Temperature != 0.8 {
		t.Error("expected transformed temperature 0.8")
	}
}

func TestWrapLanguageModel_WrapStream(t *testing.T) {
	mock := &mockLanguageModel{
		doStreamFn: func(opts lm.CallOptions) (lm.StreamResult, error) {
			return lm.StreamResult{Stream: make(<-chan lm.StreamPart)}, nil
		},
	}
	var wrapCalled bool
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: mock,
		Middleware: []mw.LanguageModelMiddleware{{
			WrapStream: func(opts mw.WrapStreamOptions) (lm.StreamResult, error) {
				wrapCalled = true
				return opts.DoStream()
			},
		}},
	})

	params := lm.CallOptions{Prompt: testPrompt()}
	_, err := wrapped.DoStream(params)
	if err != nil {
		t.Fatal(err)
	}
	if !wrapCalled {
		t.Error("expected wrapStream to be called")
	}
}

func TestWrapLanguageModel_MultipleTransformParamsGenerate(t *testing.T) {
	mock := &mockLanguageModel{}
	var calls []string
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: mock,
		Middleware: []mw.LanguageModelMiddleware{
			{
				TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
					calls = append(calls, "first")
					opts.Params.Temperature = ptrFloat64(0.5)
					return opts.Params, nil
				},
			},
			{
				TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
					calls = append(calls, "second")
					opts.Params.TopK = ptrInt(10)
					return opts.Params, nil
				},
			},
		},
	})

	params := lm.CallOptions{Prompt: testPrompt()}
	_, err := wrapped.DoGenerate(params)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Errorf("expected [first, second], got %v", calls)
	}
	called := mock.DoGenerateCalls[0]
	if called.Temperature == nil || *called.Temperature != 0.5 {
		t.Error("expected temperature 0.5")
	}
	if called.TopK == nil || *called.TopK != 10 {
		t.Error("expected topK 10")
	}
}

func TestWrapLanguageModel_MultipleTransformParamsStream(t *testing.T) {
	mock := &mockLanguageModel{}
	var calls []string
	wrapped := WrapLanguageModel(WrapLanguageModelOptions{
		Model: mock,
		Middleware: []mw.LanguageModelMiddleware{
			{
				TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
					calls = append(calls, "first")
					opts.Params.Temperature = ptrFloat64(0.5)
					return opts.Params, nil
				},
			},
			{
				TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
					calls = append(calls, "second")
					opts.Params.TopK = ptrInt(10)
					return opts.Params, nil
				},
			},
		},
	})

	params := lm.CallOptions{Prompt: testPrompt()}
	_, err := wrapped.DoStream(params)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Errorf("expected [first, second], got %v", calls)
	}
}

func TestWrapLanguageModel_DoesNotMutateMiddlewareArray(t *testing.T) {
	mws := []mw.LanguageModelMiddleware{{}, {}}
	WrapLanguageModel(WrapLanguageModelOptions{
		Model:      &mockLanguageModel{},
		Middleware: mws,
	})
	if len(mws) != 2 {
		t.Errorf("expected middleware array length 2, got %d", len(mws))
	}
}
