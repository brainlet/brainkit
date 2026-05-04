# Go SDK

The Go API surface you write against is the `brainkit` package
(Kit, Config, startup builders, Call wrappers) plus a thin `sdk` package
that owns the message envelope and typed message shapes. This guide
covers every piece you'd reach for in a production program.

## The Kit

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/transports/embeddednats"
)

kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: embeddednats.New(),
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

Identity helpers:

```go
kit.Namespace()      // string
kit.CallerID()       // string stamped into outbound metadata
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
| `TraceStore` | `TraceStore` | Explicit span store. Nil leaves tracing no-op unless a tracing module attaches one. |
| `TraceSampleRate` | `float64` | 0.0–1.0. Default 1.0. |
| `Store` | `KitStore` | Persistence for deployments, schedules, plugins. `nil` = ephemeral. |
| `Logger` | `*slog.Logger` | Default `slog.Default()`. |
| `LogHandler` | `func(LogEntry)` | Tagged log stream from `.ts` and the runtime. |
| `ErrorHandler` | `func(error)` | Non-fatal error sink. |
| `MaxConcurrency` | `int` | Concurrent bus handler cap. 0 = unlimited. |
| `JSRuntime` | `bool` | Requests the embedded JS/TS runtime for deploy/eval/workflow/harness paths. Zero-value Kit leaves it off; import `modules/jsruntime` or `presets/standard` so the request can be satisfied. |
| `MaxStackSize` | `int` | QuickJS stack bytes. Default 1 MB. |
| `RetryPolicies` | `map[string]RetryPolicy` | Topic glob → retry config. |
| `Modules` | `[]module.Module` | Opt-in subsystems. |

## Runtime Admin Modules

After `New`, runtime provider/storage/vector/secret administration is exposed
by opt-in modules, not root Kit accessors.

```go
kit, err := brainkit.New(brainkit.Config{
    Transport: brainkit.Memory(),
    Modules: []module.Module{
        registrymod.New(),
        secretsmod.New(),
    },
})

providerConfig, _ := json.Marshal(map[string]any{
    "APIKey": os.Getenv("OPENAI_API_KEY"),
})
_, err = registrymsg.CallProviderAdd(kit, ctx, registrymsg.ProviderAddMsg{
    Name:   "openai",
    Type:   "openai",
    Config: providerConfig,
})

_, err = secretmsg.CallSecretsSet(kit, ctx,
    secretmsg.SecretsSetMsg{Name: "API_TOKEN", Value: "sk-..."})
```

Use `Config.Providers`, `Config.Storages`, and `Config.Vectors` for startup
configuration. Mount `modules/registry` when runtime mutation/listing is part
of the process contract. Mount `modules/secrets` when `secrets.*` bus commands
should exist.

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
For SQLite storage/vector resolution from deployed `.ts` code, link
the optional runtime bridge:

```go
import _ "github.com/brainlet/brainkit/storagebridges/sqlite"
```

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
`*sdk.NoDeadlineError`.

```go
resp, err := brainkit.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](
    kit, ctx,
    toolmsg.ToolCallMsg{Name: "echo", Input: map[string]any{"msg": "hi"}},
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

Generated `typed_gen.go` files ship call wrappers — one per typed
Msg/Resp pair — that saturate the generics so your call sites stay
readable without adding module command names to the root `brainkit`
API. SDK-owned commands live in `sdk/typed_gen.go`; module-owned
commands live with their module message package:

```go
resp, err := toolmsg.CallToolCall(kit, ctx, toolmsg.ToolCallMsg{...})
resp, err := secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{Name: "API_TOKEN"})
resp, err := auditmsg.CallAuditQuery(kit, ctx, auditmsg.AuditQueryMsg{...})
resp, err := schedulemsg.CallScheduleCreate(kit, ctx, schedulemsg.ScheduleCreateMsg{...})
resp, err := mcpmsg.CallMcpListTools(kit, ctx, mcpmsg.McpListToolsMsg{})
resp, err := health.CallKitHealth(kit, ctx, health.KitHealthMsg{})
```

Your editor's autocomplete on `Call` in the message-owning package
will show the typed helper set. Use generic `brainkit.Call` when you
need Kit-specific options such as topology-aware `brainkit.WithCallTo`.
Module code that only has the narrow request/reply capability can use the
matching `CallXxxWithCaller(caller, ctx, msg, opts...)` wrapper instead of
constructing a fake runtime.

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

### Diagnostics-Only Protocol Access

The typed Call helpers are the normal path. `sdk/protocol` is for transport
diagnostics, protocol bridges, and tests that intentionally inspect wire
payloads. Application code should use `brainkit.Call`, `brainkit.CallStream`,
or generated module-owned `CallXxx` helpers.

```go
import (
    "github.com/brainlet/brainkit/sdk"
)

type DomainEvent struct { ID string `json:"id"` }
func (DomainEvent) BusTopic() string { return "domain.event" }

err := sdk.Emit(kit, ctx, DomainEvent{ID: "evt-123"})
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

err := kit.Mount(ctx, toolsmod.GoTool("math.add", toolsmod.TypedTool[AddInput]{
    Description: "Return a + b as a typed sum.",
    Execute: func(_ context.Context, in AddInput) (any, error) {
        return AddOutput{Sum: in.A + in.B}, nil
    },
}))
```

`GoTool[T]` returns a module; mounting it registers the tool for that
module scope. Schema derives from struct tags via reflection. Invoke:

```go
resp, err := toolmsg.CallToolCall(kit, ctx, toolmsg.ToolCallMsg{
    Name:  "math.add",
    Input: map[string]any{"a": 40, "b": 2},
})
// resp.Result == json.RawMessage(`{"sum":42}`)
```

See [`examples/go-tools/`](../../examples/go-tools/).

## Deploying TypeScript

`.ts` services are built as packages and evaluated inside SES
Compartments. `standard.PackageSet()` and `standard.CommandSet()` include the
standard source package builder. Custom Kit assemblies that mount
`packages.New()` directly should also import the package builder they want,
usually `_ "github.com/brainlet/brainkit/modules/packages/bundlers/esbuild"`.
Caller-side deploy helpers are in
`github.com/brainlet/brainkit/modules/packages/client`.

```go
// Inline — handy for tests and demos.
packageclient.Deploy(ctx, kit, packageclient.Inline("greeter", "greeter.ts",
    `bus.on("hello", (m) => m.reply({ok: true}));`))

// Single file from disk.
pkg, err := packageclient.FromFile("./svc/agent.ts")
if err == nil {
    _, err = packageclient.Deploy(ctx, kit, pkg)
}

// Directory with a brainkit.yaml manifest (multi-file packages).
pkg, err = packageclient.FromDir("./svc")
if err == nil {
    _, err = packageclient.Deploy(ctx, kit, pkg)
}

// List everything currently deployed.
names, err := packageclient.List(ctx, kit)

// Get the source / manifest of a deployment.
pkgInfo, ok, err := packageclient.Get(ctx, kit, "greeter")

// Remove a deployment. All resources it registered are dropped.
err = packageclient.Teardown(ctx, kit, "greeter")
```

Deployments register handlers on the mailbox namespace
`ts.<deployment-name>.<topic>` — the topic string used by
`bus.on(...)` inside the `.ts` file.

## Module composition

Modules are opt-in subsystems that extend the Kit with scoped resources
such as bus commands, tools, subscriptions, and hooks. They implement the
hot-mount interface:

```go
type Module interface {
    ID() string
    Mount(context.Context, module.Host) error
}
```

Modules may additionally implement `module.StatusReporter` to expose a
maturity tag (`stable`, `beta`, `wip`).

Shipped modules live under `modules/`:

| Module | Constructor | Status |
|---|---|---|
| `modules/agents` | `agents.New()` | stable |
| `modules/audit` | `audit.NewModule(audit.Config{...})` | stable |
| `modules/control` | `control.New()` | stable |
| `modules/discovery` | `discovery.NewModule(discovery.ModuleConfig{...})` | beta |
| `modules/eval` | `eval.New()` | beta |
| `modules/gateway` | `gateway.New(gateway.Config{...})` | stable |
| `modules/health` | `health.New()` | stable |
| `modules/harness` | `harness.NewModule(harness.Config{...})` | WIP |
| `modules/jsruntime` | `jsruntime.New()` | beta |
| `modules/mcp` | `mcpmod.New(map[string]mcpmod.ServerConfig{...})` | stable |
| `modules/messaging` | `messaging.New()` | stable |
| `modules/metrics` | `metrics.New()` | stable |
| `modules/packages` | `packages.New()` | stable |
| `modules/plugins` | `pluginsmod.NewModule(pluginsmod.Config{...})` | stable |
| `modules/probes` | `probes.New(probes.Config{...})` | beta |
| `modules/reference` | `reference.New()` | stable |
| `modules/registry` | `registry.New()` | stable |
| `modules/schedules` | `schedulesmod.NewModule(schedulesmod.Config{...})` | beta |
| `modules/secrets` | `secrets.New()` | stable |
| `modules/testing` | `testing.New()` | beta |
| `modules/tools` | `tools.New()` | stable |
| `modules/topology` | `topology.NewModule(topology.Config{...})` | beta |
| `modules/tracing` | `tracing.New(tracing.Config{...})` | beta |
| `modules/workflow` | `workflowmod.New()` | stable |

Wire them by passing to `Config.Modules`:

```go
import (
    "github.com/brainlet/brainkit/modules/audit"
    "github.com/brainlet/brainkit/modules/tracing"
    "github.com/brainlet/brainkit/transports/embeddednats"
)

kit, err := brainkit.New(brainkit.Config{
    Namespace: "obs-demo",
    Transport: embeddednats.New(),
    FSRoot:    "/tmp/obs",
    Modules: []module.Module{
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

Raw cross-namespace publish / subscribe exists for modules and transport
diagnostics, but the `WithCallTo` option on `Call` / `CallStream` is the normal
application path.

## The server package

For long-running processes, `brainkit/server` composes a Kit with a
YAML config and explicit standard profiles.

```go
import (
	"github.com/brainlet/brainkit/server"
	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/packageboot"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
	_ "github.com/brainlet/brainkit/server/configfile/transportbackends/embeddednats"
	_ "github.com/brainlet/brainkit/server/standard/commands"
	_ "github.com/brainlet/brainkit/server/standard/server"
)

cfg, err := configfile.Load("brainkit.yaml")
srv, err := server.New(cfg)
defer srv.Close()

ctx, stop := signal.NotifyContext(context.Background(),
    syscall.SIGINT, syscall.SIGTERM)
defer stop()

if err := srv.Start(ctx); err != nil { log.Fatal(err) }
```

For programmatic use without a YAML file:

```go
import "github.com/brainlet/brainkit/server/quickstart"

srv, err := quickstart.New("my-app", "/var/lib/my-app",
    quickstart.WithListen(":8080"),
    quickstart.WithSecretKey(os.Getenv("BRAINKIT_SECRET_KEY")),
    quickstart.WithPackages(myPackage),
    quickstart.WithExtraModules(myModule),
)
```

See [`examples/hello-server/`](../../examples/hello-server/).

## Error types

Every typed call error is matchable with `errors.As`:

```go
import (
    "errors"
    "github.com/brainlet/brainkit/sdk"
)

_, err := brainkit.Call[...](kit, ctx, req)
if err != nil {
    var (
        timeout *sdk.CallTimeoutError
        cancel  *sdk.CallCancelledError
        decode  *sdk.CallDecodeError
        noDead  *sdk.NoDeadlineError
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
