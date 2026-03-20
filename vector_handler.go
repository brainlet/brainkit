package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/bus"
)

func (k *Kit) handleVectors(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "vectors.upsert":
		return k.handleVectorUpsert(ctx, msg)
	case "vectors.query":
		return k.handleVectorQuery(ctx, msg)
	case "vectors.createIndex":
		return k.handleVectorCreateIndex(ctx, msg)
	case "vectors.deleteIndex":
		return k.handleVectorDeleteIndex(ctx, msg)
	case "vectors.listIndexes":
		return k.handleVectorListIndexes(ctx, msg)
	default:
		return nil, fmt.Errorf("vectors: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleVectorCreateIndex(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__vec_req.js", fmt.Sprintf("globalThis.__vec_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__vec_createIndex.ts", `
		var req = globalThis.__vec_pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured — add a vector store to Kit config");
		await vs.createIndex(req.name, req.dimension, req.metric);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.createIndex: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleVectorDeleteIndex(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__vec_req.js", fmt.Sprintf("globalThis.__vec_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__vec_deleteIndex.ts", `
		var req = globalThis.__vec_pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		await vs.deleteIndex(req.name);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.deleteIndex: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleVectorListIndexes(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	resultJSON, err := k.EvalTS(ctx, "__vec_listIndexes.ts", `
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		var indexes = await vs.listIndexes();
		return JSON.stringify(indexes);
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.listIndexes: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleVectorUpsert(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__vec_req.js", fmt.Sprintf("globalThis.__vec_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__vec_upsert.ts", `
		var req = globalThis.__vec_pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		await vs.upsert(req.index, req.vectors);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.upsert: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleVectorQuery(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__vec_req.js", fmt.Sprintf("globalThis.__vec_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__vec_query.ts", `
		var req = globalThis.__vec_pending_req;
		var vs = globalThis.__kit_vector_store;
		if (!vs) throw new Error("vector store not configured");
		var matches = await vs.query(req.index, req.embedding, req.topK, req.filter);
		return JSON.stringify({ matches: matches });
	`)
	if err != nil {
		return nil, fmt.Errorf("vectors.query: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}
