package kit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// VectorsDomain handles vector store operations.
type VectorsDomain struct {
	kit *Kernel
}

func newVectorsDomain(k *Kernel) *VectorsDomain {
	return &VectorsDomain{kit: k}
}

func (d *VectorsDomain) CreateIndex(ctx context.Context, req messages.VectorCreateIndexMsg) (*messages.VectorCreateIndexResp, error) {
	_, err := d.kit.evalDomain(ctx, req, "__vec_createIndex.ts", `
		var req = globalThis.__pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		await vs.createIndex({ indexName: req.name, dimension: req.dimension });
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.createIndex: %w", err)
	}
	return &messages.VectorCreateIndexResp{OK: true}, nil
}

func (d *VectorsDomain) DeleteIndex(ctx context.Context, req messages.VectorDeleteIndexMsg) (*messages.VectorDeleteIndexResp, error) {
	_, err := d.kit.evalDomain(ctx, req, "__vec_deleteIndex.ts", `
		var req = globalThis.__pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		await vs.deleteIndex(req.name);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.deleteIndex: %w", err)
	}
	return &messages.VectorDeleteIndexResp{OK: true}, nil
}

func (d *VectorsDomain) ListIndexes(ctx context.Context, req messages.VectorListIndexesMsg) (*messages.VectorListIndexesResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__vec_listIndexes.ts", `
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		var indexes = await vs.listIndexes();
		return JSON.stringify(indexes);
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.listIndexes: %w", err)
	}
	var indexes []messages.VectorIndexInfo
	json.Unmarshal(raw, &indexes)
	return &messages.VectorListIndexesResp{Indexes: indexes}, nil
}

func (d *VectorsDomain) Upsert(ctx context.Context, req messages.VectorUpsertMsg) (*messages.VectorUpsertResp, error) {
	_, err := d.kit.evalDomain(ctx, req, "__vec_upsert.ts", `
		var req = globalThis.__pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		await vs.upsert({
			indexName: req.index,
			vectors: req.vectors.map(function(v) {
				return { id: v.id, vector: v.values, metadata: v.metadata };
			}),
		});
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.upsert: %w", err)
	}
	return &messages.VectorUpsertResp{OK: true}, nil
}

func (d *VectorsDomain) Query(ctx context.Context, req messages.VectorQueryMsg) (*messages.VectorQueryResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__vec_query.ts", `
		var req = globalThis.__pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		var results = await vs.query({
			indexName: req.index,
			queryVector: req.embedding,
			topK: req.topK,
		});
		var matches = (results || []).map(function(r) {
			return { id: r.id, score: r.score, values: r.vector, metadata: r.metadata };
		});
		return JSON.stringify({ matches: matches });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.query: %w", err)
	}
	var resp messages.VectorQueryResp
	json.Unmarshal(raw, &resp)
	return &resp, nil
}
