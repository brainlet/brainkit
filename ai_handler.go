package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/bus"
)

func (k *Kit) handleAI(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "ai.generate":
		return k.handleAiGenerate(ctx, msg)
	case "ai.embed":
		return k.handleAiEmbed(ctx, msg)
	case "ai.embedMany":
		return k.handleAiEmbedMany(ctx, msg)
	case "ai.generateObject":
		return k.handleAiGenerateObject(ctx, msg)
	default:
		return nil, fmt.Errorf("ai: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleAiGenerate(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	// Set request as global variable (safe — no JS injection)
	k.bridge.Eval("__ai_req.js", fmt.Sprintf("globalThis.__ai_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__ai_generate.ts", `
		var req = globalThis.__ai_pending_req;
		var result = await ai.generate(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.generate: %w", err)
	}

	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleAiEmbed(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__ai_req.js", fmt.Sprintf("globalThis.__ai_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__ai_embed.ts", `
		var req = globalThis.__ai_pending_req;
		var result = await ai.embed(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.embed: %w", err)
	}

	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleAiEmbedMany(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__ai_req.js", fmt.Sprintf("globalThis.__ai_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__ai_embedmany.ts", `
		var req = globalThis.__ai_pending_req;
		var result = await ai.embedMany(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.embedMany: %w", err)
	}

	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleAiGenerateObject(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__ai_req.js", fmt.Sprintf("globalThis.__ai_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__ai_generateobject.ts", `
		var req = globalThis.__ai_pending_req;
		var result = await ai.generateObject(req);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("ai.generateObject: %w", err)
	}

	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}
