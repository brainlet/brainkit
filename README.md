# brainkit

**Deploy AI agents as `.ts` files at runtime. No restart. No schema migration. No service to babysit.**

brainkit is a Go library that embeds a hardened JS/TS runtime (QuickJS + SES
Compartments) and a typed pub/sub bus (Watermill) behind a single `Kit` type.
You write agents as TypeScript, hand them to a running Kit, and they execute
inside an isolated Compartment with first-class access to the [Mastra][mastra]
framework, the AI SDK, and every Go tool you register.

One binary. Zero external services in library mode. Same code path scales from
embedded-in-a-CLI to a multi-kit cluster over NATS.

[mastra]: https://mastra.ai

## 30-second taste

Embed a Kit, deploy an agent, call a tool:

```go
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "myapp",
    Transport: brainkit.EmbeddedNATS(),
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    },
})
defer kit.Close()

// Register a Go tool — the agent sees it as a typed Mastra tool.
brainkit.RegisterTool(kit, "add", tools.TypedTool[struct{ A, B int }]{
    Description: "adds two numbers",
    Execute: func(_ context.Context, in struct{ A, B int }) (any, error) {
        return map[string]int{"sum": in.A + in.B}, nil
    },
})

// Deploy an agent. No restart, no build step. bus.on("ask", ...) listens
// on `ts.math.ask` — deployment-namespaced automatically.
kit.Deploy(ctx, brainkit.PackageInline("math", "math.ts", `
    import { Agent } from "agent";
    import { model, tool, bus } from "kit";
    bus.on("ask", async (msg) => {
        const agent = new Agent({
            name: "math",
            model: model("openai", "gpt-4o-mini"),
            tools: { add: tool("add") },
            instructions: "Use the add tool. Return only the number.",
        });
        const r = await agent.generate(msg.payload.q);
        msg.reply({ answer: r.text });
    });
`))

// Call the agent from Go. CallKitSend is the typed request/reply helper
// for sending to a deployed .ts topic.
resp, err := brainkit.CallKitSend(kit, ctx, sdk.KitSendMsg{
    Topic:   "ts.math.ask",
    Payload: json.RawMessage(`{"q":"what is 6 + 7?"}`),
}, brainkit.WithCallTimeout(30*time.Second))
if err != nil {
    log.Fatal(err)
}
fmt.Println(string(resp.Payload)) // {"answer":"13"}
```

That's it — hot-reload an agent, reach a Go tool from `.ts`, reply on the
bus, call from Go with a typed request/reply.

## What's in the box

Every deployed `.ts` gets the full Mastra surface without npm installs:

| Surface                | Included                                                                                                                                                  |
|------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Agent**              | `Agent` — executor. `new Agent({...}).generate() / .stream()`. Sub-agent networks via `agents` config. Simple one-shot flows.                              |
| **Mastra**             | `Mastra` — container that binds agents + storage + workflow registry. Required for HITL `approveToolCallGenerate`, resumable workflows, shared sub-agent memory. Bare `Agent` silently returns `{finishReason: "suspended"}` on resume paths without it. |
| **AI SDK**             | `generateText`, `streamText`, `generateObject`, `streamObject`, `embed`, `embedMany` across 12 providers (OpenAI, Anthropic, Google, Mistral, xAI, …)      |
| **Memory**             | `Memory` + `InMemoryStore`, `LibSQLStore`, `PostgresStore`, `MongoDBStore`, `UpstashStore` + working memory, semantic recall, generate-title              |
| **Vector**             | `LibSQLVector`, `PgVector`, `MongoDBVector`, `PineconeVector`, `ChromaVector`, `QdrantVector`                                                              |
| **RAG**                | `MDocument.fromText / fromHTML / fromJSON / fromMarkdown / fromCSV / fromDocx / fromPDF`, `GraphRAG`, `rerank`, `createVectorQueryTool`                   |
| **Processors**         | `ModerationProcessor`, `PIIDetector`, `PromptInjectionDetector`, `SystemPromptScrubber`, `StructuredOutputProcessor`, `TokenLimiterProcessor`, …           |
| **Scorers**            | 16 prebuilt scorers: `createAnswerRelevancyScorer`, `createFaithfulnessScorer`, `createHallucinationScorer`, `createBiasScorer`, `createToxicityScorer`, … |
| **Voice**              | `OpenAIVoice`, `OpenAIRealtimeVoice`, `AzureVoice`, `ElevenLabsVoice`, `CloudflareVoice`, `DeepgramVoice`, `PlayAIVoice`, `SpeechifyVoice`, `SarvamVoice`, `MurfVoice` |
| **Observability**      | `Observability`, `DefaultExporter`, `SensitiveDataFilter`, full OpenTelemetry `sdk-trace-base`: `BatchSpanProcessor`, `SimpleSpanProcessor`, `BasicTracerProvider`, samplers, exporters |
| **Workspace + tools**  | `Workspace`, `LocalFilesystem`, `LocalSandbox`, `createTool` with Zod schemas, Go-registered tools via `kit.tool(name)`                                    |
| **Bus**                | `bus.publish / subscribe / on / sendTo / call / schedule / unschedule`, streaming via `msg.send` + `msg.stream.*`                                         |
| **Polyfills**          | `fetch`, `WebSocket`, `FormData`, `ReadableStream`, `structuredClone` (real deep-clone), `crypto.subtle`, Node `stream` / `net` / `os` / `zlib`           |

86 primitives on `agent`, 14 on `kit` — every one reachable inside a deployed
`.ts` as if you'd npm-installed it.

## Testing

Real tests only. No mocks.

- **272 fixture tests** across 21 categories — every fixture is an `index.ts`
  deployed into a fresh Kit, executed against the real Mastra runtime.
- Real **OpenAI** for AI-backed paths (`OPENAI_API_KEY` gates them). Real
  **Postgres**, **MongoDB**, **libsql-server** via [testcontainers-go][tc] —
  `pgvector`, `PostgresStore`, `MongoDBStore`, `LibSQLStore`, `LibSQLVector`
  all run against the actual databases.
- Real **httptest + coder/websocket** for streaming suites.
- Full OTel span pipeline: `BasicTracerProvider → BatchSpanProcessor →
  InMemorySpanExporter` asserted end-to-end, not stubbed.

Run the lot (OpenAI key + Podman needed):

```sh
go test ./test/fixtures/ -count=1 -timeout=60m
```

[tc]: https://golang.testcontainers.org/

## Examples

Curated — `examples/` has 34 total.

| Example                        | Shows                                                                                |
|--------------------------------|--------------------------------------------------------------------------------------|
| `hello-embedded` / `hello-server` | Minimum viable Kit in library and service mode.                                   |
| `agent-forge`                  | Architect-style agent that writes new `.ts` agents and deploys them at runtime.      |
| `rag-pipeline`                 | Chunk → embed → upsert → query against LibSQLVector with streaming.                  |
| `voice-realtime`               | Bidirectional browser ↔ `OpenAIRealtimeVoice` WebSocket with live PCM16 playback.    |
| `hitl-tool-approval`           | Tool calls routed through the bus for human approval before execution.               |
| `multi-kit`                    | Two Kits on one NATS transport routing via `modules/topology`.                       |
| `plugin-host` / `plugin-author` | Subprocess plugin supervisor and the other half of the contract.                    |
| `schedules`                    | Persisted cron-style scheduling via `modules/schedules`.                             |
| `observability`                | `BatchSpanProcessor` + `InMemorySpanExporter` end-to-end trace capture.              |

## Service mode

When library mode isn't enough — long-running agent backend, gateway, plugins:

```go
import "github.com/brainlet/brainkit/server"

srv, _ := server.QuickStart("my-app", "/var/brainkit",
    server.WithSecretKey(os.Getenv("SECRET_KEY")))
defer srv.Close()
_ = srv.Start(ctx)
```

Or scaffold a server binary via the CLI:

```sh
brainkit new server my-service
cd my-service && go mod tidy
go run . --config brainkit.yaml
```

## Modules

All opt-in. Enable by passing to `Config.Modules`; nothing runs you didn't ask
for.

| Module              | Maturity | What it adds                                         |
|---------------------|----------|------------------------------------------------------|
| `modules/gateway`   | stable   | HTTP gateway: routes, SSE, WebSocket, static FS      |
| `modules/mcp`       | stable   | MCP client — external MCP servers become tools       |
| `modules/plugins`   | beta     | Subprocess plugin supervisor + WebSocket control plane |
| `modules/schedules` | beta     | Persisted cron-style scheduling                      |
| `modules/audit`     | beta     | Audit log query surface + SQLite / Postgres stores   |
| `modules/tracing`   | beta     | Distributed tracing with SQLite backing              |
| `modules/probes`    | beta     | Provider health probes                               |
| `modules/discovery` | beta     | Bus-mode peer discovery                              |
| `modules/topology`  | beta     | Cross-kit routing + `peers.*` bus commands           |
| `modules/workflow`  | beta     | Declarative agent workflow DSL                       |
| `modules/harness`   | **wip**  | Higher-level agent orchestration layer               |

## Architecture shape

```
    ┌──────────────────────── your Go binary ───────────────────────┐
    │                                                                │
    │  brainkit.Kit ──► kernel ──► QuickJS context ──► SES lockdown  │
    │      │                           │                             │
    │      │                           ▼                             │
    │      │              ┌───── Compartment ─────┐                  │
    │      │              │  deployed .ts file    │                  │
    │      │              │  ├── Agent / Mastra   │                  │
    │      │              │  ├── AI SDK           │                  │
    │      │              │  ├── RAG / Memory     │                  │
    │      │              │  └── endowments       │◄───┐             │
    │      │              └───────────────────────┘    │ bus.*        │
    │      │                           ▲                │ tool.*       │
    │      ▼                           │                │ fetch        │
    │  Watermill bus ◄─────────────────┼────────────────┘             │
    │      │                           │                              │
    │      ▼                           ▼                              │
    │  Go handlers · tools · plugins · MCP · modules                  │
    │                                                                │
    └────────────────────────────────────────────────────────────────┘
```

- **Compartment isolation** — each deployment runs in its own SES Compartment
  with a hand-picked endowment set. No ambient authority, no `require("fs")`.
- **Bus-first** — every subsystem talks over the bus. Plugins and agents get
  the same API shape; swap a Go handler for a `.ts` handler and callers don't
  notice.
- **Pluggable transport** — memory (dev), embedded NATS, external NATS,
  RabbitMQ, Redis Streams. Same bus API across all five.

## Is this for me?

| Use case                                     | Shape                                          |
|----------------------------------------------|------------------------------------------------|
| Add AI agents to a Go service                | Library mode, one Kit                          |
| Long-running agent backend                   | Service mode, `server.New` or `brainkit start` |
| Multi-kit routing (analytics + ingest + …)   | Kits on a shared bus + `modules/topology`      |
| Write a plugin in Go                         | Subprocess plugin via `sdk/plugin`              |
| Embed in tests                               | `brainkit.Memory()` transport, no persistence  |

## Design docs

Architecture, migration plans, and vision live in
[`../brainkit-maps/brainkit/designs/`](../brainkit-maps/brainkit/designs/).

Empirical findings (SES patterns, storage resilience, benchmark baselines) —
[`../brainkit-maps/knowledge/`](../brainkit-maps/knowledge/).

## License

Apache 2.0.
