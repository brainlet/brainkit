# working-memory

Multi-turn agent that actually remembers what it was told in a
previous call. Mastra's `Memory` class + the
`memory: { thread, resource }` option on `agent.generate` give the
LLM the prior conversation history on the next turn.

## What the example proves

Three turns, in order:

1. **Turn 1** (`thread=t1`, `resource=user-alice`) — *"Hi, my name is Alice. Please remember it."*
2. **Turn 2** (`thread=t1`, `resource=user-alice`) — *"What did I tell you my name was? One word answer."*
3. **Turn 3** (`thread=t2`, `resource=user-alice`) — *"What did I tell you my name was?"*

Expected:

- Turn 2 → `"Alice."` (same thread, recall works)
- Turn 3 → something like `"I don't know your name."` (fresh
  thread, even though the same user owns it)

Turn 3 is the important one — it proves **thread isolation**. The
`resource` groups threads per user/tenant for analytics and
cross-thread semantic recall; it does NOT automatically share the
literal conversation buffer. Only the `thread` id does.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/working-memory
```

## threadId vs resourceId

| | |
|---|---|
| `thread.id` | A single conversation. One back-and-forth. Memory + history are scoped here. |
| `resource` | A user / tenant / agent instance. Multiple threads can share it. Used for analytics, cross-thread recall (`semanticRecall.scope === "resource"`), and memory scoping when `workingMemory.scope === "resource"`. |

For a chat UI: one `resourceId` per logged-in user, one
`thread.id` per conversation window. Archive a thread to keep its
history; start a fresh `thread.id` to begin a new conversation
for the same user.

## Storage

The example uses SQLite under the Kit's `FSRoot`:

```go
Storages: map[string]brainkit.StorageConfig{
    "default": brainkit.SQLiteStorage(filepath.Join(tmp, "memory.db")),
},
```

Swap for any Mastra-supported backend by changing that single
line — the `.ts` code is unchanged:

```go
"default": brainkit.PostgresStorage(os.Getenv("DATABASE_URL"))
"default": brainkit.MongoDBStorage(uri, dbName)
"default": brainkit.UpstashStorage(url, token)
```

Point at `examples/storage-vectors/docker-compose.yml` for a
Postgres that's running and ready for pgvector + memory.

## The deployed `.ts`, annotated

```ts
const memory = new Memory({ storage: storage("default") });

const agent = new Agent({
    name: "memory-demo",
    model: model("openai", "gpt-4o-mini"),
    instructions: "...",
    memory,                       // <-- binds Memory to the agent
});

bus.on("ask", async (msg) => {
    const result = await agent.generate(msg.payload.prompt, {
        memory: {
            thread: { id: msg.payload.threadId },
            resource: msg.payload.resourceId,
        },
    });
    msg.reply({ text: result.text });
});
```

The `thread` can be a bare string (`thread: "t1"`) or an object
(`thread: { id: "t1", title: "...", metadata: {...} }`). The
object form is canonical per Mastra's docs and all five
brainkit memory fixtures (`fixtures/ts/agent/memory/*`).

## Extension ideas

- **Working memory** (a markdown scratchpad the model updates
  across turns): pass `options: { workingMemory: { enabled: true } }`
  to `new Memory`. `updateWorkingMemory` lets an external caller
  rewrite the scratchpad directly.
- **Semantic recall** across threads: pass a vector store into
  `new Memory({ storage, vector, embedder, options: { semanticRecall: { scope: "resource", topK: 5 } } })`.
  The agent pulls in the top-K similar messages from ALL of the
  resource's threads when answering.
- **Conversation TTL**: prune old threads on a schedule via
  `modules/schedules` + `memory.deleteThread({threadId})`.
- **Per-user personas**: store user preferences in the thread's
  `metadata` when you create the thread.

## Under the hood

- The `storage()` resolver inside `.ts` looks up entries in
  `brainkit.Config.Storages`. `storage("default")` returns the
  SQLite instance the Kit was configured with.
- Mastra's `Memory` maintains a conversation table keyed on
  `(resourceId, threadId)`. Each `agent.generate` call with
  `memory: {...}` appends the new user/assistant turn and
  slices the last N messages back into the LLM's context
  window on the next call.
- brainkit exposes five memory backends as Compartment
  endowments (`InMemoryStore`, `LibSQLStore`, `PostgresStore`,
  `MongoDBStore`, `UpstashStore`), all usable via the same
  `new Memory({ storage: … })` shape.

## See also

- `examples/storage-vectors/` — raw `memory.saveThread` /
  `getThreadById` + a vector store.
- `docs/guides/storage-and-memory.md` — the full memory
  guide.
- `fixtures/ts/agent/memory/*` — reference round-trip tests for
  every memory backend brainkit exposes.
