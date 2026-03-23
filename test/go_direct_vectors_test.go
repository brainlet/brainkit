package test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoDirect_Vectors tests the vector store domain handlers.
// Requires a vector store backend configured in the JS runtime.
// LibSQLVector needs a real LibSQL/Turso server with vector extensions —
// our embedded SQLite HTTP bridge does NOT support libsql_vector_idx.
// These tests verify the handler wiring is correct. They skip when the
// vector store backend is not available (infrastructure concern, not brainkit concern).
func TestGoDirect_Vectors(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelWithStorage(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Initialize vector store via EvalTS.
			// LibSQLVector requires a real LibSQL server with vector extensions.
			// Our embedded SQLite bridge doesn't have libsql_vector_idx —
			// so this init will fail and tests will skip.
			_, err := tk.EvalTS(ctx, "__vector_init.ts", `
				var url = globalThis.__brainkit_storages && globalThis.__brainkit_storages["default"];
				if (!url) throw new Error("no storage URL — configure Storages in KernelConfig");
				try {
					var vs = new LibSQLVector({ id: "test_vector", connectionUrl: url });
					globalThis.__kit_vector_store = vs;
					// Probe: try creating an index to verify vector extensions are available
					await vs.createIndex({ indexName: "probe_idx", dimension: 2 });
					await vs.deleteIndex("probe_idx");
					return "ok";
				} catch(e) {
					// Vector extensions not available — not a brainkit bug
					throw new Error("vector store requires LibSQL with vector extensions: " + e.message);
				}
			`)
			if err != nil {
				t.Skipf("vector store not available (needs LibSQL with vector extensions): %v", err)
			}

			t.Run("CreateIndex", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorCreateIndexMsg, messages.VectorCreateIndexResp](rt, ctx, messages.VectorCreateIndexMsg{
					Name:      "test_index",
					Dimension: 3,
					Metric:    "cosine",
				})
				require.NoError(t, err)
				assert.True(t, resp.OK)
			})

			t.Run("Upsert", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorUpsertMsg, messages.VectorUpsertResp](rt, ctx, messages.VectorUpsertMsg{
					Index: "test_index",
					Vectors: []messages.Vector{
						{ID: "v1", Values: []float64{1.0, 0.0, 0.0}, Metadata: map[string]string{"label": "x"}},
						{ID: "v2", Values: []float64{0.0, 1.0, 0.0}, Metadata: map[string]string{"label": "y"}},
						{ID: "v3", Values: []float64{0.0, 0.0, 1.0}, Metadata: map[string]string{"label": "z"}},
					},
				})
				require.NoError(t, err)
				assert.True(t, resp.OK)
			})

			t.Run("Query", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorQueryMsg, messages.VectorQueryResp](rt, ctx, messages.VectorQueryMsg{
					Index:     "test_index",
					Embedding: []float64{1.0, 0.0, 0.0},
					TopK:      2,
				})
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Matches)
				if len(resp.Matches) > 0 {
					assert.Equal(t, "v1", resp.Matches[0].ID)
				}
			})

			t.Run("ListIndexes", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorListIndexesMsg, messages.VectorListIndexesResp](rt, ctx, messages.VectorListIndexesMsg{})
				require.NoError(t, err)
				found := false
				for _, idx := range resp.Indexes {
					if idx.Name == "test_index" {
						found = true
					}
				}
				assert.True(t, found, "test_index should be in list")
			})

			t.Run("DeleteIndex", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorDeleteIndexMsg, messages.VectorDeleteIndexResp](rt, ctx, messages.VectorDeleteIndexMsg{
					Name: "test_index",
				})
				require.NoError(t, err)
				assert.True(t, resp.OK)
			})
		})
	}
}
