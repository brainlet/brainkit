# TypeScript Services

A "service" in brainkit is a `.ts` file deployed into a SES
Compartment that subscribes to bus topics and handles incoming
messages. Services host agents, tools, workflows, and ad-hoc
handlers — anything you'd otherwise write as a long-running
process, but expressed as a bundle of message handlers.

## The mailbox namespace

Every deployment has a stable namespace:

- Deploy with name `"my-service"` → handlers land on
  `ts.my-service.<topic>`.
- `bus.on("ask")` inside that deployment subscribes to
  `ts.my-service.ask`.
- Go callers reach it with
  `brainkit.Call[sdk.CustomMsg, Resp](..., sdk.CustomMsg{Topic: "ts.my-service.ask", ...})`
  or `sdk.SendToService(kit, ctx, "my-service", "ask", payload)`.
- Other `.ts` code reaches it with
  `bus.sendTo("my-service", "ask", data)` for fire-and-forget or
  `bus.callService("my-service", "ask", data, { timeoutMs })` for
  request/response. Use `bus.callServiceStream(...)` when the
  callee sends chunks before the terminal reply.

The deployment name is the first argument to `packages.Inline`, the
`name` field in a package `manifest.yaml`, or the directory basename
for `packages.FromDir`.

## Endowments

`.ts` code runs inside a SES Compartment with a fixed set of
endowments on `globalThis`:

```typescript
import { generateText, streamText, generateObject, streamObject,
         embed, embedMany, z } from "ai";
import { Agent, createTool, createWorkflow, createStep,
         Memory, InMemoryStore, LibSQLStore } from "agent";
import { bus, kit, model, embeddingModel,
         storage, vectorStore, tools, fs, mcp, output } from "kit";
```

The `import` lines are stripped during deployment — the symbols are
already on `globalThis`. Keep them in source for type-checking and
IDE autocomplete via the shipped `.d.ts` files.

## Bus API

Symmetric with the Go surface:

| Go | TypeScript |
|---|---|
| `sdk.Emit` / event publish | `bus.publish(topic, payload)` or `bus.emit(topic, payload)` |
| `sdk.Emit` | `bus.emit(topic, payload)` |
| `sdk.SubscribeTo` | `bus.subscribe(topic, handler)` |
| `brainkit.Call` | `bus.call(topic, payload, { timeoutMs })` |
| `brainkit.Call` to a service topic | `bus.callService(service, topic, payload, { timeoutMs })` |
| `sdk.SendToService` / event send | `bus.sendTo(service, topic, payload)` |
| `brainkit.CallStream` | `bus.callStream(topic, payload, { timeoutMs, onChunk })` |
| `brainkit.CallStream` to a service topic | `bus.callServiceStream(service, topic, payload, { timeoutMs, onChunk })` |
| `WithCallTo("peer")` | `bus.callTo("peer", topic, payload, { timeoutMs })` / `bus.callToStream(...)` |
| `packages.Deploy` handler | `bus.on(topic, handler)` |

### bus.on — subscribe to the mailbox

```typescript
bus.on("greet", (msg) => {
    const name = msg.payload?.name || "world";
    msg.reply({ greeting: `hello, ${name}` });
});
```

`bus.on` is only valid inside a deployed `.ts` file — it needs the
deployment name to scope the subscription. Calling it outside a
deployment throws.

Handlers may be `async`; the runtime's job pump drives promises and
scheduled callbacks.

### bus.publish / bus.subscribe — arbitrary topics

```typescript
const sub = bus.subscribe("inventory.update", (msg) => {
    // event payload
});

bus.publish("inventory.update", { sku: "X", qty: 5 });
bus.unsubscribe(sub);
```

`bus.publish` does not attach a reply inbox. Use `bus.call` or
`bus.callService` when a response is part of the contract.

### bus.emit — fire-and-forget

```typescript
bus.emit("events.signup", { userId: "123" });
```

### bus.call — await a reply

```typescript
const resp = await bus.call("tools.call", {
    name: "math.add",
    input: { a: 2, b: 3 },
}, { timeoutMs: 2000 });
```

`bus.call` is the `.ts` twin of `brainkit.Call`. It uses the Kit's
shared caller inbox and resolves with the decoded payload.

### bus.callStream — stream chunks and await the final reply

```typescript
const chunks: any[] = [];
const final = await bus.callStream("ts.counter.count", { n: 3 }, {
    timeoutMs: 5000,
    onChunk: async (chunk, streamMsg) => {
        chunks.push(chunk);
        // streamMsg carries topic/correlationId metadata for tracing.
    },
    bufferSize: 16,
    bufferPolicy: "block",
});
```

`callStream` uses the same shared caller inbox as `call`. Intermediate
`msg.send(...)` or `msg.stream.*(...)` replies are delivered to
`onChunk`, and the returned promise resolves only after the terminal
`msg.reply(...)` is received and queued chunks have drained. `onChunk`
may be async; chunks are processed one at a time. `bufferPolicy` is one
of `"block"`, `"dropNewest"`, `"dropOldest"`, or `"error"`.

### bus.sendTo / bus.callService / bus.callServiceStream / bus.callTo — service addressing

```typescript
// Fire-and-forget send to another service's mailbox.
bus.sendTo("logger", "write", { line: "ok" });

// Request/reply against a local service mailbox.
const local = await bus.callService("logger", "status", {}, { timeoutMs: 2000 });

// Streaming request/reply against a local service mailbox.
const streamed = await bus.callServiceStream("logger", "tail", {}, {
    timeoutMs: 5000,
    onChunk: (chunk) => console.log(chunk),
});

// Request/response against a named peer via topology.
const r = await bus.callTo("analytics", "summary.get", { day: "2024-11-14" }, { timeoutMs: 2000 });
```

## The message object

Every handler receives:

```typescript
interface BusMessage {
    payload: unknown;        // parsed JSON body
    topic: string;
    replyTo: string;         // "" for emitted events
    correlationId: string;
    callerId: string;

    reply(data: unknown): void;  // terminal — sets done=true
    send(data:  unknown): void;  // intermediate chunk — done=false
}
```

### msg.reply — terminal response

```typescript
bus.on("add", (msg) => {
    const { a, b } = msg.payload;
    msg.reply({ sum: a + b });
});
```

### msg.send — streaming chunks

```typescript
bus.on("count", (msg) => {
    const n = msg.payload.n || 3;
    for (let i = 1; i <= n; i++) msg.send({ tick: i });
    msg.reply({ done: true, total: n });
});
```

The Go side distinguishes chunks from the terminal reply via the
`done` flag in metadata. Use `brainkit.CallStream` from Go or
`bus.callStream` / `bus.callServiceStream` from TypeScript to consume
them.

See [`examples/streaming/`](../../examples/streaming/).

## Registering resources

`kit.register(type, name, ref)` is the only way to make a resource
visible outside the deployment. Valid types: `"tool"`, `"agent"`,
`"workflow"`, `"memory"`.

```typescript
const myTool = createTool({
    id: "multiply",
    description: "Multiplies two numbers",
    inputSchema: z.object({ a: z.number(), b: z.number() }),
    execute: async ({ context: { a, b } }) => ({ result: a * b }),
});
kit.register("tool", "multiply", myTool);
```

```typescript
const agent = new Agent({
    name: "haiku-bot",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Write one three-line haiku per prompt.",
});
kit.register("agent", "haiku-bot", agent);
```

Registration is idempotent — registering the same `type+name`
twice is a no-op. When the deployment is torn down, every resource
it registered is removed automatically.

## Calling Go-registered tools

Go-registered tools are first-class bus citizens. Call them with
`bus.call("tools.call", ...)`:

```typescript
const sum = await bus.call("tools.call", {
    name: "math.add",
    input: { a: 2, b: 3 },
}, { timeoutMs: 2000 });
// sum.result is the tool's output
```

Or wrap one as a Mastra-compatible agent tool with `tool(name)`:

```typescript
const agent = new Agent({
    name: "math-bot",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the math.add tool for arithmetic.",
    tools: { multiply: tool("math.add") },
});
```

See [`examples/go-tools/`](../../examples/go-tools/).

## Pattern: service with an agent

```typescript
const researcher = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Answer research questions concisely.",
});
kit.register("agent", "researcher", researcher);

bus.on("ask", async (msg) => {
    try {
        const r = await researcher.generate(msg.payload.prompt, { maxSteps: 3 });
        msg.reply({ text: r.text, usage: r.usage });
    } catch (e) {
        msg.reply({ error: String(e?.message || e) });
    }
});
```

See [`examples/ai-chat/`](../../examples/ai-chat/) and
[`examples/agent-spawner/`](../../examples/agent-spawner/).

## Pattern: streaming reply

```typescript
bus.on("stream", async (msg) => {
    const stream = await streamText({
        model: model("openai", "gpt-4o-mini"),
        prompt: msg.payload.prompt,
    });
    for await (const delta of stream.textStream) {
        msg.send({ delta });
    }
    msg.reply({ done: true });
});
```

Consume with `brainkit.CallStream`, `gateway.HandleStream` (SSE),
or `gateway.HandleWebSocket`. See
[`examples/streaming/`](../../examples/streaming/).

## Pattern: cancellation

Long-running handlers should watch for cancellation — the
`brainkit.Call` helper publishes a `_brainkit.cancel` control
message when its `ctx` is cancelled before a terminal reply.

```typescript
bus.on("slow", async (msg) => {
    for (let i = 0; i < 100; i++) {
        if (msg.cancelled) break;
        msg.send({ tick: i });
        await new Promise((r) => setTimeout(r, 100));
    }
    msg.reply({ done: true });
});
```

Pass `WithCallNoCancelSignal()` on the Go side if a peer opts out of
this signal.

## Workflows

`createStep` + `createWorkflow` wire typed steps into a pipeline.

```typescript
const step1 = createStep({
    id: "uppercase",
    inputSchema:  z.object({ text: z.string() }),
    outputSchema: z.object({ upper: z.string() }),
    execute: async ({ inputData }) => ({ upper: inputData.text.toUpperCase() }),
});

const wf = createWorkflow({
    id: "shout",
    inputSchema:  z.object({ text: z.string() }),
    outputSchema: z.object({ upper: z.string() }),
}).then(step1).commit();

kit.register("workflow", "shout", wf);
```

Run from Go with `workflowmsg.CallWorkflowStart`. See
[`examples/workflows/`](../../examples/workflows/).

## output

`output(value)` stores a final return value on the deployment,
readable from Go.

```typescript
output({ ready: true, version: "1.0.0" });
```

## console

`console.log / warn / error / info / debug` are tagged with the
source file and routed through `Config.LogHandler` when set,
otherwise printed via the runtime logger.

```typescript
console.log("starting up");
// [my-service.ts] [log] starting up
```

## What's intentionally absent

- No blocking bus helpers, no `PublishAwait`, no hidden synchronous
  waits. Every round trip is an explicit publish / subscribe /
  wait.
- No ES module resolution. The `import` lines are stripped; all
  symbols come from endowments.
- No module-level side effects at deploy time beyond creating the
  resources and calling `kit.register` / `bus.on`. The deployment
  returns once the top-level code finishes; handlers then run as
  messages arrive.
