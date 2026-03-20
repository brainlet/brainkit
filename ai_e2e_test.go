package brainkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

// These tests use real LLM APIs — they skip if OPENAI_API_KEY is not set.

func TestAIHandler_Generate_RealLLM(t *testing.T) {
	kit := newTestKit(t) // skips if no API key

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "ai.generate",
		Payload: json.RawMessage(`{"model":"openai/gpt-4o-mini","prompt":"Reply with exactly one word: PONG"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ Text string `json:"text"` }
	json.Unmarshal(resp.Payload, &result)
	// Verify we got a non-empty response — content varies with LLM non-determinism
	if result.Text == "" {
		t.Fatalf("expected non-empty text, got: %s", resp.Payload)
	}
	t.Logf("generated: %s", result.Text)
}

func TestAIHandler_Embed_RealLLM(t *testing.T) {
	kit := newTestKit(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "ai.embed",
		Payload: json.RawMessage(`{"model":"openai/text-embedding-3-small","value":"hello world"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	json.Unmarshal(resp.Payload, &result)
	if len(result.Embedding) == 0 {
		t.Fatalf("expected non-empty embedding, got: %s", resp.Payload)
	}
	t.Logf("embedding dimension: %d", len(result.Embedding))
}

func TestAIHandler_EmbedMany_RealLLM(t *testing.T) {
	kit := newTestKit(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "ai.embedMany",
		Payload: json.RawMessage(`{"model":"openai/text-embedding-3-small","values":["hello","world","test"]}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	json.Unmarshal(resp.Payload, &result)
	if len(result.Embeddings) != 3 {
		t.Fatalf("expected 3 embeddings, got %d: %s", len(result.Embeddings), resp.Payload)
	}
	for i, emb := range result.Embeddings {
		if len(emb) == 0 {
			t.Errorf("embedding %d is empty", i)
		}
	}
}

func TestAIHandler_GenerateObject_RealLLM(t *testing.T) {
	kit := newTestKit(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic: "ai.generateObject",
		Payload: json.RawMessage(`{
			"model": "openai/gpt-4o-mini",
			"prompt": "Generate a person with name and age",
			"schema": {
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"age": {"type": "integer"}
				},
				"required": ["name", "age"]
			}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		Object struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		} `json:"object"`
	}
	json.Unmarshal(resp.Payload, &result)
	if result.Object.Name == "" {
		t.Fatalf("expected name, got: %s", resp.Payload)
	}
	t.Logf("generated: %s, age %d", result.Object.Name, result.Object.Age)
}
