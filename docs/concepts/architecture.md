# Architecture

brainkit 1.0 is a single Go package that boots an in-process AI runtime.
One call — `brainkit.New(Config)` — returns a `*Kit` that owns a QuickJS
sandbox, a Watermill message bus, a tool/provider/storage registry, and
any modules you plug in. Everything runs in one OS process by default;
transports (NATS, Redis, AMQP) are opt-in knobs on the same type.

## The Kit

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "hello",
    Transport: brainkit.Memory(),
    FSRoot:    ".",
})
defer kit.Close()
```

That Kit is the runtime. It is the only top-level object. A Kit:

- Hosts **one QuickJS runtime** with SES locked down. All deployed `.ts`
  services and every agent, tool, workflow, and MCP client share the
  same JS heap. Isolation is per-Compartment, not per-OS-process.
- Owns **one Watermill router** (the bus). Every subsystem — Go, JS,
  plugins, gateway handlers — speaks to every other subsystem by
  publishing messages. There is no separate RPC layer.
- Exposes **typed Go accessors** for providers, storages, vectors, and
  secrets (`kit.Providers()`, `kit.Storages()`, `kit.Vectors()`,
  `kit.Secrets()`).
- Loads **zero or more Modules** (gateway, audit, tracing, probes,
  topology, discovery, plugins, MCP, schedules, workflow, harness)
  which wire themselves into the bus during `Init`.
- Accepts **deployments** — `.ts` packages that register handlers via
  `bus.on` and become addressable at `ts.<pkg>.<topic>`.

See `examples/hello-embedded/main.go` for the minimum viable Kit.

## Narrow Public Surface

The top-level `brainkit` package is intentionally small. The shipped API
is:

- **Constructor.** `New(Config) (*Kit, error)`.
- **Config builders.** `Memory()`, `EmbeddedNATS()`, `NATS(url)`,
  `AMQP(url)`, `Redis(url)`; `OpenAI(key)`, `Anthropic(key)`, … for the
  12 supported providers.
- **Deployment helpers.** `PackageInline(name, entry, source)`,
  `PackageFromDir(dir)`, `PackageFromFile(path)`, then
  `kit.Deploy(ctx, pkg)`.
- **Bus calls.** The generic
  `Call[Req, Resp any](kit, ctx, req, opts…) (Resp, error)` plus
  `CallStream[Req, Chunk, Resp any]` for servers that emit chunks
  before a terminal reply.
- **62 generated wrappers** in `call_gen.go` that bind `Call` to every
  typed bus topic shipped out of the box (`CallPackageDeploy`,
  `CallKitHealth`, `CallAgentDiscover`, `CallAuditQuery`,
  `CallPeersResolve`, …). Regenerate them with `make generate` after
  adding new typed message types.
- **Typed tool registration.** `RegisterTool(kit, name, TypedTool[T])`
  registers a typed Go function as a first-class tool. See
  `examples/go-tools/main.go`.
- **Lifecycle.** `kit.Shutdown(ctx)` drains gracefully, `kit.Close()`
  is the quick equivalent.

Everything else — envelopes, error codes, cross-namespace helpers,
message types — lives in `github.com/brainlet/brainkit/sdk` and is
consumed by both module authors and plugin authors.

## Modules

Modules are the extension point. A module implements three methods:

```go
type Module interface {
    Name() string
    Init(k *Kit) error
    Close() error
}
```

Optionally a module can implement `StatusReporter` to declare itself
`ModuleStatusStable`, `ModuleStatusBeta`, or `ModuleStatusWIP`. The
status surfaces through `kit.Status()` so a caller can refuse to boot
when a WIP module is loaded in production.

`Init` runs in registration order before the router starts, so a module
can freely subscribe to bus topics, register commands
(`RegisterCommand`), or configure its own transport. The 11 modules
that ship in 1.0-rc.1 are:

| Module     | Status | Purpose                                            |
| ---------- | ------ | -------------------------------------------------- |
| audit      | stable | Append-only log of bus traffic and tool calls.     |
| discovery  | stable | Static/bus-based peer discovery (no multicast).    |
| gateway    | stable | HTTP/SSE/WS/Webhook edge on top of typed topics.   |
| harness    | WIP    | Streaming agent harness (only `Instance` frozen).  |
| mcp        | stable | MCP stdio/HTTP servers as tool sources.            |
| plugins    | stable | Subprocess plugins over a WebSocket control plane. |
| probes     | stable | `/healthz`, `/readyz`, periodic probe scheduler.   |
| schedules  | stable | Cron + one-shot bus publishes.                     |
| topology   | stable | Named peer table used by `WithCallTo`.             |
| tracing    | stable | OpenTelemetry span export.                         |
| workflow   | stable | Mastra workflow start/cancel/status wrappers.      |

Loading a module is declarative:

```go
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "edge",
    Transport: brainkit.EmbeddedNATS(),
    Modules: []brainkit.Module{
        gateway.New(gateway.Config{Listen: ":8080"}),
        topology.NewModule(topology.Config{
            Peers: []topology.Peer{{Name: "analytics", Namespace: "analytics-prod"}},
        }),
    },
})
```

Modules never reach into each other. They compose through the bus.

## Transports

`Config.Transport` is a struct value, not a string:

```go
Transport: brainkit.NATS("nats://localhost:4222"),
```

The five constructors return typed `TransportConfig` values:

| Constructor       | Kind         | Use case                            |
| ----------------- | ------------ | ----------------------------------- |
| `Memory()`        | `"memory"`   | Single-process, fastest path.       |
| `EmbeddedNATS()`  | `"embedded"` | Single-process, JetStream semantics. |
| `NATS(url)`       | `"nats"`     | Multi-Kit production, JetStream.    |
| `AMQP(url)`       | `"amqp"`     | RabbitMQ, topic sanitizer.          |
| `Redis(url)`      | `"redis"`    | Redis Streams.                      |

Leaving `Transport` zero defaults to `Memory()` inside `New`. The
kernel and router are wired in the same place regardless of kind;
changing transport changes only where bytes travel. See
[bus-and-messaging.md](bus-and-messaging.md) for the topic rules each
backend applies.

## Deployment Pipeline

A deployment is a `.ts` (or `.js`) package plus a manifest:

```go
kit.Deploy(ctx, brainkit.PackageInline(
    "greeter", "greeter.ts",
    `bus.on("hello", (msg) => msg.reply({ greeting: "hi " + msg.payload.name }));`,
))
```

Under the hood, `Deploy` publishes a `sdk.PackageDeployMsg` on the
`package.deploy` topic. The handler runs the package entry through the
vendored microsoft/typescript-go transpiler, loads it into a fresh SES
Compartment, and exposes every `bus.on(topic, …)` at
`ts.<pkg>.<topic>`. Any subsequent Kit call to that topic enters the
Compartment, runs the JS handler, and replies through the bus. See
[deployment-pipeline.md](deployment-pipeline.md).

## Providers, Storages, Vectors, Secrets

Four typed registries hang off the Kit:

```go
kit.Providers().Register("openai", "openai", ProviderConfig{APIKey: "..."})
kit.Storages().Register("main", "libsql", StorageConfig{URL: "file:./kit.db"})
kit.Vectors().Register("qdrant", "qdrant", VectorConfig{URL: "..."})
```

Each registry owns its own table of named backends. Deployed `.ts` code
sees the same table through `globalThis.__kit_providers` and calls it
through Mastra (`model("openai", "gpt-4o")`) or through
`kit.register(type, name, ref)` for tools/agents/workflows/memories.
See [provider-registry.md](provider-registry.md).

## CLI

`brainkit` the binary is a thin shell around the library. It exposes
five verbs:

- `brainkit start` — boots `server.New(cfg)` from `brainkit.yaml`.
- `brainkit deploy <file|dir>` — wraps a `.ts` tree as a
  `PackageDeployMsg` and POSTs it to a running Kit's gateway.
- `brainkit call <topic> --payload '{…}'` — POSTs a typed bus call to
  `/api/bus` (or `/api/stream` for chunked replies).
- `brainkit inspect <kit>` — prints deployments, modules, providers.
- `brainkit new <name>` — scaffolds a `.ts` package.

The CLI never embeds a Kit of its own. It talks to running Kits via
the gateway module's HTTP surface. For embedded use, import the Go
library directly.

## Server Package

`brainkit/server` packages the common production layout:

```go
srv, _ := server.QuickStart("edge", "./workspace")
_ = srv.Start(ctx)
```

`QuickStart` wires `EmbeddedNATS`, a SQLite storage, and a gateway on
`:8080`; `server.New(cfg)` accepts the full `server.Config` (transport,
providers, plugins, audit, tracing, probes, packages, extra modules)
for real deployments. The server rejects `Memory()` transport — use
the library directly if you want an in-process Kit.

## Plugins

Plugins are separate Go binaries with their own `go.mod`. They link
`github.com/brainlet/brainkit/sdk/plugin`, declare tools and bus
subscriptions through `bkplugin.New(...).Tool(...).On(...)`, and ship
as `./bin/myplugin`. The host Kit's `plugins` module launches each
plugin as a subprocess, connects to it over a WebSocket control plane,
and brokers its tool calls / bus messages through the host bus. The
host Kit must run on a non-memory transport because plugins connect
into the transport by URL. See `examples/plugin-author/main.go` and
`modules/plugins/README.md`.

## Ports of Call

- Minimal Kit: `examples/hello-embedded/main.go`.
- Typed Go tools: `examples/go-tools/main.go`.
- Agent that spawns other agents: `examples/agent-spawner/main.go`.
- Streaming over bus + SSE + WebSocket + webhook:
  `examples/streaming/main.go`.
- Multiple Kits in one process: `examples/multi-kit/main.go`.
- Kits over a shared external NATS: `examples/cross-kit/main.go`.
- Standalone plugin binary: `examples/plugin-author/main.go`.

Every example is `go run`-able from the repo root.
