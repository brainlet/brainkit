// Ported from: packages/ai/src/middleware/wrap-image-model.test.ts
package middleware

import (
	"testing"

	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

func TestWrapImageModel_ModelIDPassThrough(t *testing.T) {
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model:      &mockImageModel{modelIDVal: "test-model"},
		Middleware: []mw.ImageModelMiddleware{{}},
	})
	if wrapped.ModelID() != "test-model" {
		t.Errorf("expected test-model, got %s", wrapped.ModelID())
	}
}

func TestWrapImageModel_OverrideModelID(t *testing.T) {
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model: &mockImageModel{modelIDVal: "test-model"},
		Middleware: []mw.ImageModelMiddleware{{
			OverrideModelID: func(model im.ImageModel) string {
				return "override-model"
			},
		}},
	})
	if wrapped.ModelID() != "override-model" {
		t.Errorf("expected override-model, got %s", wrapped.ModelID())
	}
}

func TestWrapImageModel_ModelIDParameter(t *testing.T) {
	mid := "override-model"
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model:      &mockImageModel{modelIDVal: "test-model"},
		Middleware: []mw.ImageModelMiddleware{{}},
		ModelID:    &mid,
	})
	if wrapped.ModelID() != "override-model" {
		t.Errorf("expected override-model, got %s", wrapped.ModelID())
	}
}

func TestWrapImageModel_ProviderPassThrough(t *testing.T) {
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model:      &mockImageModel{providerVal: "test-provider"},
		Middleware: []mw.ImageModelMiddleware{{}},
	})
	if wrapped.Provider() != "test-provider" {
		t.Errorf("expected test-provider, got %s", wrapped.Provider())
	}
}

func TestWrapImageModel_OverrideProvider(t *testing.T) {
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model: &mockImageModel{providerVal: "test-provider"},
		Middleware: []mw.ImageModelMiddleware{{
			OverrideProvider: func(model im.ImageModel) string {
				return "override-provider"
			},
		}},
	})
	if wrapped.Provider() != "override-provider" {
		t.Errorf("expected override-provider, got %s", wrapped.Provider())
	}
}

func TestWrapImageModel_ProviderIDParameter(t *testing.T) {
	pid := "override-provider"
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model:      &mockImageModel{providerVal: "test-provider"},
		Middleware: []mw.ImageModelMiddleware{{}},
		ProviderID: &pid,
	})
	if wrapped.Provider() != "override-provider" {
		t.Errorf("expected override-provider, got %s", wrapped.Provider())
	}
}

func TestWrapImageModel_MaxImagesPerCallPassThrough(t *testing.T) {
	val := 2
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model:      &mockImageModel{maxImagesPerCallVal: &val},
		Middleware: []mw.ImageModelMiddleware{{}},
	})
	result, err := wrapped.MaxImagesPerCall()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || *result != 2 {
		t.Errorf("expected 2, got %v", result)
	}
}

func TestWrapImageModel_OverrideMaxImagesPerCall(t *testing.T) {
	val := 2
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model: &mockImageModel{maxImagesPerCallVal: &val},
		Middleware: []mw.ImageModelMiddleware{{
			OverrideMaxImagesPerCall: func(model im.ImageModel) (*int, error) {
				v := 3
				return &v, nil
			},
		}},
	})
	result, err := wrapped.MaxImagesPerCall()
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || *result != 3 {
		t.Errorf("expected 3, got %v", result)
	}
}

func TestWrapImageModel_TransformParams(t *testing.T) {
	mock := &mockImageModel{
		doGenerateFn: func(opts im.CallOptions) (im.GenerateResult, error) {
			return im.GenerateResult{}, nil
		},
	}
	var transformCalled bool
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model: mock,
		Middleware: []mw.ImageModelMiddleware{{
			TransformParams: func(opts mw.ImageTransformParamsOptions) (im.CallOptions, error) {
				transformCalled = true
				opts.Params.Prompt = ptrStr("transformed")
				return opts.Params, nil
			},
		}},
	})

	params := im.CallOptions{Prompt: ptrStr("original"), N: 1}
	_, err := wrapped.DoGenerate(params)
	if err != nil {
		t.Fatal(err)
	}
	if !transformCalled {
		t.Error("expected transformParams to be called")
	}
	if len(mock.DoGenerateCalls) != 1 {
		t.Fatalf("expected 1 doGenerate call, got %d", len(mock.DoGenerateCalls))
	}
	if mock.DoGenerateCalls[0].Prompt == nil || *mock.DoGenerateCalls[0].Prompt != "transformed" {
		t.Errorf("expected prompt=transformed, got %v", mock.DoGenerateCalls[0].Prompt)
	}
}

func TestWrapImageModel_WrapGenerate(t *testing.T) {
	mock := &mockImageModel{
		doGenerateFn: func(opts im.CallOptions) (im.GenerateResult, error) {
			return im.GenerateResult{}, nil
		},
	}
	var wrapCalled bool
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model: mock,
		Middleware: []mw.ImageModelMiddleware{{
			WrapGenerate: func(opts mw.WrapImageGenerateOptions) (im.GenerateResult, error) {
				wrapCalled = true
				return opts.DoGenerate()
			},
		}},
	})

	params := im.CallOptions{Prompt: ptrStr("original"), N: 1}
	_, err := wrapped.DoGenerate(params)
	if err != nil {
		t.Fatal(err)
	}
	if !wrapCalled {
		t.Error("expected wrapGenerate to be called")
	}
}

func TestWrapImageModel_MultipleTransformParams(t *testing.T) {
	mock := &mockImageModel{
		doGenerateFn: func(opts im.CallOptions) (im.GenerateResult, error) {
			return im.GenerateResult{}, nil
		},
	}
	var calls []string
	wrapped := WrapImageModel(WrapImageModelOptions{
		Model: mock,
		Middleware: []mw.ImageModelMiddleware{
			{
				TransformParams: func(opts mw.ImageTransformParamsOptions) (im.CallOptions, error) {
					calls = append(calls, "first")
					return opts.Params, nil
				},
			},
			{
				TransformParams: func(opts mw.ImageTransformParamsOptions) (im.CallOptions, error) {
					calls = append(calls, "second")
					return opts.Params, nil
				},
			},
		},
	})

	params := im.CallOptions{Prompt: ptrStr("original"), N: 1}
	_, err := wrapped.DoGenerate(params)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Errorf("expected [first, second], got %v", calls)
	}
}

func TestWrapImageModel_DoesNotMutateMiddlewareArray(t *testing.T) {
	mws := []mw.ImageModelMiddleware{{}, {}}
	WrapImageModel(WrapImageModelOptions{
		Model:      &mockImageModel{},
		Middleware: mws,
	})
	if len(mws) != 2 {
		t.Errorf("expected middleware array length 2, got %d", len(mws))
	}
}
