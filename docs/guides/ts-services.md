# TypeScript Services

A .ts service is a TypeScript file deployed into a SES Compartment that subscribes to bus topics and handles messages. This is the primary pattern for AI agent services, tool providers, and message handlers in brainkit.

## The Service Pattern

```typescript
// Deploy this via sdk.Publish(rt, ctx, messages.KitDeployMsg{Source: "my-service.ts", Code: ...})

// bus.on subscribes to the deployment mailbox: ts.my-service.<topic>
bus.on("greet", async (msg) => {
    const name = msg.payload?.name || "world";
    msg.reply({ greeting: `Hello, ${name}!` });
});
```

When this is deployed as `my-service.ts`:
- `bus.on("greet")` subscribes to `ts.my-service.greet`
- Go code sends messages via `sdk.SendToService(rt, ctx, "my-service.ts", "greet", payload)`
- Other .ts code sends via `bus.sendTo("my-service.ts", "greet", data)`

## Four Modules

Every .ts file has access to four modules via endowments:

```typescript
import { generateText, streamText, generateObject, streamObject, embed, embedMany, z } from "ai";
import { Agent, createTool, createWorkflow, createStep, Memory, InMemoryStore, ... } from "agent";
import { bus, kit, model, embeddingModel, tools, fs, mcp, output, generateWithApproval, ... } from "kit";
import { compile } from "compiler";
```

The `import` statements are stripped during deployment — the symbols come from Compartment endowments, not ES module resolution. But the imports serve as documentation and give TypeScript IDE support via the `.d.ts` files.

## Bus API

### bus.publish — request/response

```typescript
// fixtures/ts/bus/publish-reply/index.ts
const result = bus.publish("test.greet", { name: "brainkit" });
// result.replyTo: unique reply topic
// result.correlationId: for filtering

bus.subscribe(result.replyTo, (msg) => {
    // msg.payload is the response
});
```

### bus.emit — fire-and-forget

```typescript
// fixtures/ts/bus/emit-fire-and-forget/index.ts
bus.emit("events.user-logged-in", { userId: "123" });
// No replyTo, no response expected
```

### bus.on — mailbox pattern (deployed services only)

```typescript
// Subscribes to ts.<source>.<localTopic>
bus.on("ask", (msg) => {
    msg.reply({ answer: "42" });
});
```

`bus.on` only works inside deployed .ts files — it requires a deployment namespace. Calling it outside a deployment throws `"bus.on() can only be used inside a deployed .ts file"`.

### bus.subscribe — absolute topic

```typescript
// Subscribe to any topic (not namespace-scoped)
bus.subscribe("system.events.shutdown", (msg) => {
    console.log("shutdown received");
});
```

### bus.sendTo — service-to-service

```typescript
// fixtures/ts/bus/send-to-service/index.ts
// Equivalent to sdk.SendToService in Go
const result = bus.sendTo("other-service.ts", "process", { data: "hello" });
// Publishes to ts.other-service.process with replyTo
```

### bus.sendToShard — to WASM shards

```typescript
// fixtures/ts/bus/send-to-shard/index.ts
const result = bus.sendToShard("counter-shard", "counter.increment", { amount: 1 });
// Publishes to counter.increment with replyTo
```

## Message Object

Every handler receives a `msg` with:

```typescript
interface BusMessage {
    payload: unknown;           // parsed JSON data
    replyTo: string;            // reply topic (empty for emit'd events)
    correlationId: string;      // for response filtering
    topic: string;              // the topic this arrived on
    callerId: string;           // sender identity

    reply(data: unknown): void;  // final response (done=true)
    send(data: unknown): void;   // intermediate chunk (done=false)
}
```

### msg.reply — final response

```typescript
bus.on("calculate", (msg) => {
    const result = msg.payload.a + msg.payload.b;
    msg.reply({ result }); // done=true in metadata
});
```

### msg.send — streaming chunks

```typescript
// fixtures/ts/bus/streaming-send-reply/index.ts
bus.on("stream", (msg) => {
    msg.send({ chunk: 1 });  // done=false
    msg.send({ chunk: 2 });  // done=false
    msg.send({ chunk: 3 });  // done=false
    msg.reply({ done: true, total: 3 }); // done=true (final)
});
```

The Go side distinguishes chunks from the final response via `msg.Metadata["done"]`.

## Resource Registration

Creating a tool, agent, or workflow in .ts code does NOT automatically register it. You must call `kit.register`:

```typescript
// fixtures/ts/tools/create-register-call/index.ts
const myTool = createTool({
    id: "multiply",
    description: "multiplies two numbers",
    inputSchema: z.object({ a: z.number(), b: z.number() }),
    execute: async ({ context: input }) => ({ result: input.a * input.b }),
});

kit.register("tool", "multiply", myTool);
// NOW it's callable from Go, WASM, plugins, other .ts code
```

```typescript
// fixtures/ts/agent/generate-basic/index.ts
const agent = new Agent({
    name: "my-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are helpful.",
});

kit.register("agent", "my-agent", agent);
// NOW it appears in agents.list
```

Valid types: `"tool"`, `"agent"`, `"workflow"`, `"memory"`. Registration is idempotent — registering the same type+name twice is a no-op.

## AI in Services

The canonical pattern — a .ts service that calls AI and replies via the bus:

```typescript
// Pattern from test/surface/ts_test.go — TestTS_BusServiceAsAIProxy
bus.on("generate", async (msg) => {
    try {
        const result = await generateText({
            model: model("openai", "gpt-4o-mini"),
            prompt: msg.payload.prompt || "say hello",
            maxTokens: 20,
        });
        msg.reply({ text: result.text, usage: result.usage });
    } catch (e) {
        msg.reply({ error: e.message });
    }
});
```

Async handlers work — the `bus.on` handler can `await` any Promise (fetch, generateText, agent.generate, tools.call, setTimeout). The QuickJS job pump processes Schedule'd callbacks every 10ms, enabling full async operation inside bus handlers.

## Agents in Services

```typescript
// fixtures/ts/agent/generate-with-tools/index.ts
const searchTool = createTool({
    id: "search",
    description: "searches the web",
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
});

kit.register("agent", "researcher", agent);

const result = await agent.generate("What is brainkit?");
output({ text: result.text, hasSteps: result.steps.length > 0 });
```

## Agents with Memory

```typescript
// fixtures/ts/agent/with-memory-inmemory/index.ts
const store = new InMemoryStore();
const mem = new Memory({ storage: store });

const agent = new Agent({
    name: "memory-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember the user's name.",
    memory: mem,
});

kit.register("agent", "memory-agent", agent);

// First call — agent learns the name
await agent.generate("My name is David", {
    threadId: "thread-1",
    resourceId: "user-1",
});

// Second call — agent remembers
const result = await agent.generate("What's my name?", {
    threadId: "thread-1",
    resourceId: "user-1",
});
// result.text contains "David"
```

For persistent memory, replace `InMemoryStore` with `LibSQLStore`, `PostgresStore`, or `MongoDBStore`. See [storage-and-memory.md](storage-and-memory.md).

## HITL Approval

`generateWithApproval` routes tool approval through the bus. Any surface (Go, .ts, plugin) can approve or decline.

```typescript
// fixtures/ts/agent/hitl-bus-approval/index.ts
const deleteTool = createTool({
    id: "delete-record",
    description: "Delete a record — requires human approval",
    inputSchema: z.object({ id: z.string() }),
    requireApproval: true,
    execute: async ({ id }) => ({ deleted: true }),
});

const agent = new Agent({
    name: "hitl-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Always use the delete-record tool when asked to delete.",
    tools: { "delete-record": deleteTool },
});

const result = await generateWithApproval(agent, "Delete record xyz-789", {
    approvalTopic: "approvals.pending",
    timeout: 10000,
});
// result.text — the agent's final response after approval
```

The bus lifecycle (publish approval request, subscribe for response, wait with timeout) is handled by a Go bridge function — no JS closures, no setTimeout, no GC risk. See [hitl-approval.md](hitl-approval.md).

## Using Go-Registered Tools from .ts

```typescript
// fixtures/ts/agent/with-registered-tool/index.ts
// "multiply" was registered in Go before this .ts runs
const multiplyTool = tool("multiply");

const agent = new Agent({
    name: "math-bot",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the multiply tool.",
    tools: { multiply: multiplyTool },
});

const result = await agent.generate("What is 6 times 7?", { maxSteps: 3 });
```

`tool("name")` resolves a Go-registered tool into a Mastra-compatible tool object that agents can use. It calls `tools.resolve(name)` to get the metadata and wraps it with `createTool({ execute: async (input) => tools.call(name, input) })`.

## Workflows

```typescript
// fixtures/ts/workflow/then-basic/index.ts
const step1 = createStep({
    id: "uppercase",
    inputSchema: z.object({ text: z.string() }),
    outputSchema: z.object({ upper: z.string() }),
    execute: async ({ inputData }) => ({ upper: inputData.text.toUpperCase() }),
});

const step2 = createStep({
    id: "exclaim",
    inputSchema: z.object({ upper: z.string() }),
    outputSchema: z.object({ result: z.string() }),
    execute: async ({ inputData }) => ({ result: inputData.upper + "!!!" }),
});

const wf = createWorkflow({
    id: "my-workflow",
    inputSchema: z.object({ text: z.string() }),
    outputSchema: z.object({ result: z.string() }),
}).then(step1).then(step2).commit();

const run = await wf.createRun();
const result = await run.start({ inputData: { text: "hello" } });
output({ status: result.status, result: result.result });
// status: "success", result: { result: "HELLO!!!" }
```

## Output

`output(value)` sets the deployment's return value, readable from Go:

```typescript
output({ status: "ok", count: 42 });
```

```go
// Go side — after deploy
result, _ := k.EvalTS(ctx, "__read.ts", `return globalThis.__module_result || "null"`)
// result: '{"status":"ok","count":42}'
```

## Console

`console.log/warn/error/info/debug` are per-source tagged:

```typescript
console.log("starting up");  // [my-service.ts] [log] starting up
console.error("something broke"); // [my-service.ts] [error] something broke
```

Routed through `KernelConfig.LogHandler` if set, otherwise printed via `log.Printf`.
