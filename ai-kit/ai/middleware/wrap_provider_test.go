// Ported from: packages/ai/src/middleware/wrap-provider.test.ts
package middleware

import (
	"testing"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

func TestWrapProvider_WrapsAllLanguageModels(t *testing.T) {
	model1 := &mockLanguageModel{modelIDVal: "model-1"}
	model2 := &mockLanguageModel{modelIDVal: "model-2"}
	model3 := &mockLanguageModel{modelIDVal: "model-3"}

	p := &mockProvider{
		languageModels: map[string]lm.LanguageModel{
			"model-1": model1,
			"model-2": model2,
			"model-3": model3,
		},
	}

	var overrideCalls []string
	wrapped := WrapProvider(WrapProviderOptions{
		Provider: p,
		LanguageModelMiddleware: []mw.LanguageModelMiddleware{{
			OverrideModelID: func(model lm.LanguageModel) string {
				overrideCalls = append(overrideCalls, model.ModelID())
				return "override-" + model.ModelID()
			},
		}},
	})

	lm1, err := wrapped.LanguageModel("model-1")
	if err != nil {
		t.Fatal(err)
	}
	if lm1.ModelID() != "override-model-1" {
		t.Errorf("expected override-model-1, got %s", lm1.ModelID())
	}

	lm2, err := wrapped.LanguageModel("model-2")
	if err != nil {
		t.Fatal(err)
	}
	if lm2.ModelID() != "override-model-2" {
		t.Errorf("expected override-model-2, got %s", lm2.ModelID())
	}

	lm3, err := wrapped.LanguageModel("model-3")
	if err != nil {
		t.Fatal(err)
	}
	if lm3.ModelID() != "override-model-3" {
		t.Errorf("expected override-model-3, got %s", lm3.ModelID())
	}

	if len(overrideCalls) != 3 {
		t.Errorf("expected 3 override calls, got %d", len(overrideCalls))
	}
}

func TestWrapProvider_WrapsAllImageModels(t *testing.T) {
	imgModel1 := &mockImageModel{modelIDVal: "model-1"}
	imgModel2 := &mockImageModel{modelIDVal: "model-2"}
	imgModel3 := &mockImageModel{modelIDVal: "model-3"}
	langModel := &mockLanguageModel{modelIDVal: "language-model"}

	p := &mockProvider{
		languageModels: map[string]lm.LanguageModel{
			"language-model": langModel,
		},
		imageModels: map[string]im.ImageModel{
			"model-1": imgModel1,
			"model-2": imgModel2,
			"model-3": imgModel3,
		},
	}

	var overrideCalls []string
	wrapped := WrapProvider(WrapProviderOptions{
		Provider:               p,
		LanguageModelMiddleware: []mw.LanguageModelMiddleware{{}},
		ImageModelMiddleware: []mw.ImageModelMiddleware{{
			OverrideModelID: func(model im.ImageModel) string {
				overrideCalls = append(overrideCalls, model.ModelID())
				return "override-" + model.ModelID()
			},
		}},
	})

	im1, err := wrapped.ImageModel("model-1")
	if err != nil {
		t.Fatal(err)
	}
	if im1.ModelID() != "override-model-1" {
		t.Errorf("expected override-model-1, got %s", im1.ModelID())
	}

	im2, err := wrapped.ImageModel("model-2")
	if err != nil {
		t.Fatal(err)
	}
	if im2.ModelID() != "override-model-2" {
		t.Errorf("expected override-model-2, got %s", im2.ModelID())
	}

	im3, err := wrapped.ImageModel("model-3")
	if err != nil {
		t.Fatal(err)
	}
	if im3.ModelID() != "override-model-3" {
		t.Errorf("expected override-model-3, got %s", im3.ModelID())
	}

	if len(overrideCalls) != 3 {
		t.Errorf("expected 3 override calls, got %d", len(overrideCalls))
	}
}
