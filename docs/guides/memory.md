# Memory

Memory provides thread-based conversation context for agents. Each thread stores a sequence of messages (user/assistant/system) that can be saved, recalled, and managed. The bus handlers route through EvalTS to Mastra's Memory API.

> **Prerequisite**: Memory requires a storage backend. Add `Storages` to your Kit config and call `createMemory()` in your deployment code. See the [storage guide](storage.md) for provider options.

---

## Bus Topics

| Topic | Payload | Response |
|-------|---------|----------|
| `memory.createThread` | `{"opts":{"title":"...","metadata":{...}}}` | `{"threadId":"abc-123"}` |
| `memory.getThread` | `{"threadId":"abc-123"}` | `{"id":"abc-123","title":"...","metadata":{...}}` |
| `memory.listThreads` | `{"filter":{"title":"...","limit":10}}` | `[{"id":"abc-123","title":"..."}]` |
| `memory.save` | `{"threadId":"abc-123","messages":[{"role":"user","content":"..."}]}` | `{"ok":true}` |
| `memory.recall` | `{"threadId":"abc-123","query":"..."}` | `{"messages":[...]}` |
| `memory.deleteThread` | `{"threadId":"abc-123"}` | `{"ok":true}` |

All topics require a configured memory instance (`globalThis.__kit_memory`). If memory is not configured, handlers return an error.

---

## Usage from Go (via bus.AskSync)

```go
ctx := context.Background()

// Create a thread
resp, _ := bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:   "memory.createThread",
    Payload: json.RawMessage(`{"opts":{"title":"Support chat"}}`),
})
// resp.Payload: {"threadId":"abc-123"}

// Save messages
bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic: "memory.save",
    Payload: json.RawMessage(`{
        "threadId":"abc-123",
        "messages":[
            {"role":"user","content":"How do I deploy an agent?"},
            {"role":"assistant","content":"Use kit.Deploy(ctx, source, code)."}
        ]
    }`),
})

// Recall messages
resp, _ = bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:   "memory.recall",
    Payload: json.RawMessage(`{"threadId":"abc-123"}`),
})
// resp.Payload: {"messages":[...]}

// List threads
resp, _ = bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:   "memory.listThreads",
    Payload: json.RawMessage(`{"filter":{"limit":10}}`),
})

// Delete thread
bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:   "memory.deleteThread",
    Payload: json.RawMessage(`{"threadId":"abc-123"}`),
})
```

---

## Usage from .ts

Memory is typically configured on agents directly:

```typescript
import { agent, createMemory, LibSQLStore } from "kit";

const store = new LibSQLStore({ id: "mem" });
const memory = createMemory({ storage: store });

const a = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a helpful assistant.",
    memory: {
        thread: "session-1",
        resource: "user-1",
        storage: store,
    },
});

// Memory is automatic during generate/stream
await a.generate("Hello!");
// Messages saved to thread "session-1" automatically
```

For direct thread management without an agent:

```typescript
import { bus } from "kit";

const { threadId } = await bus.ask("memory.createThread", {
    opts: { title: "Research notes" },
});

await bus.ask("memory.save", {
    threadId,
    messages: [
        { role: "user", content: "Summarize RLHF" },
        { role: "assistant", content: "RLHF is..." },
    ],
});

const recalled = await bus.ask("memory.recall", { threadId });
```

---

## Usage from plugins (via SDK)

```go
// Create thread
sdk.Ask[messages.MemoryCreateThreadResp](client, ctx,
    messages.MemoryCreateThreadMsg{
        Opts: &messages.MemoryCreateThreadOpts{Title: "Plugin chat"},
    },
    func(resp messages.MemoryCreateThreadResp, err error) {
        fmt.Println("thread:", resp.ThreadID)
    },
)

// Save messages
client.Ask(ctx, messages.MemorySaveMsg{
    ThreadID: "abc-123",
    Messages: []messages.MemoryMessage{
        {Role: "user", Content: "What's the status?"},
    },
}, func(msg messages.Message) {})
```

---

## Thread Lifecycle

```
createThread(opts) --> threadId
save(threadId, messages) --> messages stored
recall(threadId, query) --> messages retrieved
deleteThread(threadId) --> thread and messages removed
```

Threads are scoped by resource ID when used with agents. `listThreads` can filter by resource to find all conversations for a given user or entity.

---

## Observational Memory

When using LibSQLStore, PostgresStore, or MongoDBStore, agents can use observational memory (3-tier compression: messages to observations to reflections). See the [storage guide](storage.md) for compatibility details.
