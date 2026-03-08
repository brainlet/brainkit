// Ported from: packages/ai/src/registry/provider-registry.test.ts
package registry

import (
	"fmt"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	aierrors "github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// mockProviderV4 implements registry.Provider for testing.
// It uses function maps to look up models, similar to MockProviderV4 in TS.
type mockProviderV4 struct {
	languageModels      map[string]languagemodel.LanguageModel
	embeddingModels     map[string]embeddingmodel.EmbeddingModel
	imageModels         map[string]imagemodel.ImageModel
	transcriptionModels map[string]transcriptionmodel.TranscriptionModel
	speechModels        map[string]speechmodel.SpeechModel
	rerankingModels     map[string]rerankingmodel.RerankingModel
	videoModels         map[string]videomodel.VideoModel
}

type mockProviderV4Options struct {
	languageModels      map[string]languagemodel.LanguageModel
	embeddingModels     map[string]embeddingmodel.EmbeddingModel
	imageModels         map[string]imagemodel.ImageModel
	transcriptionModels map[string]transcriptionmodel.TranscriptionModel
	speechModels        map[string]speechmodel.SpeechModel
	rerankingModels     map[string]rerankingmodel.RerankingModel
	videoModels         map[string]videomodel.VideoModel
}

func newMockProviderV4(opts mockProviderV4Options) *mockProviderV4 {
	return &mockProviderV4{
		languageModels:      opts.languageModels,
		embeddingModels:     opts.embeddingModels,
		imageModels:         opts.imageModels,
		transcriptionModels: opts.transcriptionModels,
		speechModels:        opts.speechModels,
		rerankingModels:     opts.rerankingModels,
		videoModels:         opts.videoModels,
	}
}

func (p *mockProviderV4) SpecificationVersion() string { return "v4" }

func (p *mockProviderV4) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	if p.languageModels != nil {
		if m, ok := p.languageModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

func (p *mockProviderV4) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	if p.embeddingModels != nil {
		if m, ok := p.embeddingModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

func (p *mockProviderV4) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	if p.imageModels != nil {
		if m, ok := p.imageModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

func (p *mockProviderV4) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	if p.transcriptionModels != nil {
		if m, ok := p.transcriptionModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

func (p *mockProviderV4) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	if p.speechModels != nil {
		if m, ok := p.speechModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

func (p *mockProviderV4) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	if p.rerankingModels != nil {
		if m, ok := p.rerankingModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

func (p *mockProviderV4) VideoModel(modelID string) (videomodel.VideoModel, error) {
	if p.videoModels != nil {
		if m, ok := p.videoModels[modelID]; ok {
			return m, nil
		}
	}
	return nil, nil
}

// --- inlineProvider allows creating providers with custom function callbacks,
// porting the inline TS provider objects used in the tests. ---

type inlineProvider struct {
	languageModelFn      func(id string) (languagemodel.LanguageModel, error)
	embeddingModelFn     func(id string) (embeddingmodel.EmbeddingModel, error)
	imageModelFn         func(id string) (imagemodel.ImageModel, error)
	transcriptionModelFn func(id string) (transcriptionmodel.TranscriptionModel, error)
	speechModelFn        func(id string) (speechmodel.SpeechModel, error)
	rerankingModelFn     func(id string) (rerankingmodel.RerankingModel, error)
	videoModelFn         func(id string) (videomodel.VideoModel, error)
}

func (p *inlineProvider) SpecificationVersion() string { return "v4" }

func (p *inlineProvider) LanguageModel(id string) (languagemodel.LanguageModel, error) {
	if p.languageModelFn != nil {
		return p.languageModelFn(id)
	}
	return nil, nil
}

func (p *inlineProvider) EmbeddingModel(id string) (embeddingmodel.EmbeddingModel, error) {
	if p.embeddingModelFn != nil {
		return p.embeddingModelFn(id)
	}
	return nil, nil
}

func (p *inlineProvider) ImageModel(id string) (imagemodel.ImageModel, error) {
	if p.imageModelFn != nil {
		return p.imageModelFn(id)
	}
	return nil, nil
}

func (p *inlineProvider) TranscriptionModel(id string) (transcriptionmodel.TranscriptionModel, error) {
	if p.transcriptionModelFn != nil {
		return p.transcriptionModelFn(id)
	}
	return nil, nil
}

func (p *inlineProvider) SpeechModel(id string) (speechmodel.SpeechModel, error) {
	if p.speechModelFn != nil {
		return p.speechModelFn(id)
	}
	return nil, nil
}

func (p *inlineProvider) RerankingModel(id string) (rerankingmodel.RerankingModel, error) {
	if p.rerankingModelFn != nil {
		return p.rerankingModelFn(id)
	}
	return nil, nil
}

func (p *inlineProvider) VideoModel(id string) (videomodel.VideoModel, error) {
	if p.videoModelFn != nil {
		return p.videoModelFn(id)
	}
	return nil, nil
}

// --- languageModel tests ---

func TestProviderRegistry_LanguageModel_ReturnsModelFromProvider(t *testing.T) {
	model := newMockLanguageModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			languageModelFn: func(id string) (languagemodel.LanguageModel, error) {
				if id != "model" {
					t.Errorf("expected id 'model', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.LanguageModel("provider:model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Since no middleware is applied, the returned model should be the same instance.
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_LanguageModel_ReturnsModelWithAdditionalColon(t *testing.T) {
	model := newMockLanguageModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			languageModelFn: func(id string) (languagemodel.LanguageModel, error) {
				if id != "model:part2" {
					t.Errorf("expected id 'model:part2', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.LanguageModel("provider:model:part2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_LanguageModel_ThrowsNoSuchProviderError(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.LanguageModel("provider:model:part2")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsNoSuchProviderError(err) {
		t.Fatalf("expected NoSuchProviderError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_LanguageModel_ThrowsNoSuchModelErrorIfProviderReturnsNil(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			// All methods return nil, nil by default
		},
	})

	_, err := registry.LanguageModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_LanguageModel_ThrowsNoSuchModelErrorIfNoColon(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.LanguageModel("model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_LanguageModel_CustomSeparator(t *testing.T) {
	model := newMockLanguageModelV4()

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider": &inlineProvider{
				languageModelFn: func(id string) (languagemodel.LanguageModel, error) {
					if id != "model" {
						t.Errorf("expected id 'model', got %q", id)
					}
					return model, nil
				},
			},
		},
		ProviderRegistryOptions{Separator: "|"},
	)

	result, err := registry.LanguageModel("provider|model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_LanguageModel_CustomSeparatorMultipleCharacters(t *testing.T) {
	model := newMockLanguageModelV4()

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider": &inlineProvider{
				languageModelFn: func(id string) (languagemodel.LanguageModel, error) {
					if id != "model" {
						t.Errorf("expected id 'model', got %q", id)
					}
					return model, nil
				},
			},
		},
		ProviderRegistryOptions{Separator: " > "},
	)

	result, err := registry.LanguageModel("provider > model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

// --- embeddingModel tests ---

func TestProviderRegistry_EmbeddingModel_ReturnsModelFromProvider(t *testing.T) {
	model := newMockEmbeddingModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			embeddingModelFn: func(id string) (embeddingmodel.EmbeddingModel, error) {
				if id != "model" {
					t.Errorf("expected id 'model', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.EmbeddingModel("provider:model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_EmbeddingModel_ThrowsNoSuchProviderError(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.EmbeddingModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsNoSuchProviderError(err) {
		t.Fatalf("expected NoSuchProviderError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_EmbeddingModel_ThrowsNoSuchModelErrorIfProviderReturnsNil(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{},
	})

	// Note: The TS test checks languageModel here (it's testing that null model from
	// the provider triggers NoSuchModelError regardless of which model method).
	_, err := registry.LanguageModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_EmbeddingModel_ThrowsNoSuchModelErrorIfNoColon(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.EmbeddingModel("model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_EmbeddingModel_CustomSeparator(t *testing.T) {
	model := newMockEmbeddingModelV4()

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider": &inlineProvider{
				embeddingModelFn: func(id string) (embeddingmodel.EmbeddingModel, error) {
					if id != "model" {
						t.Errorf("expected id 'model', got %q", id)
					}
					return model, nil
				},
			},
		},
		ProviderRegistryOptions{Separator: "|"},
	)

	result, err := registry.EmbeddingModel("provider|model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

// --- imageModel tests ---

func TestProviderRegistry_ImageModel_ReturnsModelFromProvider(t *testing.T) {
	model := newMockImageModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			imageModelFn: func(id string) (imagemodel.ImageModel, error) {
				if id != "model" {
					t.Errorf("expected id 'model', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.ImageModel("provider:model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_ImageModel_ThrowsNoSuchProviderError(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.ImageModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsNoSuchProviderError(err) {
		t.Fatalf("expected NoSuchProviderError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_ImageModel_ThrowsNoSuchModelErrorIfProviderReturnsNil(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{},
	})

	_, err := registry.ImageModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_ImageModel_ThrowsNoSuchModelErrorIfNoColon(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.ImageModel("model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_ImageModel_CustomSeparator(t *testing.T) {
	model := newMockImageModelV4()

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider": &inlineProvider{
				imageModelFn: func(id string) (imagemodel.ImageModel, error) {
					if id != "model" {
						t.Errorf("expected id 'model', got %q", id)
					}
					return model, nil
				},
			},
		},
		ProviderRegistryOptions{Separator: "|"},
	)

	result, err := registry.ImageModel("provider|model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

// --- transcriptionModel tests ---

func TestProviderRegistry_TranscriptionModel_ReturnsModelFromProvider(t *testing.T) {
	model := newMockTranscriptionModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			transcriptionModelFn: func(id string) (transcriptionmodel.TranscriptionModel, error) {
				if id != "model" {
					t.Errorf("expected id 'model', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.TranscriptionModel("provider:model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_TranscriptionModel_ThrowsNoSuchProviderError(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.TranscriptionModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsNoSuchProviderError(err) {
		t.Fatalf("expected NoSuchProviderError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_TranscriptionModel_ThrowsNoSuchModelErrorIfProviderReturnsNil(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{},
	})

	_, err := registry.TranscriptionModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_TranscriptionModel_ThrowsNoSuchModelErrorIfNoColon(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.TranscriptionModel("model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

// --- speechModel tests ---

func TestProviderRegistry_SpeechModel_ReturnsModelFromProvider(t *testing.T) {
	model := newMockSpeechModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			speechModelFn: func(id string) (speechmodel.SpeechModel, error) {
				if id != "model" {
					t.Errorf("expected id 'model', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.SpeechModel("provider:model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_SpeechModel_ThrowsNoSuchProviderError(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.SpeechModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsNoSuchProviderError(err) {
		t.Fatalf("expected NoSuchProviderError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_SpeechModel_ThrowsNoSuchModelErrorIfProviderReturnsNil(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{},
	})

	_, err := registry.SpeechModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_SpeechModel_ThrowsNoSuchModelErrorIfNoColon(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.SpeechModel("model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

// --- rerankingModel tests ---

func TestProviderRegistry_RerankingModel_ReturnsModelFromProvider(t *testing.T) {
	model := newMockRerankingModelV4()

	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{
			rerankingModelFn: func(id string) (rerankingmodel.RerankingModel, error) {
				if id != "model" {
					t.Errorf("expected id 'model', got %q", id)
				}
				return model, nil
			},
		},
	})

	result, err := registry.RerankingModel("provider:model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

func TestProviderRegistry_RerankingModel_ThrowsNoSuchProviderError(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.RerankingModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !IsNoSuchProviderError(err) {
		t.Fatalf("expected NoSuchProviderError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_RerankingModel_ThrowsNoSuchModelErrorIfProviderReturnsNil(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{
		"provider": &inlineProvider{},
	})

	_, err := registry.RerankingModel("provider:model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_RerankingModel_ThrowsNoSuchModelErrorIfNoColon(t *testing.T) {
	registry := CreateProviderRegistry(map[string]Provider{})

	_, err := registry.RerankingModel("model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestProviderRegistry_RerankingModel_CustomSeparator(t *testing.T) {
	model := newMockRerankingModelV4()

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider": &inlineProvider{
				rerankingModelFn: func(id string) (rerankingmodel.RerankingModel, error) {
					if id != "model" {
						t.Errorf("expected id 'model', got %q", id)
					}
					return model, nil
				},
			},
		},
		ProviderRegistryOptions{Separator: "|"},
	)

	result, err := registry.RerankingModel("provider|model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ModelID() != model.ModelID() {
		t.Fatalf("expected model ID %q, got %q", model.ModelID(), result.ModelID())
	}
}

// --- middleware tests ---

func TestProviderRegistry_Middleware_WrapsLanguageModels(t *testing.T) {
	model1 := newMockLanguageModelV4WithID("model-1")
	model2 := newMockLanguageModelV4WithID("model-2")
	model3 := newMockLanguageModelV4WithID("model-3")

	provider1 := newMockProviderV4(mockProviderV4Options{
		languageModels: map[string]languagemodel.LanguageModel{
			"model-1": model1,
			"model-2": model2,
		},
	})

	provider2 := newMockProviderV4(mockProviderV4Options{
		languageModels: map[string]languagemodel.LanguageModel{
			"model-3": model3,
		},
	})

	var overrideCalls []languagemodel.LanguageModel

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider1": provider1,
			"provider2": provider2,
		},
		ProviderRegistryOptions{
			LanguageModelMiddleware: []LanguageModelMiddleware{
				{
					OverrideModelID: func(opts OverrideModelIDOptions) string {
						overrideCalls = append(overrideCalls, opts.Model)
						return fmt.Sprintf("override-%s", opts.Model.ModelID())
					},
				},
			},
		},
	)

	result1, err := registry.LanguageModel("provider1:model-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1.ModelID() != "override-model-1" {
		t.Fatalf("expected model ID 'override-model-1', got %q", result1.ModelID())
	}

	result2, err := registry.LanguageModel("provider1:model-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ModelID() != "override-model-2" {
		t.Fatalf("expected model ID 'override-model-2', got %q", result2.ModelID())
	}

	result3, err := registry.LanguageModel("provider2:model-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result3.ModelID() != "override-model-3" {
		t.Fatalf("expected model ID 'override-model-3', got %q", result3.ModelID())
	}

	if len(overrideCalls) != 3 {
		t.Fatalf("expected overrideModelId to be called 3 times, got %d", len(overrideCalls))
	}

	// Verify the models passed to overrideModelId were the originals
	if overrideCalls[0] != model1 {
		t.Errorf("expected first call with model1")
	}
	if overrideCalls[1] != model2 {
		t.Errorf("expected second call with model2")
	}
	if overrideCalls[2] != model3 {
		t.Errorf("expected third call with model3")
	}
}

func TestProviderRegistry_Middleware_WrapsImageModels(t *testing.T) {
	model1 := newMockImageModelV4WithID("model-1")
	model2 := newMockImageModelV4WithID("model-2")
	model3 := newMockImageModelV4WithID("model-3")

	provider1 := newMockProviderV4(mockProviderV4Options{
		imageModels: map[string]imagemodel.ImageModel{
			"model-1": model1,
			"model-2": model2,
		},
	})

	provider2 := newMockProviderV4(mockProviderV4Options{
		imageModels: map[string]imagemodel.ImageModel{
			"model-3": model3,
		},
	})

	var overrideCalls []imagemodel.ImageModel

	registry := CreateProviderRegistry(
		map[string]Provider{
			"provider1": provider1,
			"provider2": provider2,
		},
		ProviderRegistryOptions{
			ImageModelMiddleware: []ImageModelMiddleware{
				{
					OverrideModelID: func(opts ImageOverrideModelIDOptions) string {
						overrideCalls = append(overrideCalls, opts.Model)
						return fmt.Sprintf("override-%s", opts.Model.ModelID())
					},
				},
			},
		},
	)

	result1, err := registry.ImageModel("provider1:model-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result1.ModelID() != "override-model-1" {
		t.Fatalf("expected model ID 'override-model-1', got %q", result1.ModelID())
	}

	result2, err := registry.ImageModel("provider1:model-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result2.ModelID() != "override-model-2" {
		t.Fatalf("expected model ID 'override-model-2', got %q", result2.ModelID())
	}

	result3, err := registry.ImageModel("provider2:model-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result3.ModelID() != "override-model-3" {
		t.Fatalf("expected model ID 'override-model-3', got %q", result3.ModelID())
	}

	if len(overrideCalls) != 3 {
		t.Fatalf("expected overrideModelId to be called 3 times, got %d", len(overrideCalls))
	}

	if overrideCalls[0] != model1 {
		t.Errorf("expected first call with model1")
	}
	if overrideCalls[1] != model2 {
		t.Errorf("expected second call with model2")
	}
	if overrideCalls[2] != model3 {
		t.Errorf("expected third call with model3")
	}
}
