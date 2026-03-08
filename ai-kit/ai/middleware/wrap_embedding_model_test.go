// Ported from: packages/ai/src/middleware/wrap-embedding-model.test.ts
package middleware

import (
	"testing"

	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

func TestWrapEmbeddingModel_ModelIDPassThrough(t *testing.T) {
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{modelIDVal: "test-model"},
		Middleware: []mw.EmbeddingModelMiddleware{{}},
	})
	if wrapped.ModelID() != "test-model" {
		t.Errorf("expected test-model, got %s", wrapped.ModelID())
	}
}

func TestWrapEmbeddingModel_OverrideModelID(t *testing.T) {
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: &mockEmbeddingModel{modelIDVal: "test-model"},
		Middleware: []mw.EmbeddingModelMiddleware{{
			OverrideModelID: func(model em.EmbeddingModel) string {
				return "override-model"
			},
		}},
	})
	if wrapped.ModelID() != "override-model" {
		t.Errorf("expected override-model, got %s", wrapped.ModelID())
	}
}

func TestWrapEmbeddingModel_ModelIDParameter(t *testing.T) {
	mid := "override-model"
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{modelIDVal: "test-model"},
		Middleware: []mw.EmbeddingModelMiddleware{{}},
		ModelID:    &mid,
	})
	if wrapped.ModelID() != "override-model" {
		t.Errorf("expected override-model, got %s", wrapped.ModelID())
	}
}

func TestWrapEmbeddingModel_ProviderPassThrough(t *testing.T) {
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{providerVal: "test-provider"},
		Middleware: []mw.EmbeddingModelMiddleware{{}},
	})
	if wrapped.Provider() != "test-provider" {
		t.Errorf("expected test-provider, got %s", wrapped.Provider())
	}
}

func TestWrapEmbeddingModel_OverrideProvider(t *testing.T) {
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: &mockEmbeddingModel{providerVal: "test-provider"},
		Middleware: []mw.EmbeddingModelMiddleware{{
			OverrideProvider: func(model em.EmbeddingModel) string {
				return "override-provider"
			},
		}},
	})
	if wrapped.Provider() != "override-provider" {
		t.Errorf("expected override-provider, got %s", wrapped.Provider())
	}
}

func TestWrapEmbeddingModel_ProviderIDParameter(t *testing.T) {
	pid := "override-provider"
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{providerVal: "test-provider"},
		Middleware: []mw.EmbeddingModelMiddleware{{}},
		ProviderID: &pid,
	})
	if wrapped.Provider() != "override-provider" {
		t.Errorf("expected override-provider, got %s", wrapped.Provider())
	}
}

func TestWrapEmbeddingModel_MaxEmbeddingsPerCallPassThrough(t *testing.T) {
	val := 2
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{maxEmbeddingsPerCallVal: &val},
		Middleware: []mw.EmbeddingModelMiddleware{{}},
	})
	result, err := wrapped.MaxEmbeddingsPerCall()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || *result != 2 {
		t.Errorf("expected 2, got %v", result)
	}
}

func TestWrapEmbeddingModel_OverrideMaxEmbeddingsPerCall(t *testing.T) {
	val := 2
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: &mockEmbeddingModel{maxEmbeddingsPerCallVal: &val},
		Middleware: []mw.EmbeddingModelMiddleware{{
			OverrideMaxEmbeddingsPerCall: func(model em.EmbeddingModel) (*int, error) {
				v := 3
				return &v, nil
			},
		}},
	})
	result, err := wrapped.MaxEmbeddingsPerCall()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || *result != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestWrapEmbeddingModel_SupportsParallelCallsPassThrough(t *testing.T) {
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{supportsParallelCallsVal: true},
		Middleware: []mw.EmbeddingModelMiddleware{{}},
	})
	result, err := wrapped.SupportsParallelCalls()
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestWrapEmbeddingModel_OverrideSupportsParallelCalls(t *testing.T) {
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: &mockEmbeddingModel{supportsParallelCallsVal: false},
		Middleware: []mw.EmbeddingModelMiddleware{{
			OverrideSupportsParallelCalls: func(model em.EmbeddingModel) (bool, error) {
				return true, nil
			},
		}},
	})
	result, err := wrapped.SupportsParallelCalls()
	if err != nil {
		t.Fatal(err)
	}
	if !result {
		t.Error("expected true")
	}
}

func TestWrapEmbeddingModel_TransformParams(t *testing.T) {
	mock := &mockEmbeddingModel{}
	var transformCalled bool
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: mock,
		Middleware: []mw.EmbeddingModelMiddleware{{
			TransformParams: func(opts mw.EmbeddingTransformParamsOptions) (em.CallOptions, error) {
				transformCalled = true
				opts.Params.Values = append(opts.Params.Values, "extra")
				return opts.Params, nil
			},
		}},
	})

	params := em.CallOptions{Values: []string{"hello"}}
	_, err := wrapped.DoEmbed(params)
	if err != nil {
		t.Fatal(err)
	}
	if !transformCalled {
		t.Error("expected transformParams to be called")
	}
	if len(mock.DoEmbedCalls) != 1 {
		t.Fatalf("expected 1 doEmbed call, got %d", len(mock.DoEmbedCalls))
	}
	if len(mock.DoEmbedCalls[0].Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(mock.DoEmbedCalls[0].Values))
	}
}

func TestWrapEmbeddingModel_WrapEmbed(t *testing.T) {
	mock := &mockEmbeddingModel{
		doEmbedFn: func(opts em.CallOptions) (em.Result, error) {
			return em.Result{Embeddings: [][]float64{{1.0, 2.0}}}, nil
		},
	}
	var wrapCalled bool
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: mock,
		Middleware: []mw.EmbeddingModelMiddleware{{
			WrapEmbed: func(opts mw.WrapEmbedOptions) (em.Result, error) {
				wrapCalled = true
				return opts.DoEmbed()
			},
		}},
	})

	params := em.CallOptions{Values: []string{"hello"}}
	result, err := wrapped.DoEmbed(params)
	if err != nil {
		t.Fatal(err)
	}
	if !wrapCalled {
		t.Error("expected wrapEmbed to be called")
	}
	if len(result.Embeddings) != 1 {
		t.Error("expected 1 embedding")
	}
}

func TestWrapEmbeddingModel_MultipleTransformParams(t *testing.T) {
	mock := &mockEmbeddingModel{}
	var calls []string
	wrapped := WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model: mock,
		Middleware: []mw.EmbeddingModelMiddleware{
			{
				TransformParams: func(opts mw.EmbeddingTransformParamsOptions) (em.CallOptions, error) {
					calls = append(calls, "first")
					opts.Params.Values = append(opts.Params.Values, "from-first")
					return opts.Params, nil
				},
			},
			{
				TransformParams: func(opts mw.EmbeddingTransformParamsOptions) (em.CallOptions, error) {
					calls = append(calls, "second")
					opts.Params.Values = append(opts.Params.Values, "from-second")
					return opts.Params, nil
				},
			},
		},
	})

	params := em.CallOptions{Values: []string{"original"}}
	_, err := wrapped.DoEmbed(params)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Errorf("expected [first, second], got %v", calls)
	}
	if len(mock.DoEmbedCalls[0].Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(mock.DoEmbedCalls[0].Values))
	}
}

func TestWrapEmbeddingModel_DoesNotMutateMiddlewareArray(t *testing.T) {
	mws := []mw.EmbeddingModelMiddleware{{}, {}}
	WrapEmbeddingModel(WrapEmbeddingModelOptions{
		Model:      &mockEmbeddingModel{},
		Middleware: mws,
	})
	if len(mws) != 2 {
		t.Errorf("expected middleware array length 2, got %d", len(mws))
	}
}
