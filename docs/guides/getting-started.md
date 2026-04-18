# Getting Started

brainkit is a Go runtime library that embeds QuickJS (JS/TS) and
Watermill pub/sub into a single platform for AI agent teams. This
guide walks from zero to calling a deployed TypeScript service from
Go, then to running the same thing as a standalone server.

Every snippet below is trimmed from a working example under
[`examples/`](../../examples/) — run the example directly to see the
whole program.

## Install

```bash
go get github.com/brainlet/brainkit@v1.0.0-rc.1
```

Requires Go 1.26+. Node.js 22+ is only needed if you plan to rebuild
the embedded JS bundle (most users never touch it). Podman is only
needed for container-backed tests.

## Library mode: hello-embedded

Build a Kit in-process, deploy a tiny `.ts` handler, call it from
Go. No external services, no containers.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/sdk"
)

func main() {
    kit, err := brainkit.New(brainkit.Config{
        Namespace: "hello-embedded",
        Transport: brainkit.Memory(),
        FSRoot:    ".",
    })
    if err != nil {
        log.Fatalf("new kit: %v", err)
    }
    defer kit.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if _, err := kit.Deploy(ctx, brainkit.PackageInline(
        "greeter", "greeter.ts",
        `bus.on("hello", (msg) => msg.reply({ greeting: "hello, " + msg.payload.name }));`,
    )); err != nil {
        log.Fatalf("deploy: %v", err)
    }

    payload, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](
        kit, ctx,
        sdk.CustomMsg{
            Topic:   "ts.greeter.hello",
            Payload: json.RawMessage(`{"name":"world"}`),
        },
        brainkit.WithCallTimeout(2*time.Second),
    )
    if err != nil {
        log.Fatalf("call: %v", err)
    }

    fmt.Println(string(payload))
}
```

Run it:

```bash
go run ./examples/hello-embedded
# {"greeting":"hello, world"}
```

What each piece does:

- `brainkit.Memory()` is the in-process GoChannel transport — fast,
  synchronous, zero dependencies. Use it in tests and single-process
  demos. For real deployments pick one of the backends in
  [transport-backends.md](transport-backends.md).
- `FSRoot` is the filesystem sandbox for deployed `.ts` code. The
  Kit never reads or writes outside that directory.
- `PackageInline(name, entry, code)` wraps a single source string as
  a deployable package. A deployed `.ts` service bound to
  `ts.<name>.<topic>` — here `ts.greeter.hello`.
- `brainkit.Call[Req, Resp]` is the typed generic call helper. It
  publishes, waits for the reply on a private `replyTo` topic, and
  returns the decoded payload. Use `WithCallTimeout` to bound the
  wait.

See [`examples/hello-embedded/`](../../examples/hello-embedded/).

## Add an AI provider

Set an API key in the environment or pass it explicitly:

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "ai-chat",
    Transport: brainkit.Memory(),
    FSRoot:    ".",
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    },
})
```

Deploy a `.ts` that calls the model, then reach it with
`brainkit.Call`:

```ts
bus.on("ask", async (msg) => {
    const r = await generateText({
        model: model("openai", "gpt-4o-mini"),
        prompt: msg.payload.prompt,
    });
    msg.reply({ text: r.text });
});
```

Twelve providers ship out of the box (`OpenAI`, `Anthropic`,
`Google`, `Mistral`, `Groq`, `DeepSeek`, `XAI`, `Cohere`,
`Perplexity`, `TogetherAI`, `Fireworks`, `Cerebras`) with optional
`WithBaseURL` / `WithHeaders` overrides. See
[ai-and-agents.md](ai-and-agents.md) for agents, tools, memory, and
streaming responses.

See [`examples/ai-chat/`](../../examples/ai-chat/).

## Add storage

Pass named backends in `Storages`; deployed `.ts` code resolves them
by name via `storage("name")`.

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: brainkit.Memory(),
    FSRoot:    "/tmp/my-app",
    Storages: map[string]brainkit.StorageConfig{
        "default": brainkit.SQLiteStorage("/tmp/my-app/data.db"),
    },
})
```

Available backends: `SQLiteStorage`, `PostgresStorage`,
`MongoDBStorage`, `UpstashStorage`, `InMemoryStorage`. Vector
backends (`SQLiteVector`, `PgVectorStore`, `MongoDBVectorStore`) go
in `Vectors`. See [storage-and-memory.md](storage-and-memory.md) and
[vectors-and-rag.md](vectors-and-rag.md).

## Register a typed Go tool

Tools registered in Go are callable from every surface — deployed
`.ts` code, plugins, and direct Go callers — through the bus topic
`tools.call`.

```go
type AddInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

type AddOutput struct {
    Sum int `json:"sum"`
}

err := brainkit.RegisterTool(kit, "math.add", brainkit.TypedTool[AddInput]{
    Description: "Return a + b as a typed sum.",
    Execute: func(_ context.Context, in AddInput) (any, error) {
        return AddOutput{Sum: in.A + in.B}, nil
    },
})
```

Call it directly from Go:

```go
resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
    Name:  "math.add",
    Input: map[string]any{"a": 40, "b": 2},
}, brainkit.WithCallTimeout(2*time.Second))
// resp.Result is json.RawMessage containing {"sum":42}
```

Or from a `.ts` service:

```ts
const sum = await bus.call("tools.call", {
    name: "math.add",
    input: { a: 2, b: 3 },
}, { timeoutMs: 2000 });
```

See [`examples/go-tools/`](../../examples/go-tools/).

## Server mode: hello-server

For long-running processes, the `server` package wraps `Kit` with
YAML config, signal handling, and an HTTP gateway.

```yaml
# brainkit.yaml
namespace: hello-server
fs_root: ./data
transport:
  type: embedded
gateway:
  listen: :8080
```

```go
package main

import (
    "context"
    "flag"
    "log"
    "os/signal"
    "syscall"

    "github.com/brainlet/brainkit/server"
)

func main() {
    cfgPath := flag.String("config", "brainkit.yaml", "path to server config")
    flag.Parse()

    cfg, err := server.LoadConfig(*cfgPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    srv, err := server.New(cfg)
    if err != nil {
        log.Fatalf("build server: %v", err)
    }
    defer srv.Close()

    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    log.Printf("listening on %s", cfg.Gateway.Listen)
    if err := srv.Start(ctx); err != nil {
        log.Fatalf("server start: %v", err)
    }
}
```

The server composes the gateway, tracing, probes, audit, and
optional plugins out of the config file. Run it:

```bash
go run ./examples/hello-server -config ./examples/hello-server/brainkit.yaml
```

See [`examples/hello-server/`](../../examples/hello-server/).

## CLI

The `brainkit` binary has five verbs:

| Verb      | Purpose                                       |
|-----------|-----------------------------------------------|
| `start`   | Run a `server.Server` from a YAML config.     |
| `deploy`  | Push a `.ts` package into a running Kit.      |
| `call`    | Publish a typed message and print the reply.  |
| `inspect` | List deployed packages, tools, subscriptions. |
| `new`     | Scaffold a server project.                    |

Every verb accepts `-config` or reads `brainkit.yaml` from the
current directory.

## Where to go next

| Guide | What it covers |
|---|---|
| [go-sdk.md](go-sdk.md) | `Config`, accessors, typed Call wrappers, `Module` composition, `server` package. |
| [ts-services.md](ts-services.md) | `bus.on`, `msg.reply`, `msg.send`, `kit.register`, mailbox topics, streaming. |
| [ai-and-agents.md](ai-and-agents.md) | Providers, `generateText` / `streamText`, `Agent`, tools, memory. |
| [storage-and-memory.md](storage-and-memory.md) | Storage backends, Mastra Memory, thread / resource IDs. |
| [vectors-and-rag.md](vectors-and-rag.md) | Vector stores, upsert / query, RAG tools. |
| [plugins.md](plugins.md) | Subprocess plugins (own `go.mod`, WebSocket control plane). |
| [mcp-integration.md](mcp-integration.md) | Wire MCP servers as first-class bus tools. |
| [observability.md](observability.md) | `modules/audit`, `modules/tracing`, query wrappers. |
| [hitl-approval.md](hitl-approval.md) | WIP — human-in-the-loop approval via `modules/harness`. |
| [transport-backends.md](transport-backends.md) | Memory, embedded NATS, external NATS, AMQP, Redis. |

## Example index

Every guide links to one or more examples. The full list:

| Example | What it shows |
|---|---|
| [agent-spawner](../../examples/agent-spawner/) | Agent that designs and deploys other agents at runtime. |
| [ai-chat](../../examples/ai-chat/) | Deploy a `.ts` that calls `generateText`. |
| [cross-kit](../../examples/cross-kit/) | Two Kits on shared NATS, routed by peer name. |
| [gateway-routes](../../examples/gateway-routes/) | HTTP gateway on a bare Kit. |
| [go-tools](../../examples/go-tools/) | Typed Go tools invoked from `.ts` and Go. |
| [harness-lite](../../examples/harness-lite/) | WIP harness surface — `Instance` + six frozen events. |
| [hello-embedded](../../examples/hello-embedded/) | Library-mode minimum. |
| [hello-server](../../examples/hello-server/) | Server-mode minimum. |
| [mcp](../../examples/mcp/) | Wire an external MCP server as tools. |
| [multi-kit](../../examples/multi-kit/) | Two Kits in one process, routed by name. |
| [observability](../../examples/observability/) | `audit.query`, `audit.stats`, `trace.list`. |
| [plugin-author](../../examples/plugin-author/) | Minimal subprocess plugin (own `go.mod`). |
| [plugin-host](../../examples/plugin-host/) | Kit that spawns and calls a plugin. |
| [schedules](../../examples/schedules/) | Cron-style scheduled bus messages. |
| [secrets](../../examples/secrets/) | Encrypted secret lifecycle. |
| [storage-vectors](../../examples/storage-vectors/) | Persistent KV + vector search from `.ts`. |
| [streaming](../../examples/streaming/) | Bus `CallStream`, SSE, WebSocket, Webhook. |
| [workflows](../../examples/workflows/) | Three-step declarative pipeline. |
