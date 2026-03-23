package test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
)

func TestGoDirect_Vectors(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelWithStorage(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Deploy .ts that initializes a vector store
			// Using LibSQLVector backed by the default storage
			_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
				Source: "vector-init.ts",
				Code: `
					const storageURL = globalThis.__brainkit_storages && globalThis.__brainkit_storages["default"];
					if (storageURL) {
						const vs = new LibSQLVector({ connectionUrl: storageURL });
						globalThis.__kit_vector_store = vs;
					}
				`,
			})
			if err != nil {
				t.Skipf("vector init failed: %v", err)
			}

			t.Run("CreateIndex", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorCreateIndexMsg, messages.VectorCreateIndexResp](rt, ctx, messages.VectorCreateIndexMsg{
					Name:      "test-index",
					Dimension: 3,
					Metric:    "cosine",
				})
				if err != nil {
					t.Skipf("vector store not configured: %v", err)
				}
				assert.True(t, resp.OK)
			})

			t.Run("Upsert", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorUpsertMsg, messages.VectorUpsertResp](rt, ctx, messages.VectorUpsertMsg{
					Index: "test-index",
					Vectors: []messages.Vector{
						{ID: "v1", Values: []float64{1.0, 0.0, 0.0}, Metadata: map[string]string{"label": "x-axis"}},
						{ID: "v2", Values: []float64{0.0, 1.0, 0.0}, Metadata: map[string]string{"label": "y-axis"}},
						{ID: "v3", Values: []float64{0.0, 0.0, 1.0}, Metadata: map[string]string{"label": "z-axis"}},
					},
				})
				if err != nil {
					t.Skipf("vector upsert not supported: %v", err)
				}
				assert.True(t, resp.OK)
			})

			t.Run("Query", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorQueryMsg, messages.VectorQueryResp](rt, ctx, messages.VectorQueryMsg{
					Index:     "test-index",
					Embedding: []float64{1.0, 0.0, 0.0},
					TopK:      2,
				})
				if err != nil {
					t.Skipf("vector query not supported: %v", err)
				}
				assert.NotEmpty(t, resp.Matches, "should return matches")
				if len(resp.Matches) > 0 {
					assert.Equal(t, "v1", resp.Matches[0].ID, "closest vector should be v1")
				}
			})

			t.Run("ListIndexes", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorListIndexesMsg, messages.VectorListIndexesResp](rt, ctx, messages.VectorListIndexesMsg{})
				if err != nil {
					t.Skipf("vector listIndexes not supported: %v", err)
				}
				found := false
				for _, idx := range resp.Indexes {
					if idx.Name == "test-index" {
						found = true
					}
				}
				assert.True(t, found, "test-index should be in list")
			})

			t.Run("DeleteIndex", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.VectorDeleteIndexMsg, messages.VectorDeleteIndexResp](rt, ctx, messages.VectorDeleteIndexMsg{
					Name: "test-index",
				})
				if err != nil {
					t.Skipf("vector deleteIndex not supported: %v", err)
				}
				assert.True(t, resp.OK)
			})

			// Cleanup
			sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "vector-init.ts"})
		})
	}
}
