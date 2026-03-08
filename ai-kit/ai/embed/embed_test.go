// Ported from: packages/ai/src/embed/embed.test.ts
package embed

import (
	"context"
	"reflect"
	"testing"
)

// mockEmbeddingModel is a mock implementation of EmbeddingModel for testing.
type mockEmbeddingModel struct {
	provider string
	modelID  string
	doEmbed  func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error)
}

func (m *mockEmbeddingModel) Provider() string { return m.provider }
func (m *mockEmbeddingModel) ModelID() string  { return m.modelID }
func (m *mockEmbeddingModel) DoEmbed(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
	return m.doEmbed(ctx, opts)
}

func newMockEmbeddingModel(doEmbed func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error)) *mockEmbeddingModel {
	return &mockEmbeddingModel{
		provider: "mock-provider",
		modelID:  "mock-model-id",
		doEmbed:  doEmbed,
	}
}

var dummyEmbedding = Embedding{0.1, 0.2, 0.3}
var testValue = "sunny day at the beach"

func TestEmbed_ResultEmbedding(t *testing.T) {
	t.Run("should generate embedding", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if !reflect.DeepEqual(opts.Values, []string{testValue}) {
				t.Errorf("expected values %v, got %v", []string{testValue}, opts.Values)
			}
			return &DoEmbedResult{
				Embeddings: []Embedding{dummyEmbedding},
				Warnings:   []Warning{},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Embedding, dummyEmbedding) {
			t.Errorf("expected embedding %v, got %v", dummyEmbedding, result.Embedding)
		}
	})
}

func TestEmbed_ResultValue(t *testing.T) {
	t.Run("should include value in the result", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings: []Embedding{dummyEmbedding},
				Warnings:   []Warning{},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Value != testValue {
			t.Errorf("expected value %q, got %q", testValue, result.Value)
		}
	})
}

func TestEmbed_ResultUsage(t *testing.T) {
	t.Run("should include usage in the result", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings: []Embedding{dummyEmbedding},
				Usage:      &EmbeddingModelUsage{Tokens: 10},
				Warnings:   []Warning{},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := EmbeddingModelUsage{Tokens: 10}
		if !reflect.DeepEqual(result.Usage, expected) {
			t.Errorf("expected usage %v, got %v", expected, result.Usage)
		}
	})
}

func TestEmbed_ResultResponse(t *testing.T) {
	t.Run("should include response in the result", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings: []Embedding{dummyEmbedding},
				Warnings:   []Warning{},
				Response: &EmbedResponseData{
					Headers: map[string]string{"foo": "bar"},
					Body:    map[string]any{"foo": "bar"},
				},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if result.Response.Headers["foo"] != "bar" {
			t.Errorf("expected response header foo=bar, got %q", result.Response.Headers["foo"])
		}
	})
}

func TestEmbed_ResultProviderMetadata(t *testing.T) {
	t.Run("should include provider metadata when returned by the model", func(t *testing.T) {
		providerMetadata := ProviderMetadata{
			"gateway": {
				"routing": map[string]any{
					"resolvedProvider": "test-provider",
				},
			},
		}

		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings:       []Embedding{dummyEmbedding},
				Warnings:         []Warning{},
				ProviderMetadata: providerMetadata,
				Response: &EmbedResponseData{
					Headers: map[string]string{},
					Body:    map[string]any{},
				},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.ProviderMetadata, providerMetadata) {
			t.Errorf("expected provider metadata %v, got %v", providerMetadata, result.ProviderMetadata)
		}
	})
}

func TestEmbed_OptionsHeaders(t *testing.T) {
	t.Run("should set headers", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if opts.Headers["custom-request-header"] != "request-header-value" {
				t.Errorf("expected custom header, got %v", opts.Headers)
			}
			return &DoEmbedResult{
				Embeddings: []Embedding{dummyEmbedding},
				Warnings:   []Warning{},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
			Headers: map[string]string{
				"custom-request-header": "request-header-value",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Embedding, dummyEmbedding) {
			t.Errorf("expected embedding %v, got %v", dummyEmbedding, result.Embedding)
		}
	})
}

func TestEmbed_OptionsProviderOptions(t *testing.T) {
	t.Run("should pass provider options to model", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			expected := map[string]map[string]any{
				"aProvider": {"someKey": "someValue"},
			}
			if !reflect.DeepEqual(opts.ProviderOptions, expected) {
				t.Errorf("expected provider options %v, got %v", expected, opts.ProviderOptions)
			}
			return &DoEmbedResult{
				Embeddings: []Embedding{{1, 2, 3}},
				Warnings:   []Warning{},
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: "test-input",
			ProviderOptions: map[string]map[string]any{
				"aProvider": {"someKey": "someValue"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := Embedding{1, 2, 3}
		if !reflect.DeepEqual(result.Embedding, expected) {
			t.Errorf("expected embedding %v, got %v", expected, result.Embedding)
		}
	})
}

func TestEmbed_ResultWarnings(t *testing.T) {
	t.Run("should include warnings in the result", func(t *testing.T) {
		expectedWarnings := []Warning{
			{Type: "other", Message: "Setting is not supported"},
			{Type: "unsupported", Feature: "dimensions", Details: "Dimensions parameter not supported"},
		}

		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings: []Embedding{dummyEmbedding},
				Warnings:   expectedWarnings,
			}, nil
		})

		result, err := Embed(context.Background(), EmbedOptions{
			Model: model,
			Value: testValue,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
			t.Errorf("expected warnings %v, got %v", expectedWarnings, result.Warnings)
		}
	})
}
