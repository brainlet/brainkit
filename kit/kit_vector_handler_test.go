package kit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/internal/bus"
)

func setupMockVectorStore(t *testing.T, kit *Kit) {
	t.Helper()
	_, err := kit.EvalTS(context.Background(), "__mock_vectors.ts", `
		var __mock_indexes = {};

		globalThis.__kit_vector_store = {
			createIndex: async function(name, dimension, metric) {
				__mock_indexes[name] = { name: name, dimension: dimension, metric: metric, vectors: [] };
			},
			deleteIndex: async function(name) {
				delete __mock_indexes[name];
			},
			listIndexes: async function() {
				return Object.values(__mock_indexes).map(function(idx) {
					return { name: idx.name, dimension: idx.dimension };
				});
			},
			upsert: async function(index, vectors) {
				var idx = __mock_indexes[index];
				if (!idx) throw new Error("index '" + index + "' not found");
				for (var i = 0; i < vectors.length; i++) {
					idx.vectors.push(vectors[i]);
				}
			},
			query: async function(index, embedding, topK, filter) {
				var idx = __mock_indexes[index];
				if (!idx) throw new Error("index '" + index + "' not found");
				// Simple mock: return first topK vectors with score 1.0
				return idx.vectors.slice(0, topK).map(function(v) {
					return { id: v.id, score: 1.0, values: v.values, metadata: v.metadata };
				});
			},
		};
		return "ok";
	`)
	if err != nil {
		t.Fatalf("setup mock vector store: %v", err)
	}
}

func TestVectorHandler_CreateIndexAndQuery(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockVectorStore(t, kit)

	// createIndex
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.createIndex",
		Payload: json.RawMessage(`{"name":"test-idx","dimension":3,"metric":"cosine"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var createResult struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &createResult)
	if !createResult.OK {
		t.Fatalf("createIndex: %s", resp.Payload)
	}

	// upsert
	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.upsert",
		Payload: json.RawMessage(`{"index":"test-idx","vectors":[{"id":"v1","values":[1,0,0]},{"id":"v2","values":[0,1,0]}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// query
	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.query",
		Payload: json.RawMessage(`{"index":"test-idx","embedding":[1,0,0],"topK":1}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var queryResult struct {
		Matches []struct {
			ID    string  `json:"id"`
			Score float64 `json:"score"`
		} `json:"matches"`
	}
	json.Unmarshal(resp.Payload, &queryResult)
	if len(queryResult.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d: %s", len(queryResult.Matches), resp.Payload)
	}
	if queryResult.Matches[0].ID != "v1" {
		t.Errorf("expected v1, got %s", queryResult.Matches[0].ID)
	}
}

func TestVectorHandler_ListIndexes(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockVectorStore(t, kit)

	// Create 2 indexes
	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.createIndex",
		Payload: json.RawMessage(`{"name":"idx-a","dimension":3,"metric":"cosine"}`),
	})
	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.createIndex",
		Payload: json.RawMessage(`{"name":"idx-b","dimension":128,"metric":"euclidean"}`),
	})

	// List
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.listIndexes",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var indexes []struct{ Name string `json:"name"` }
	json.Unmarshal(resp.Payload, &indexes)
	if len(indexes) != 2 {
		t.Fatalf("expected 2 indexes, got %d: %s", len(indexes), resp.Payload)
	}
}

func TestVectorHandler_DeleteIndex(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockVectorStore(t, kit)

	// Create then delete
	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.createIndex",
		Payload: json.RawMessage(`{"name":"temp","dimension":3,"metric":"cosine"}`),
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.deleteIndex",
		Payload: json.RawMessage(`{"name":"temp"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var delResult struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &delResult)
	if !delResult.OK {
		t.Fatalf("deleteIndex: %s", resp.Payload)
	}

	// Verify gone
	resp, _ = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.listIndexes",
		Payload: json.RawMessage(`{}`),
	})
	var indexes []any
	json.Unmarshal(resp.Payload, &indexes)
	if len(indexes) != 0 {
		t.Errorf("expected 0 indexes after delete, got %d", len(indexes))
	}
}

func TestVectorHandler_UnknownTopic(t *testing.T) {
	kit := newTestKitNoKey(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "vectors.bogus",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected error for unknown vectors topic")
	}
}
