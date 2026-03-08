// Ported from: packages/ai/src/rerank/rerank.test.ts
package rerank

import (
	"context"
	"reflect"
	"testing"
)

// mockRerankingModel is a mock implementation of RerankingModel for testing.
type mockRerankingModel struct {
	provider string
	modelID  string
	doRerank func(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error)
}

func (m *mockRerankingModel) Provider() string { return m.provider }
func (m *mockRerankingModel) ModelID() string  { return m.modelID }
func (m *mockRerankingModel) DoRerank(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error) {
	return m.doRerank(ctx, opts)
}

func newMockRerankingModel(doRerank func(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error)) *mockRerankingModel {
	return &mockRerankingModel{
		provider: "mock-provider",
		modelID:  "mock-model-id",
		doRerank: doRerank,
	}
}

func TestRerank_StringDocuments(t *testing.T) {
	documents := []any{
		"sunny day at the beach",
		"rainy day in the city",
		"cloudy day in the mountains",
	}

	model := newMockRerankingModel(func(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error) {
		// Verify document type detection.
		if opts.Documents.Type != "text" {
			t.Errorf("expected document type text, got %q", opts.Documents.Type)
		}
		if len(opts.Documents.TextValues) != 3 {
			t.Errorf("expected 3 text values, got %d", len(opts.Documents.TextValues))
		}

		return &DoRerankResult{
			Ranking: []ModelRanking{
				{Index: 2, RelevanceScore: 0.9},
				{Index: 0, RelevanceScore: 0.8},
				{Index: 1, RelevanceScore: 0.7},
			},
			ProviderMetadata: ProviderMetadata{
				"aProvider": {"someResponseKey": "someResponseValue"},
			},
			Response: &DoRerankResponseData{
				Headers: map[string]string{"content-type": "application/json"},
				Body:    map[string]any{"id": "123"},
				ModelID: "mock-response-model-id",
				ID:      "mock-response-id",
			},
		}, nil
	})

	topN := 3
	result, err := Rerank(context.Background(), RerankOptions{
		Model:     model,
		Documents: documents,
		Query:     "rainy day",
		TopN:      &topN,
		ProviderOptions: map[string]map[string]any{
			"aProvider": {"someKey": "someValue"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should return the correct original documents", func(t *testing.T) {
		if !reflect.DeepEqual(result.OriginalDocuments, documents) {
			t.Errorf("expected original documents %v, got %v", documents, result.OriginalDocuments)
		}
	})

	t.Run("should return the correct reranked documents", func(t *testing.T) {
		reranked := result.RerankedDocuments()
		expected := []any{
			"cloudy day in the mountains",
			"sunny day at the beach",
			"rainy day in the city",
		}
		if !reflect.DeepEqual(reranked, expected) {
			t.Errorf("expected reranked documents %v, got %v", expected, reranked)
		}
	})

	t.Run("should return the correct ranking", func(t *testing.T) {
		if len(result.Ranking) != 3 {
			t.Fatalf("expected 3 rankings, got %d", len(result.Ranking))
		}
		if result.Ranking[0].OriginalIndex != 2 || result.Ranking[0].Score != 0.9 {
			t.Errorf("ranking[0] mismatch: got index=%d score=%f", result.Ranking[0].OriginalIndex, result.Ranking[0].Score)
		}
		if result.Ranking[1].OriginalIndex != 0 || result.Ranking[1].Score != 0.8 {
			t.Errorf("ranking[1] mismatch: got index=%d score=%f", result.Ranking[1].OriginalIndex, result.Ranking[1].Score)
		}
		if result.Ranking[2].OriginalIndex != 1 || result.Ranking[2].Score != 0.7 {
			t.Errorf("ranking[2] mismatch: got index=%d score=%f", result.Ranking[2].OriginalIndex, result.Ranking[2].Score)
		}
	})

	t.Run("should return the correct provider metadata", func(t *testing.T) {
		expected := ProviderMetadata{
			"aProvider": {"someResponseKey": "someResponseValue"},
		}
		if !reflect.DeepEqual(result.ProviderMetadata, expected) {
			t.Errorf("expected provider metadata %v, got %v", expected, result.ProviderMetadata)
		}
	})

	t.Run("should return the correct response", func(t *testing.T) {
		if result.Response.ID != "mock-response-id" {
			t.Errorf("expected response ID mock-response-id, got %q", result.Response.ID)
		}
		if result.Response.ModelID != "mock-response-model-id" {
			t.Errorf("expected model ID mock-response-model-id, got %q", result.Response.ModelID)
		}
		if result.Response.Headers["content-type"] != "application/json" {
			t.Errorf("expected content-type header, got %v", result.Response.Headers)
		}
	})
}

func TestRerank_ObjectDocuments(t *testing.T) {
	documents := []any{
		map[string]any{"id": "123", "name": "sunny day at the beach"},
		map[string]any{"id": "456", "name": "rainy day in the city"},
		map[string]any{"id": "789", "name": "cloudy day in the mountains"},
	}

	model := newMockRerankingModel(func(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error) {
		if opts.Documents.Type != "object" {
			t.Errorf("expected document type object, got %q", opts.Documents.Type)
		}

		return &DoRerankResult{
			Ranking: []ModelRanking{
				{Index: 2, RelevanceScore: 0.9},
				{Index: 0, RelevanceScore: 0.8},
				{Index: 1, RelevanceScore: 0.7},
			},
			ProviderMetadata: ProviderMetadata{
				"aProvider": {"someResponseKey": "someResponseValue"},
			},
			Response: &DoRerankResponseData{
				Headers: map[string]string{"content-type": "application/json"},
				Body:    map[string]any{"id": "123"},
				ModelID: "mock-response-model-id",
				ID:      "mock-response-id",
			},
		}, nil
	})

	topN := 3
	result, err := Rerank(context.Background(), RerankOptions{
		Model:     model,
		Documents: documents,
		Query:     "rainy day",
		TopN:      &topN,
		ProviderOptions: map[string]map[string]any{
			"aProvider": {"someKey": "someValue"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should return the correct original documents", func(t *testing.T) {
		if !reflect.DeepEqual(result.OriginalDocuments, documents) {
			t.Errorf("expected original documents %v, got %v", documents, result.OriginalDocuments)
		}
	})

	t.Run("should return the correct reranked documents", func(t *testing.T) {
		reranked := result.RerankedDocuments()
		expected := []any{
			map[string]any{"id": "789", "name": "cloudy day in the mountains"},
			map[string]any{"id": "123", "name": "sunny day at the beach"},
			map[string]any{"id": "456", "name": "rainy day in the city"},
		}
		if !reflect.DeepEqual(reranked, expected) {
			t.Errorf("expected reranked documents %v, got %v", expected, reranked)
		}
	})

	t.Run("should return the correct ranking", func(t *testing.T) {
		if len(result.Ranking) != 3 {
			t.Fatalf("expected 3 rankings, got %d", len(result.Ranking))
		}
		doc0, ok := result.Ranking[0].Document.(map[string]any)
		if !ok || doc0["id"] != "789" {
			t.Errorf("ranking[0] document mismatch")
		}
	})
}

func TestRerank_EmptyDocuments(t *testing.T) {
	t.Run("should return empty result for empty documents", func(t *testing.T) {
		model := newMockRerankingModel(func(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error) {
			t.Fatal("should not be called for empty documents")
			return nil, nil
		})

		result, err := Rerank(context.Background(), RerankOptions{
			Model:     model,
			Documents: []any{},
			Query:     "rainy day",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.OriginalDocuments) != 0 {
			t.Errorf("expected 0 original documents, got %d", len(result.OriginalDocuments))
		}
		if len(result.Ranking) != 0 {
			t.Errorf("expected 0 rankings, got %d", len(result.Ranking))
		}
		if result.Response.ModelID != "mock-model-id" {
			t.Errorf("expected model ID mock-model-id, got %q", result.Response.ModelID)
		}
	})
}
