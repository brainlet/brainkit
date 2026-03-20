package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/bus"
)

func (k *Kit) handleMemory(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "memory.createThread":
		return k.handleMemoryCreateThread(ctx, msg)
	case "memory.getThread":
		return k.handleMemoryGetThread(ctx, msg)
	case "memory.listThreads":
		return k.handleMemoryListThreads(ctx, msg)
	case "memory.save":
		return k.handleMemorySave(ctx, msg)
	case "memory.recall":
		return k.handleMemoryRecall(ctx, msg)
	case "memory.deleteThread":
		return k.handleMemoryDeleteThread(ctx, msg)
	default:
		return nil, fmt.Errorf("memory: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleMemoryCreateThread(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__mem_req.js", fmt.Sprintf("globalThis.__mem_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__mem_createThread.ts", `
		var req = globalThis.__mem_pending_req || {};
		var mem = globalThis.__kit_memory;
		if (!mem) throw new Error("memory not configured — add Storages to Kit config and call createMemory()");
		var thread = await mem.createThread(req.opts);
		return JSON.stringify({ threadId: thread.id });
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.createThread: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleMemoryGetThread(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__mem_req.js", fmt.Sprintf("globalThis.__mem_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__mem_getThread.ts", `
		var req = globalThis.__mem_pending_req;
		var mem = globalThis.__kit_memory;
		if (!mem) throw new Error("memory not configured");
		var thread = await mem.getThread(req.threadId);
		return JSON.stringify(thread);
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.getThread: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleMemoryListThreads(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__mem_req.js", fmt.Sprintf("globalThis.__mem_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__mem_listThreads.ts", `
		var req = globalThis.__mem_pending_req || {};
		var mem = globalThis.__kit_memory;
		if (!mem) throw new Error("memory not configured");
		var threads = await mem.listThreads(req.filter);
		return JSON.stringify(threads);
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.listThreads: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleMemorySave(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__mem_req.js", fmt.Sprintf("globalThis.__mem_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__mem_save.ts", `
		var req = globalThis.__mem_pending_req;
		var mem = globalThis.__kit_memory;
		if (!mem) throw new Error("memory not configured");
		await mem.save(req.threadId, req.messages);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.save: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleMemoryRecall(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__mem_req.js", fmt.Sprintf("globalThis.__mem_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__mem_recall.ts", `
		var req = globalThis.__mem_pending_req;
		var mem = globalThis.__kit_memory;
		if (!mem) throw new Error("memory not configured");
		var result = await mem.recall(req.threadId, req.query);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.recall: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleMemoryDeleteThread(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__mem_req.js", fmt.Sprintf("globalThis.__mem_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__mem_deleteThread.ts", `
		var req = globalThis.__mem_pending_req;
		var mem = globalThis.__kit_memory;
		if (!mem) throw new Error("memory not configured");
		await mem.deleteThread(req.threadId);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.deleteThread: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}
