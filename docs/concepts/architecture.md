# Architecture

brainkit 1.0 boots an in-process runtime host. One call —
`brainkit.New(Config)` — returns a `*Kit` that owns a Watermill message bus,
a tool/provider/storage registry, optional QuickJS/TS execution, and any
modules you plug in. Everything runs in one OS process by default; transports
(NATS, Redis, AMQP) are opt-in knobs on the same type.

## The Kit

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "hello",
    Transport: brainkit.Memory(),
    JSRuntime: true,
    FSRoot:    ".",
})
defer kit.Close()
```

That Kit is the runtime. It is the only top-level object. A Kit:

- Can host **one QuickJS runtime** with SES locked down when `JSRuntime` is
  enabled or the `jsruntime` module is mounted ahead of JS-dependent modules.
  Deployed `.ts` services, JS agents, JS tools, and workflows share the same
  JS heap. Isolation is per-Compartment, not per-OS-process. A zero-value Kit
  can run as a lighter control plane without starting QuickJS.
- Owns **one Watermill router** (the bus). Every subsystem — Go, JS,
  plugins, gateway handlers — speaks to every other subsystem by
  publishing messages. There is no separate RPC layer.
- Seeds provider/storage/vector/secret registries from `Config`; runtime
  administration is exposed by opt-in modules such as `modules/registry` and
  `modules/secrets`.
- Loads **zero or more Modules** (gateway, audit, tracing, probes,
  topology, discovery, plugins, MCP, schedules, workflow, harness)
  which hot-mount scoped resources into the running Kit.
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
- **Package deployment helpers.** `modules/packages` owns
  `packages.Inline`, `packages.FromDir`, `packages.FromFile`, and
  `packages.Deploy(ctx, kit, pkg)`.
- **Bus calls.** The generic
  `Call[Req, Resp any](kit, ctx, req, opts…) (Resp, error)` plus
  `CallStream[Req, Chunk, Resp any]` for servers that emit chunks
  before a terminal reply.
- **Generated typed call wrappers** in package-owned `typed_gen.go`
  files that bind `sdk.Call` to typed bus topics shipped out of the
  box (`packagemsg.CallPackageDeploy`, `health.CallKitHealth`,
  `agentmsg.CallAgentDiscover`, `auditmsg.CallAuditQuery`,
  `topology.CallPeersResolve`, …). Regenerate them with `make generate`
  after adding new typed message types.
- **Typed tool registration.** `modules/tools.GoTool(name, TypedTool[T])`
  returns a scoped module that registers a typed Go function as a first-class
  tool. See `examples/go-tools/main.go`.
- **Lifecycle.** `kit.Shutdown(ctx)` drains gracefully, `kit.Close()`
  is the quick equivalent.

Everything else — envelopes, error codes, cross-namespace helpers,
message types — lives in `github.com/brainlet/brainkit/sdk` and is
consumed by both module authors and plugin authors.

## Modules

Modules are the extension point. A module implements the hot-mount
contract:

```go
type Module interface {
    ID() string
    Mount(context.Context, module.Host) error
}
```

Factories and modules can also expose a `module.Descriptor`. The descriptor is
the module manifest: status, module dependencies, owned commands, emitted
events, raw subscriptions, host capabilities, and generic resources. `Kit.Mount`
records command/subscription topics, provided capabilities, registered tools,
and explicit `Scope.Resource(...)` entries from `module.Host`, then
`Kit.MountedModules()` returns the live manifest snapshot. `brainkit modules
list --json` shows the registry-side manifests compiled into the binary.
When the control module is mounted, the same lifecycle is available over the
bus through `kit.modules`, `kit.module.describe`, `kit.module.mount`, and
`kit.module.unmount`; the CLI wraps these as `brainkit inspect modules` and
`brainkit modules inspect|mount|unmount`.

Optionally a module can implement `StatusReporter` to declare itself
`ModuleStatusStable`, `ModuleStatusBeta`, or `ModuleStatusWIP`. The
status surfaces through module descriptors and CLI listing so a caller
can refuse to boot when a WIP module is loaded in production.

Configured modules mount after the router starts, and `Kit.Mount` can
mount additional linked-code modules later. Module-owned commands,
tools, subscriptions, capabilities, goroutines, and HTTP servers are
leased into the module's scope and released on `Kit.Unmount`,
`Kit.Close`, or `Kit.Shutdown`. Runtime bus unmount refuses to remove a
module while another mounted module declares it in `Requires`. The 11
standard modules are:

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
packages.Deploy(ctx, kit, packages.Inline(
    "greeter", "greeter.ts",
    `bus.on("hello", (msg) => msg.reply({ greeting: "hi " + msg.payload.name }));`,
))
```

Under the hood, `Deploy` publishes a `packagemsg.PackageDeployMsg` on the
`package.deploy` topic. The handler runs the package entry through the
vendored microsoft/typescript-go transpiler, loads it into a fresh SES
Compartment, and exposes every `bus.on(topic, …)` at
`ts.<pkg>.<topic>`. Any subsequent Kit call to that topic enters the
Compartment, runs the JS handler, and replies through the bus. See
[deployment-pipeline.md](deployment-pipeline.md).

## Providers, Storages, Vectors, Secrets

Four typed registries are seeded by Kit config:

```go
brainkit.New(brainkit.Config{
    Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
    Storages: map[string]brainkit.StorageConfig{"main": brainkit.SQLiteStorage("./kit.db")},
    Vectors:  map[string]brainkit.VectorConfig{"docs": brainkit.SQLiteVector("./vectors.db")},
})
```

Each registry owns its own table of named backends. Deployed `.ts` code
sees the same table through `globalThis.__kit_providers` and calls it
through Mastra (`model("openai", "gpt-4o")`) or through
`kit.register(type, name, ref)` for tools/agents/workflows/memories. Mount
`modules/registry` or `modules/secrets` for runtime admin messages. See
[provider-registry.md](provider-registry.md).

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
