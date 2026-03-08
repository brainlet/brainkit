// Ported from: packages/ai/src/embed/embed-many.test.ts
package embed

import (
	"context"
	"reflect"
	"testing"
)

var dummyEmbeddings = []Embedding{
	{0.1, 0.2, 0.3},
	{0.4, 0.5, 0.6},
	{0.7, 0.8, 0.9},
}

var testValues = []string{
	"sunny day at the beach",
	"rainy afternoon in the city",
	"snowy night in the mountains",
}

func TestEmbedMany_ResultEmbeddings(t *testing.T) {
	t.Run("should generate embeddings", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if !reflect.DeepEqual(opts.Values, testValues) {
				t.Errorf("expected values %v, got %v", testValues, opts.Values)
			}
			return &DoEmbedResult{
				Embeddings: dummyEmbeddings,
				Warnings:   []Warning{},
			}, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:  model,
			Values: testValues,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Embeddings, dummyEmbeddings) {
			t.Errorf("expected embeddings %v, got %v", dummyEmbeddings, result.Embeddings)
		}
	})

	t.Run("should generate embeddings when several calls are required", func(t *testing.T) {
		// Dispatch based on received values, not call order, since chunks run concurrently.
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if reflect.DeepEqual(opts.Values, testValues[:2]) {
				return &DoEmbedResult{
					Embeddings: dummyEmbeddings[:2],
					Warnings:   []Warning{},
				}, nil
			} else if reflect.DeepEqual(opts.Values, testValues[2:]) {
				return &DoEmbedResult{
					Embeddings: dummyEmbeddings[2:],
					Warnings:   []Warning{},
				}, nil
			}
			t.Fatalf("unexpected values: %v", opts.Values)
			return nil, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:                model,
			Values:               testValues,
			MaxEmbeddingsPerCall: 2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Embeddings, dummyEmbeddings) {
			t.Errorf("expected embeddings %v, got %v", dummyEmbeddings, result.Embeddings)
		}
	})
}

func TestEmbedMany_ResultValues(t *testing.T) {
	t.Run("should include values in the result", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings: dummyEmbeddings,
				Warnings:   []Warning{},
			}, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:  model,
			Values: testValues,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Values, testValues) {
			t.Errorf("expected values %v, got %v", testValues, result.Values)
		}
	})
}

func TestEmbedMany_ResultUsage(t *testing.T) {
	t.Run("should include usage in the result", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if reflect.DeepEqual(opts.Values, testValues[:2]) {
				return &DoEmbedResult{
					Embeddings: dummyEmbeddings[:2],
					Usage:      &EmbeddingModelUsage{Tokens: 10},
					Warnings:   []Warning{},
				}, nil
			} else if reflect.DeepEqual(opts.Values, testValues[2:]) {
				return &DoEmbedResult{
					Embeddings: dummyEmbeddings[2:],
					Usage:      &EmbeddingModelUsage{Tokens: 20},
					Warnings:   []Warning{},
				}, nil
			}
			t.Fatal("unexpected values")
			return nil, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:                model,
			Values:               testValues,
			MaxEmbeddingsPerCall: 2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := EmbeddingModelUsage{Tokens: 30}
		if !reflect.DeepEqual(result.Usage, expected) {
			t.Errorf("expected usage %v, got %v", expected, result.Usage)
		}
	})
}

func TestEmbedMany_OptionsHeaders(t *testing.T) {
	t.Run("should set headers", func(t *testing.T) {
		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if opts.Headers["custom-request-header"] != "request-header-value" {
				t.Errorf("expected custom header, got %v", opts.Headers)
			}
			return &DoEmbedResult{
				Embeddings: dummyEmbeddings,
				Warnings:   []Warning{},
			}, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:  model,
			Values: testValues,
			Headers: map[string]string{
				"custom-request-header": "request-header-value",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Embeddings, dummyEmbeddings) {
			t.Errorf("expected embeddings %v, got %v", dummyEmbeddings, result.Embeddings)
		}
	})
}

func TestEmbedMany_ResultWarnings(t *testing.T) {
	t.Run("should include warnings in the result (single call path)", func(t *testing.T) {
		expectedWarnings := []Warning{
			{Type: "other", Message: "Setting is not supported"},
		}

		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings: dummyEmbeddings,
				Warnings:   expectedWarnings,
			}, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:  model,
			Values: testValues,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
			t.Errorf("expected warnings %v, got %v", expectedWarnings, result.Warnings)
		}
	})

	t.Run("should aggregate warnings from multiple calls", func(t *testing.T) {
		warning1 := Warning{Type: "other", Message: "Warning from call 1"}
		warning2 := Warning{Type: "unsupported", Feature: "dimensions"}

		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			if reflect.DeepEqual(opts.Values, testValues[:2]) {
				return &DoEmbedResult{
					Embeddings: dummyEmbeddings[:2],
					Warnings:   []Warning{warning1},
				}, nil
			} else if reflect.DeepEqual(opts.Values, testValues[2:]) {
				return &DoEmbedResult{
					Embeddings: dummyEmbeddings[2:],
					Warnings:   []Warning{warning2},
				}, nil
			}
			t.Fatal("unexpected values")
			return nil, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:                model,
			Values:               testValues,
			MaxEmbeddingsPerCall: 2,
			MaxParallelCalls:     1, // sequential to preserve order
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []Warning{warning1, warning2}
		if !reflect.DeepEqual(result.Warnings, expected) {
			t.Errorf("expected warnings %v, got %v", expected, result.Warnings)
		}
	})
}

func TestEmbedMany_ResultProviderMetadata(t *testing.T) {
	t.Run("should include provider metadata when returned by the model", func(t *testing.T) {
		providerMetadata := ProviderMetadata{
			"gateway": {"routing": map[string]any{"resolvedProvider": "test-provider"}},
		}

		model := newMockEmbeddingModel(func(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error) {
			return &DoEmbedResult{
				Embeddings:       dummyEmbeddings,
				Warnings:         []Warning{},
				ProviderMetadata: providerMetadata,
				Response: &EmbedResponseData{
					Headers: map[string]string{},
					Body:    map[string]any{},
				},
			}, nil
		})

		result, err := EmbedMany(context.Background(), EmbedManyOptions{
			Model:                model,
			Values:               testValues,
			MaxEmbeddingsPerCall: 3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.ProviderMetadata, providerMetadata) {
			t.Errorf("expected provider metadata %v, got %v", providerMetadata, result.ProviderMetadata)
		}
	})
}
