package togetherai

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// rerankingFixtureResponse is the canned response matching togetherai-reranking.1.json
var rerankingFixtureResponse = map[string]interface{}{
	"id":     "oGs6Zt9-62bZhn-99529372487b1b0a",
	"object": "rerank",
	"model":  "Salesforce/Llama-Rank-v1",
	"results": []map[string]interface{}{
		{
			"index":           0,
			"relevance_score": 0.6475887154399037,
			"document":        map[string]interface{}{},
		},
		{
			"index":           5,
			"relevance_score": 0.6323295373206566,
			"document":        map[string]interface{}{},
		},
	},
	"usage": map[string]interface{}{
		"prompt_tokens":     2966,
		"completion_tokens": 0,
		"total_tokens":      2966,
	},
}

// newRerankingTestServer creates an httptest server for reranking tests.
func newRerankingTestServer(t *testing.T, response interface{}, statusCode int) (*httptest.Server, *[]*http.Request) {
	t.Helper()
	var requests []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		requests = append(requests, r)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

// createTestRerankingModel creates a TogetherAIRerankingModel pointing at the test server.
func createTestRerankingModel(serverURL string) *TogetherAIRerankingModel {
	return NewTogetherAIRerankingModel("Salesforce/Llama-Rank-v1", TogetherAIRerankingConfig{
		Provider: "togetherai.reranking",
		BaseURL:  serverURL,
		Headers: func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer test-api-key",
			}
		},
	})
}

// getRerankRequestBody reads the request body JSON from a stored request.
func getRerankRequestBody(t *testing.T, requests []*http.Request, index int) map[string]interface{} {
	t.Helper()
	if index >= len(requests) {
		t.Fatalf("expected at least %d request(s), got %d", index+1, len(requests))
	}
	body, err := io.ReadAll(requests[index].Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse request body JSON: %v", err)
	}
	return result
}

func TestRerankingModel_DoRerank_JSONDocuments(t *testing.T) {
	server, requests := newRerankingTestServer(t, rerankingFixtureResponse, 200)
	model := createTestRerankingModel(server.URL)
	topN := 2

	result, err := model.DoRerank(rerankingmodel.CallOptions{
		Documents: rerankingmodel.DocumentsObject{
			Values: []jsonvalue.JSONObject{
				{"example": "sunny day at the beach"},
				{"example": "rainy day in the city"},
			},
		},
		Query: "rainy day",
		TopN:  &topN,
		ProviderOptions: shared.ProviderOptions{
			"togetherai": {
				"rankFields": []interface{}{"example"},
			},
		},
		Ctx: context.Background(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should send request with json documents", func(t *testing.T) {
		body := getRerankRequestBody(t, *requests, 0)

		if body["model"] != "Salesforce/Llama-Rank-v1" {
			t.Errorf("expected model 'Salesforce/Llama-Rank-v1', got %v", body["model"])
		}
		if body["query"] != "rainy day" {
			t.Errorf("expected query 'rainy day', got %v", body["query"])
		}
		if body["top_n"] != float64(2) {
			t.Errorf("expected top_n=2, got %v", body["top_n"])
		}
		if body["return_documents"] != false {
			t.Errorf("expected return_documents=false, got %v", body["return_documents"])
		}

		docs, ok := body["documents"].([]interface{})
		if !ok {
			t.Fatalf("expected documents to be an array, got %T", body["documents"])
		}
		if len(docs) != 2 {
			t.Fatalf("expected 2 documents, got %d", len(docs))
		}

		doc0, ok := docs[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected doc[0] to be a map, got %T", docs[0])
		}
		if doc0["example"] != "sunny day at the beach" {
			t.Errorf("expected doc[0].example 'sunny day at the beach', got %v", doc0["example"])
		}

		rankFields, ok := body["rank_fields"].([]interface{})
		if !ok {
			t.Fatalf("expected rank_fields to be an array, got %T", body["rank_fields"])
		}
		if len(rankFields) != 1 || rankFields[0] != "example" {
			t.Errorf("expected rank_fields=['example'], got %v", rankFields)
		}
	})

	t.Run("should send request with the correct headers", func(t *testing.T) {
		req := (*requests)[0]
		if req.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q", req.Header.Get("Authorization"))
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", req.Header.Get("Content-Type"))
		}
	})

	t.Run("should return result without warnings", func(t *testing.T) {
		if result.Warnings != nil {
			t.Errorf("expected nil warnings, got %v", result.Warnings)
		}
	})

	t.Run("should return result with the correct ranking", func(t *testing.T) {
		if len(result.Ranking) != 2 {
			t.Fatalf("expected 2 ranked documents, got %d", len(result.Ranking))
		}
		if result.Ranking[0].Index != 0 {
			t.Errorf("expected ranking[0].Index=0, got %d", result.Ranking[0].Index)
		}
		if !almostEqual(result.Ranking[0].RelevanceScore, 0.6475887154399037) {
			t.Errorf("expected ranking[0].RelevanceScore~=0.6475887154399037, got %f", result.Ranking[0].RelevanceScore)
		}
		if result.Ranking[1].Index != 5 {
			t.Errorf("expected ranking[1].Index=5, got %d", result.Ranking[1].Index)
		}
		if !almostEqual(result.Ranking[1].RelevanceScore, 0.6323295373206566) {
			t.Errorf("expected ranking[1].RelevanceScore~=0.6323295373206566, got %f", result.Ranking[1].RelevanceScore)
		}
	})

	t.Run("should not return provider metadata", func(t *testing.T) {
		if result.ProviderMetadata != nil {
			t.Errorf("expected nil providerMetadata, got %v", result.ProviderMetadata)
		}
	})

	t.Run("should return result with the correct response", func(t *testing.T) {
		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.ID == nil || *result.Response.ID != "oGs6Zt9-62bZhn-99529372487b1b0a" {
			t.Errorf("expected response ID 'oGs6Zt9-62bZhn-99529372487b1b0a', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "Salesforce/Llama-Rank-v1" {
			t.Errorf("expected response modelId 'Salesforce/Llama-Rank-v1', got %v", result.Response.ModelID)
		}
		if result.Response.Headers == nil {
			t.Error("expected non-nil response headers")
		}
		if result.Response.Body == nil {
			t.Error("expected non-nil response body")
		}
	})
}

func TestRerankingModel_DoRerank_TextDocuments(t *testing.T) {
	server, requests := newRerankingTestServer(t, rerankingFixtureResponse, 200)
	model := createTestRerankingModel(server.URL)
	topN := 2

	result, err := model.DoRerank(rerankingmodel.CallOptions{
		Documents: rerankingmodel.DocumentsText{
			Values: []string{"sunny day at the beach", "rainy day in the city"},
		},
		Query: "rainy day",
		TopN:  &topN,
		ProviderOptions: shared.ProviderOptions{
			"togetherai": {
				"rankFields": []interface{}{"example"},
			},
		},
		Ctx: context.Background(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("should send request with text documents", func(t *testing.T) {
		body := getRerankRequestBody(t, *requests, 0)

		if body["model"] != "Salesforce/Llama-Rank-v1" {
			t.Errorf("expected model 'Salesforce/Llama-Rank-v1', got %v", body["model"])
		}
		if body["query"] != "rainy day" {
			t.Errorf("expected query 'rainy day', got %v", body["query"])
		}
		if body["top_n"] != float64(2) {
			t.Errorf("expected top_n=2, got %v", body["top_n"])
		}
		if body["return_documents"] != false {
			t.Errorf("expected return_documents=false, got %v", body["return_documents"])
		}

		docs, ok := body["documents"].([]interface{})
		if !ok {
			t.Fatalf("expected documents to be an array, got %T", body["documents"])
		}
		if len(docs) != 2 {
			t.Fatalf("expected 2 documents, got %d", len(docs))
		}
		if docs[0] != "sunny day at the beach" {
			t.Errorf("expected doc[0]='sunny day at the beach', got %v", docs[0])
		}
		if docs[1] != "rainy day in the city" {
			t.Errorf("expected doc[1]='rainy day in the city', got %v", docs[1])
		}
	})

	t.Run("should send request with the correct headers", func(t *testing.T) {
		req := (*requests)[0]
		if req.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q", req.Header.Get("Authorization"))
		}
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", req.Header.Get("Content-Type"))
		}
	})

	t.Run("should return result without warnings", func(t *testing.T) {
		if result.Warnings != nil {
			t.Errorf("expected nil warnings, got %v", result.Warnings)
		}
	})

	t.Run("should return result with the correct ranking", func(t *testing.T) {
		if len(result.Ranking) != 2 {
			t.Fatalf("expected 2 ranked documents, got %d", len(result.Ranking))
		}
		if result.Ranking[0].Index != 0 {
			t.Errorf("expected ranking[0].Index=0, got %d", result.Ranking[0].Index)
		}
		if !almostEqual(result.Ranking[0].RelevanceScore, 0.6475887154399037) {
			t.Errorf("expected ranking[0].RelevanceScore~=0.6475887154399037, got %f", result.Ranking[0].RelevanceScore)
		}
		if result.Ranking[1].Index != 5 {
			t.Errorf("expected ranking[1].Index=5, got %d", result.Ranking[1].Index)
		}
		if !almostEqual(result.Ranking[1].RelevanceScore, 0.6323295373206566) {
			t.Errorf("expected ranking[1].RelevanceScore~=0.6323295373206566, got %f", result.Ranking[1].RelevanceScore)
		}
	})

	t.Run("should not return provider metadata", func(t *testing.T) {
		if result.ProviderMetadata != nil {
			t.Errorf("expected nil providerMetadata, got %v", result.ProviderMetadata)
		}
	})

	t.Run("should return result with the correct response", func(t *testing.T) {
		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.ID == nil || *result.Response.ID != "oGs6Zt9-62bZhn-99529372487b1b0a" {
			t.Errorf("expected response ID 'oGs6Zt9-62bZhn-99529372487b1b0a', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "Salesforce/Llama-Rank-v1" {
			t.Errorf("expected response modelId 'Salesforce/Llama-Rank-v1', got %v", result.Response.ModelID)
		}
		if result.Response.Headers == nil {
			t.Error("expected non-nil response headers")
		}
		if result.Response.Body == nil {
			t.Error("expected non-nil response body")
		}
	})
}

// almostEqual checks if two float64 values are approximately equal.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-12
}
