# AI and Agents

brainkit ships the Vercel AI SDK and the Mastra agent framework
inside the JS runtime, plus a thin Go surface that registers AI
providers on the Kit. Use `generateText` / `streamText` /
`generateObject` / `embed` directly, or compose agents with tools,
memory, workflows, and sub-agents.

## Register providers from Go

Twelve providers ship as `Config.Providers` builders. Either set
them explicitly:

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "ai-chat",
    Transport: brainkit.Memory(),
    FSRoot:    ".",
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
        brainkit.Anthropic(os.Getenv("ANTHROPIC_API_KEY")),
    },
})
```

...or leave `Providers` nil to auto-detect from environment
variables: `OPENAI_API_KEY` → `openai`, `ANTHROPIC_API_KEY` →
`anthropic`, and the equivalents for `google`, `mistral`, `groq`,
`deepseek`, `xai`, `cohere`, `perplexity`, `togetherai`,
`fireworks`, `cerebras`.

Point a builder at a compatible endpoint with options:

```go
brainkit.OpenAI(key,
    brainkit.WithBaseURL("https://my-proxy.example.com/v1"),
    brainkit.WithHeaders(map[string]string{"X-Org": "acme"}),
)
```

Manage providers after boot via `kit.Providers()` —
`Register` / `Unregister` / `List` / `Get` / `Has`.

See [`examples/ai-chat/`](../../examples/ai-chat/) for a complete
single-provider program, and
[`examples/agent-spawner/`](../../examples/agent-spawner/) for a
live two-agent pipeline.

## Resolve models in `.ts`

```ts
model("openai", "gpt-4o-mini")
model("anthropic", "claude-sonnet-4-5")
model("google", "gemini-2.0-flash")
model("groq", "llama-3.3-70b")

embeddingModel("openai", "text-embedding-3-small") // 1536 dims
embeddingModel("openai", "text-embedding-3-large") // 3072 dims
```

Only providers registered on the Kit resolve successfully. Calling
`model("anthropic", ...)` without an `anthropic` provider produces
a model handle that will fail at invocation time.

## Direct AI SDK calls

### generateText

```ts
bus.on("chat", async (msg) => {
    const r = await generateText({
        model: model("openai", "gpt-4o-mini"),
        prompt: msg.payload.prompt,
        maxTokens: 200,
    });
    msg.reply({
        text: r.text,
        usage: r.usage,
        finishReason: r.finishReason,
    });
});
```

`r.usage` normalizes across providers; the AI SDK also surfaces
`inputTokens` / `outputTokens` on some providers. See the
`agent-spawner` example for a defensive mapping.

### streamText

```ts
bus.on("stream", async (msg) => {
    const stream = await streamText({
        model: model("openai", "gpt-4o-mini"),
        prompt: msg.payload.prompt,
    });
    for await (const delta of stream.textStream) msg.send({ delta });
    msg.reply({ done: true });
});
```

Consume from Go with `brainkit.CallStream`, via the gateway as SSE
with `gw.HandleStream(...)`, or over WebSocket with
`gw.HandleWebSocket(...)`. See
[`examples/streaming/`](../../examples/streaming/).

### generateObject

```ts
const r = await generateObject({
    model: model("openai", "gpt-4o-mini"),
    prompt: "Generate a person with name and age.",
    schema: z.object({ name: z.string(), age: z.number() }),
});
msg.reply(r.object);
```

### embed / embedMany

```ts
const e = await embed({
    model: embeddingModel("openai", "text-embedding-3-small"),
    value: "hello world",
});
// e.embedding is number[] (length 1536)
```

```ts
const r = await embedMany({
    model: embeddingModel("openai", "text-embedding-3-small"),
    values: ["hello", "world", "foo"],
});
```

See [vectors-and-rag.md](vectors-and-rag.md) for wiring embeddings
into a vector store.

### Inline tools on generateText

```ts
const r = await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "What is 6 times 7?",
    tools: {
        multiply: {
            description: "Multiply two numbers",
            parameters: z.object({ a: z.number(), b: z.number() }),
            execute: async ({ a, b }) => a * b,
        },
    },
    maxSteps: 3,
});
```

## Agents — Mastra

Mastra exposes two primitives that solve different problems: **`Agent`** is
the executor, **`Mastra`** is the container. Pick the right one based on
whether your flow needs to suspend and resume.

### Simple path — bare `Agent`

For one-shot `.generate()` and `.stream()` calls that never suspend:

```ts
const researcher = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Answer research questions concisely.",
});
kit.register("agent", "researcher", researcher);

const r = await researcher.generate("What is brainkit?", { maxSteps: 3 });
msg.reply({ text: r.text, usage: r.usage });
```

`new Agent(...)` creates the agent; `kit.register("agent", name,
ref)` makes it visible to `agents.list`, `agents.discover`, and the
Mastra tool-registry pipeline. Unregistered agents work locally but
aren't discoverable over the bus.

### Durable path — `Mastra` container

For HITL tool approval, resumable workflows, or sub-agent networks that share
memory, wrap the agent in a `Mastra` instance with a storage backend:

```ts
import { Agent, Mastra, InMemoryStore } from "agent";

const mastra = new Mastra({
    agents:  { assistant: new Agent({ name: "assistant", model: model(...), ... }) },
    storage: new InMemoryStore(),
});
const assistant = mastra.getAgent("assistant");
```

The Mastra instance carries the workflow-snapshot store used by
`approveToolCallGenerate`, `declineToolCallGenerate`, `resumeGenerate`, and
`resumeStream`. Without it, those methods silently return the original
`{finishReason: "suspended"}` shape — the snapshot lookup short-circuits
because `this.#mastra` is undefined.

Use the durable path for:

| Flow                                    | Why Mastra is required                                       |
|-----------------------------------------|--------------------------------------------------------------|
| `generateWithApproval` / HITL           | Resume after approval loads the agent-loop snapshot.         |
| `resumeGenerate` / `resumeStream`       | Snapshot-driven continuation of a paused run.                |
| Workflows with agents as steps          | `agent` step resolves via `mastra.getAgent(name)`.           |
| Sub-agent networks with shared memory   | Shared storage + run context live on the Mastra instance.    |

See [`examples/hitl-tool-approval/`](../../examples/hitl-tool-approval/) and
[`fixtures/ts/agent/hitl/bus-approval/`](../../fixtures/ts/agent/hitl/bus-approval/)
for the end-to-end wiring.

### Tools on agents

```ts
const searchTool = createTool({
    id: "search",
    description: "Search the web",
    inputSchema: z.object({ query: z.string() }),
    execute: async ({ context: { query } }) => {
        return { results: [`Result for: ${query}`] };
    },
});

const agent = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the search tool to answer questions.",
    tools: { search: searchTool },
    maxSteps: 5,
});
```

Use a Go-registered tool from `.ts`:

```ts
const agent = new Agent({
    name: "math-bot",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the math.add tool for arithmetic.",
    tools: { math_add: tool("math.add") },
});
```

See [`examples/go-tools/`](../../examples/go-tools/).

### Streaming

```ts
const stream = await agent.stream("Count to five.");
for await (const chunk of stream.textStream) msg.send({ delta: chunk });
msg.reply({ done: true, usage: await stream.usage });
```

### Agent memory

`Memory` gives the agent persistent state keyed by
`threadId` + `resourceId`.

```ts
const mem = new Memory({ storage: new LibSQLStore({ id: "chat-memory" }) });

const chat = new Agent({
    name: "chatter",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember what the user told you.",
    memory: mem,
});

await chat.generate("My name is David", {
    threadId: "thread-1",
    resourceId: "user-1",
});
const r = await chat.generate("What's my name?", {
    threadId: "thread-1",
    resourceId: "user-1",
});
```

`LibSQLStore({ id })` resolves the Kit's named SQLite storage; swap
for `new InMemoryStore()` during development. Full details in
[storage-and-memory.md](storage-and-memory.md).

### Sub-agents

```ts
const mathAgent = new Agent({
    name: "math",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a math expert.",
});

const supervisor = new Agent({
    name: "supervisor",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Delegate math questions to the math agent.",
    agents: { math: mathAgent },
    maxSteps: 5,
});

const r = await supervisor.generate("What is 123 * 456?");
```

Each sub-agent becomes a tool the supervisor can invoke.

## Agent-spawning agents

Agents can deploy new agents at runtime — the "agent architect"
pattern. The `deploy_agent` tool template generates a `.ts` source
string and calls `bus.call("package.deploy", ...)`.

```ts
const deployAgent = createTool({
    id: "deploy_agent",
    description: "Spawn a brand new agent on this Kit.",
    inputSchema: z.object({
        name: z.string(),
        instructions: z.string(),
    }),
    execute: async ({ context }) => {
        const { name, instructions } = context;
        const src =
            `const a = new Agent({ name: ${JSON.stringify(name)},` +
            `  model: model("openai", "gpt-4o-mini"),` +
            `  instructions: ${JSON.stringify(instructions)} });` +
            `kit.register("agent", ${JSON.stringify(name)}, a);` +
            `bus.on("ask", async (msg) => {` +
            `  const r = await a.generate(msg.payload.prompt);` +
            `  msg.reply({ text: r.text, usage: r.usage });` +
            `});`;
        const resp = await bus.call("package.deploy", {
            manifest: { name, entry: `${name}.ts` },
            files: { [`${name}.ts`]: src },
        }, { timeoutMs: 30000 });
        return { deployed: !!resp.deployed, name: resp.name || name,
                 topic: `ts.${name}.ask` };
    },
});
```

The full working program, including the Go side that calls the
spawned agent, is in
[`examples/agent-spawner/`](../../examples/agent-spawner/).

## Calling agents from Go

Every registered agent responds on `agents.generate` and friends:

```go
resp, err := brainkit.CallAgentList(kit, ctx, sdk.AgentListMsg{},
    brainkit.WithCallTimeout(2*time.Second))
for _, a := range resp.Agents {
    fmt.Println(a.Name, a.Instructions)
}
```

For request/response against a service-hosted agent, call the
service's topic:

```go
reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx,
    sdk.CustomMsg{
        Topic:   "ts.researcher.ask",
        Payload: json.RawMessage(`{"prompt":"what is brainkit?"}`),
    },
    brainkit.WithCallTimeout(60*time.Second))
```

## Voice

Voice is its own guide — see [voice-and-audio.md](voice-and-audio.md).
The headline: `new Audio(stream).play()` routes TTS through a
pluggable `audio.Sink` (desktop speakers, disk, bus fan-out),
`OpenAIRealtimeVoice` works inside the SES compartment via a
client `WebSocket` polyfill, and four runnable examples
(`voice-chat`, `voice-agent`, `voice-broadcast`,
`voice-realtime`) cover the common shapes.

## What's verified

Every snippet above is taken from the `examples/` programs or from
the project's fixture suite.
