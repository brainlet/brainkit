package kit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// MemoryDomain handles thread-based conversation memory.
type MemoryDomain struct {
	kit *Kernel
}

func newMemoryDomain(k *Kernel) *MemoryDomain {
	return &MemoryDomain{kit: k}
}

// resolveMemory is the JS snippet that resolves the memory instance.
// Tries globalThis.__kit_memory first (set by .ts code via createMemory()).
// Falls back to auto-initializing from a registered "default" storage if available.
const resolveMemory = `
	var mem = globalThis.__kit_memory;
	if (!mem && typeof registry !== "undefined" && registry.has("storage", "default")) {
		mem = createMemory({ storage: storage("default") });
		globalThis.__kit_memory = mem;
	}
	if (!mem) throw new Error("memory not configured — register a storage or call createMemory() in .ts");
`

func (d *MemoryDomain) CreateThread(ctx context.Context, req messages.MemoryCreateThreadMsg) (*messages.MemoryCreateThreadResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__mem_createThread.ts", `
		var req = globalThis.__pending_req || {};
		`+resolveMemory+`
		var opts = req.opts || {};
		var thread = await mem.createThread(opts);
		return JSON.stringify({ threadId: thread.id });
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.createThread: %w", err)
	}
	var resp messages.MemoryCreateThreadResp
	json.Unmarshal(raw, &resp)
	return &resp, nil
}

func (d *MemoryDomain) GetThread(ctx context.Context, req messages.MemoryGetThreadMsg) (*messages.MemoryGetThreadResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__mem_getThread.ts", `
		var req = globalThis.__pending_req;
		`+resolveMemory+`
		var thread = await mem.getThreadById({ threadId: req.threadId });
		return JSON.stringify(thread);
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.getThread: %w", err)
	}
	return &messages.MemoryGetThreadResp{Thread: raw}, nil
}

func (d *MemoryDomain) ListThreads(ctx context.Context, req messages.MemoryListThreadsMsg) (*messages.MemoryListThreadsResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__mem_listThreads.ts", `
		var req = globalThis.__pending_req || {};
		`+resolveMemory+`
		var filter = req.filter || {};
		var result = await mem.listThreads(filter);
		return JSON.stringify(result.threads || result);
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.listThreads: %w", err)
	}
	return &messages.MemoryListThreadsResp{Threads: raw}, nil
}

func (d *MemoryDomain) Save(ctx context.Context, req messages.MemorySaveMsg) (*messages.MemorySaveResp, error) {
	_, err := d.kit.evalDomain(ctx, req, "__mem_save.ts", `
		var req = globalThis.__pending_req;
		`+resolveMemory+`
		await mem.saveMessages({ threadId: req.threadId, messages: req.messages });
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.save: %w", err)
	}
	return &messages.MemorySaveResp{OK: true}, nil
}

func (d *MemoryDomain) Recall(ctx context.Context, req messages.MemoryRecallMsg) (*messages.MemoryRecallResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__mem_recall.ts", `
		var req = globalThis.__pending_req;
		`+resolveMemory+`
		var result = await mem.recall({
			threadId: req.threadId,
			resourceId: req.resourceId || "",
			query: req.query || "",
		});
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.recall: %w", err)
	}
	var resp messages.MemoryRecallResp
	json.Unmarshal(raw, &resp)
	return &resp, nil
}

func (d *MemoryDomain) DeleteThread(ctx context.Context, req messages.MemoryDeleteThreadMsg) (*messages.MemoryDeleteThreadResp, error) {
	_, err := d.kit.evalDomain(ctx, req, "__mem_deleteThread.ts", `
		var req = globalThis.__pending_req;
		`+resolveMemory+`
		await mem.deleteThread(req.threadId);
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("memory.deleteThread: %w", err)
	}
	return &messages.MemoryDeleteThreadResp{OK: true}, nil
}
