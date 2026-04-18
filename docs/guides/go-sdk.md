# Go SDK

The Go API surface you write against is the `brainkit` package
(Kit, Config, accessors, Call wrappers) plus a thin `sdk` package
that owns the message envelope and typed message shapes. This guide
covers every piece you'd reach for in a production program.

## The Kit

```go
import "github.com/brainlet/brainkit"

kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: brainkit.EmbeddedNATS(),
    FSRoot:    "/var/lib/my-app",
})
if err != nil { log.Fatal(err) }
defer kit.Close()
```

`brainkit.New` returns a `*Kit`. Everything else hangs off the Kit
or the package — there is no separate Runtime type you need to
construct.

Lifecycle:

| Method | Behaviour |
|---|---|
| `kit.Close()` | Fast shutdown with a 5 s drain timeout. |
| `kit.Shutdown(ctx)` | Graceful drain bound by `ctx`. |
| `kit.ShutdownSignal()` | `<-chan struct{}` closed when tear-down begins. Long-running goroutines select on it. |

Identity helpers:

```go
kit.Namespace()      // string
kit.CallerID()       // string stamped into outbound metadata
kit.TransportKind()  // "memory" | "embedded" | "nats" | "amqp" | "redis"
```

## Config

`brainkit.Config` is a flat struct. Zero value yields an in-memory
Kit with no persistence and auto-detected AI providers. Common
fields:

| Field | Type | Purpose |
|---|---|---|
| `Namespace` | `string` | Bus topic namespace. Default `"user"`. |
| `CallerID` | `string` | Identity in metadata. Defaults to `Namespace`. |
| `ClusterID` | `string` | Logical group (peers with same transport + cluster discover each other). Default `"default"`. |
| `Transport` | `TransportConfig` | See [transport-backends.md](transport-backends.md). |
| `FSRoot` | `string` | Filesystem sandbox for deployed `.ts`. |
| `Storages` | `map[string]StorageConfig` | Named KV / SQL backends resolved via `storage("name")` in `.ts`. |
| `Vectors` | `map[string]VectorConfig` | Named vector stores resolved via `vectorStore("name")` in `.ts`. |
| `Providers` | `[]ProviderConfig` | AI providers. Nil = auto-detect from env. |
| `EnvVars` | `map[string]string` | Overrides `os.Getenv` within this Kit. |
| `SecretKey` | `string` | Master key for the encrypted secret store. Empty = env-only dev mode. |
| `SecretStore` | `SecretStore` | Override the auto-created store. |
| `Tracing` | `bool` | Enable tracing with an auto-created in-memory store. |
| `TraceStore` | `TraceStore` | Override the auto-created trace store. |
| `TraceSampleRate` | `float64` | 0.0–1.0. Default 1.0. |
| `Store` | `KitStore` | Persistence for deployments, schedules, plugins. `nil` = ephemeral. |
| `Logger` | `*slog.Logger` | Default `slog.Default()`. |
| `LogHandler` | `func(LogEntry)` | Tagged log stream from `.ts` and the runtime. |
| `ErrorHandler` | `func(error)` | Non-fatal error sink. |
| `MaxConcurrency` | `int` | Concurrent bus handler cap. 0 = unlimited. |
| `MaxStackSize` | `int` | QuickJS stack bytes. Default 1 MB. |
| `RetryPolicies` | `map[string]RetryPolicy` | Topic glob → retry config. |
| `Modules` | `[]Module` | Opt-in subsystems. |

## Accessors

After `New`, four accessors manage registered resources:

```go
kit.Providers()  // *Providers — Register, Unregister, List, Get, Has
kit.Storages()   // *Storages  — (plural) same five methods
kit.Vectors()    // *Vectors   — same five methods
kit.Secrets()    // *Secrets   — Set, Get, Delete, List, Rotate
```

Example:

```go
kit.Providers().Register("openai", brainkit.AIProviderType("openai"),
    map[string]any{"apiKey": os.Getenv("OPENAI_API_KEY")})

list := kit.Storages().List()
for _, s := range list { fmt.Println(s.Name, s.Type) }

if err := kit.Secrets().Set(ctx, "API_TOKEN", "sk-..."); err != nil { ... }
```

Secrets require `Config.SecretKey`; otherwise `Set`/`Get`/`Delete`
return an error wrapping a `NotConfigured` shape.

See [`examples/secrets/`](../../examples/secrets/).

## Providers

Twelve builders in the `brainkit` package, each returning
`ProviderConfig`:

```go
brainkit.OpenAI(key, opts...)
brainkit.Anthropic(key, opts...)
brainkit.Google(key, opts...)
brainkit.Mistral(key, opts...)
brainkit.Groq(key, opts...)
brainkit.DeepSeek(key, opts...)
brainkit.XAI(key, opts...)
brainkit.Cohere(key, opts...)
brainkit.Perplexity(key, opts...)
brainkit.TogetherAI(key, opts...)
brainkit.Fireworks(key, opts...)
brainkit.Cerebras(key, opts...)
```

Options: `brainkit.WithBaseURL(url)`, `brainkit.WithHeaders(h)`.

Pass them in `Config.Providers`, or leave nil to auto-detect from
env (`OPENAI_API_KEY` → `openai`, `ANTHROPIC_API_KEY` →
`anthropic`, etc.).

## Storage + vector constructors

```go
brainkit.SQLiteStorage(path)
brainkit.PostgresStorage(dsn)
brainkit.MongoDBStorage(uri, dbName)
brainkit.UpstashStorage(url, token)
brainkit.InMemoryStorage()

brainkit.SQLiteVector(path)
brainkit.PgVectorStore(dsn)
brainkit.MongoDBVectorStore(uri, dbName)
```

Use the results as values inside `Config.Storages` / `Config.Vectors`.

## Calling over the bus

### Typed generic

```go
func Call[Req sdk.BrainkitMessage, Resp any](
    k *Kit, ctx context.Context, req Req, opts ...CallOption,
) (Resp, error)
```

Publishes `req` on `req.BusTopic()`, waits for the reply on a
private shared-inbox topic, returns the decoded response. Requires
either `ctx.Deadline()` or `WithCallTimeout`; otherwise returns
`*caller.NoDeadlineError`.

```go
resp, err := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](
    kit, ctx,
    sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"msg": "hi"}},
    brainkit.WithCallTimeout(2*time.Second),
)
```

For bespoke topics use `sdk.CustomMsg`:

```go
payload, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](
    kit, ctx,
    sdk.CustomMsg{
        Topic:   "ts.greeter.hello",
        Payload: json.RawMessage(`{"name":"world"}`),
    },
    brainkit.WithCallTimeout(2*time.Second),
)
```

### Generated wrappers

`call_gen.go` ships 62 generated wrappers — one per typed
Msg/Resp pair — that saturate the generics so your call sites stay
readable:

```go
resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{...})
resp, err := brainkit.CallAuditQuery(kit, ctx, sdk.AuditQueryMsg{...})
resp, err := brainkit.CallScheduleCreate(kit, ctx, sdk.ScheduleCreateMsg{...})
resp, err := brainkit.CallMcpListTools(kit, ctx, sdk.McpListToolsMsg{})
```

Your editor's autocomplete on `brainkit.Call` will show you the
full list.

### Streaming

```go
func CallStream[Req sdk.BrainkitMessage, Chunk any, Resp any](
    k *Kit, ctx context.Context, req Req,
    onChunk func(Chunk) error,
    opts ...CallOption,
) (Resp, error)
```

`onChunk` is invoked for every intermediate `msg.send(...)` chunk
in arrival order; the terminal `msg.reply(...)` is returned as
`Resp`. Returning a non-nil error from `onChunk` finalizes the call
with that error.

```go
var chunks []json.RawMessage
final, err := brainkit.CallStream[sdk.CustomMsg, json.RawMessage, json.RawMessage](
    kit, ctx,
    sdk.CustomMsg{Topic: "ts.streamer.stream", Payload: json.RawMessage(`{}`)},
    func(c json.RawMessage) error { chunks = append(chunks, c); return nil },
    brainkit.WithCallTimeout(10*time.Second),
    brainkit.WithCallBuffer(128),
    brainkit.WithCallBufferPolicy(brainkit.BufferBlock),
)
```

See [`examples/streaming/`](../../examples/streaming/).

### Call options

| Option | Effect |
|---|---|
| `WithCallTimeout(d)` | Absolute timeout. Earlier ctx deadline wins. |
| `WithCallTo(name)` | Route to a peer namespace. When the `topology` module is wired, `name` is resolved to a namespace; otherwise `name` is used verbatim. |
| `WithCallMeta(map)` | Append metadata to the published message. |
| `WithCallBuffer(n)` | Streaming: channel capacity (default 64). |
| `WithCallBufferPolicy(p)` | `BufferBlock` (default), `BufferDropNewest`, `BufferDropOldest`, `BufferError`. |
| `WithCallNoCancelSignal()` | Suppress the best-effort `_brainkit.cancel` publish on ctx cancel. |

### Raw envelope

The typed Call helpers are the normal path. If you need the raw
envelope (custom topics, pre-built payload, subscription
bookkeeping) use the `sdk` package directly:

```go
import "github.com/brainlet/brainkit/sdk"

pr, err := sdk.Publish(kit, ctx, sdk.ToolCallMsg{Name: "echo"})
// pr.ReplyTo, pr.CorrelationID, pr.MessageID, pr.Topic

unsub, err := sdk.SubscribeTo[sdk.ToolCallResp](kit, ctx, pr.ReplyTo,
    func(resp sdk.ToolCallResp, m sdk.Message) { /* ... */ })
defer unsub()

err = sdk.Emit(kit, ctx, sdk.PluginRegisteredEvent{Name: "cron"})
pr, err = sdk.SendToService(kit, ctx, "calc.ts", "add", map[string]int{"a": 1, "b": 2})
```

`Kit` implements `sdk.Runtime`, `sdk.CrossNamespaceRuntime`, and
`sdk.Replier`, so everything in `sdk/` that takes a Runtime accepts
it.

## Registering Go tools

Tools registered in Go are callable over the bus topic `tools.call`
from every surface — `.ts` code, plugins, other Go callers — and
the registry exposes them through `tools.list` with generated JSON
schema.

```go
type AddInput  struct { A int `json:"a"`; B int `json:"b"` }
type AddOutput struct { Sum int `json:"sum"` }

err := brainkit.RegisterTool(kit, "math.add", brainkit.TypedTool[AddInput]{
    Description: "Return a + b as a typed sum.",
    Execute: func(_ context.Context, in AddInput) (any, error) {
        return AddOutput{Sum: in.A + in.B}, nil
    },
})
```

`RegisterTool[T]` is a package-level generic function, not a Kit
method. Schema derives from struct tags via reflection. Invoke:

```go
resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
    Name:  "math.add",
    Input: map[string]any{"a": 40, "b": 2},
})
// resp.Result == json.RawMessage(`{"sum":42}`)
```

See [`examples/go-tools/`](../../examples/go-tools/).

## Deploying TypeScript

`.ts` services are built as packages and evaluated inside SES
Compartments.

```go
// Inline — handy for tests and demos.
kit.Deploy(ctx, brainkit.PackageInline("greeter", "greeter.ts",
    `bus.on("hello", (m) => m.reply({ok: true}));`))

// Single file from disk.
kit.Deploy(ctx, brainkit.PackageFromFile("./svc/agent.ts"))

// Directory with a brainkit.yaml manifest (multi-file packages).
kit.Deploy(ctx, brainkit.PackageFromDir("./svc"))

// List everything currently deployed.
names, err := kit.List(ctx)

// Get the source / manifest of a deployment.
pkg, err := kit.Get(ctx, "greeter")

// Remove a deployment. All resources it registered are dropped.
err = kit.Teardown(ctx, "greeter")
```

Deployments register handlers on the mailbox namespace
`ts.<deployment-name>.<topic>` — the topic string used by
`bus.on(...)` inside the `.ts` file.

## Module composition

Modules are opt-in subsystems that extend the Kit with additional
bus commands. They implement a three-method interface:

```go
type Module interface {
    Name() string
    Init(k *Kit) error
    Close() error
}
```

Modules may additionally implement `StatusReporter` to expose a
maturity tag (`stable`, `beta`, `wip`).

Shipped modules live under `modules/`:

| Module | Constructor | Status |
|---|---|---|
| `modules/audit` | `audit.NewModule(audit.Config{...})` | stable |
| `modules/discovery` | `discovery.NewModule(discovery.Config{...})` | stable |
| `modules/gateway` | `gateway.New(gateway.Config{...})` | stable |
| `modules/harness` | `harness.NewModule(harness.Config{...})` | WIP |
| `modules/mcp` | `mcpmod.New(map[string]mcpmod.ServerConfig{...})` | stable |
| `modules/plugins` | `pluginsmod.NewModule(pluginsmod.Config{...})` | stable |
| `modules/probes` | `probes.NewModule(probes.Config{...})` | stable |
| `modules/schedules` | `schedulesmod.NewModule(schedulesmod.Config{...})` | stable |
| `modules/topology` | `topology.NewModule(topology.Config{...})` | stable |
| `modules/tracing` | `tracing.New(tracing.Config{...})` | stable |
| `modules/workflow` | `workflowmod.New()` | stable |

Wire them by passing to `Config.Modules`:

```go
import (
    "github.com/brainlet/brainkit/modules/audit"
    "github.com/brainlet/brainkit/modules/tracing"
)

kit, err := brainkit.New(brainkit.Config{
    Namespace: "obs-demo",
    Transport: brainkit.EmbeddedNATS(),
    FSRoot:    "/tmp/obs",
    Modules: []brainkit.Module{
        audit.NewModule(audit.Config{Store: auditStore}),
        tracing.New(tracing.Config{Store: traceStore}),
    },
})
```

Order within the slice determines `Init` order and reverse `Close`
order — put dependency modules (audit, tracing) before modules
that use them.

## Cross-namespace calls

`WithCallTo("peer-name")` sends to a peer on the same transport.
When `modules/topology` is wired, `peer-name` is resolved through
its peer table; otherwise the name is treated as a raw namespace.
See [`examples/cross-kit/`](../../examples/cross-kit/) and
[`examples/multi-kit/`](../../examples/multi-kit/).

Raw cross-namespace publish / subscribe is also available via
`sdk.PublishTo` + `Kit.SubscribeRawTo`, but the `WithCallTo` option
on `Call` / `CallStream` is the normal path.

## The server package

For long-running processes, `brainkit/server` composes a Kit with a
YAML config, HTTP gateway, tracing, probes, audit, and (optionally)
plugins.

```go
import "github.com/brainlet/brainkit/server"

cfg, err := server.LoadConfig("brainkit.yaml")
srv, err := server.New(cfg)
defer srv.Close()

ctx, stop := signal.NotifyContext(context.Background(),
    syscall.SIGINT, syscall.SIGTERM)
defer stop()

if err := srv.Start(ctx); err != nil { log.Fatal(err) }
```

For programmatic use without a YAML file:

```go
srv, err := server.QuickStart("my-app", "/var/lib/my-app",
    server.WithListen(":8080"),
    server.WithSecretKey(os.Getenv("BRAINKIT_SECRET_KEY")),
    server.WithPackages("./packages/*"),
    server.WithExtraModules(myModule),
)
```

See [`examples/hello-server/`](../../examples/hello-server/).

## Error types

Every typed call error is matchable with `errors.As`:

```go
import (
    "errors"
    "github.com/brainlet/brainkit/internal/bus/caller"
    "github.com/brainlet/brainkit/sdk"
)

_, err := brainkit.Call[...](kit, ctx, req)
if err != nil {
    var (
        timeout *caller.CallTimeoutError
        cancel  *caller.CallCancelledError
        decode  *caller.DecodeError
        noDead  *caller.NoDeadlineError
        notFnd  *sdk.NotFoundError
        exists  *sdk.AlreadyExistsError
        valErr  *sdk.ValidationError
    )
    switch {
    case errors.As(err, &timeout):
    case errors.As(err, &cancel):
    case errors.As(err, &decode):
    case errors.As(err, &noDead):
    case errors.As(err, &notFnd):
    case errors.As(err, &exists):
    case errors.As(err, &valErr):
    default:
    }
}
```

Decode errors preserve the raw payload so you can log the wire bytes
when a schema drifts.

## Runtime interface

`Kit` implements this interface, and so do plugin clients:

```go
type Runtime interface {
    PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (cancel func(), err error)
    Close() error
}
```

Library code that accepts `sdk.Runtime` works with both surfaces
unchanged.
