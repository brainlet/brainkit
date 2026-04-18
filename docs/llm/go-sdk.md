# Go SDK — API Reference

Dense reference for the `brainkit` Go surface. Canonical source: `github.com/brainlet/brainkit` at v1.0.0-rc.1.

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/sdk"
    "github.com/brainlet/brainkit/sdk/sdkerrors"
)
```

Two packages form the public Go surface:

- `brainkit` — runtime construction (`Kit`, `Config`, transports, providers, storages, vectors, modules, `Call`/`CallStream`, generated call wrappers).
- `sdk` — bus-level primitives that do not depend on a `Kit`: `Runtime` interfaces, typed-message contracts, `Publish`/`Emit`/`SubscribeTo`/`Reply`, envelopes, typed errors.

A third package, `brainkit/server`, composes a `Kit` with the standard module set — documented in `go-config.md`.

---

## 1. `sdk.Runtime` interfaces

`Kit` implements all three. Plugin clients implement `Runtime` only.

```go
package sdk

type Runtime interface {
    PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRaw(ctx context.Context, topic string, handler func(Message)) (cancel func(), err error)
    Close() error
}

type CrossNamespaceRuntime interface {
    Runtime
    PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(Message)) (cancel func(), err error)
}

type Replier interface {
    ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error
}
```

`SubscribeRaw` must be ready to receive before it returns — no handshake race against `Publish`. `correlationId` and `replyTo` are carried on the Watermill message metadata.

---

## 2. Bus primitives (`sdk` helpers)

### 2.1 Messages

```go
type BrainkitMessage interface {
    BusTopic() string
}

type Message struct {
    Topic    string
    Payload  []byte
    CallerID string
    TraceID  string
    Metadata map[string]string // includes correlationId, replyTo, envelope
}

type CustomMsg struct {
    Topic   string          // wire-only; not marshaled
    Payload json.RawMessage // serialized directly — no wrapper
}
func (m CustomMsg) BusTopic() string { return m.Topic }
```

`CustomMsg` is the generic escape hatch for calling `.ts`-deployed topics (`ts.<name>.<topic>`) whose request shape is not a typed Go struct.

### 2.2 Publish / Emit

```go
type PublishResult struct {
    MessageID     string
    CorrelationID string
    ReplyTo       string
    Topic         string
}

type PublishOption func(*publishConfig)
func WithReplyTo(topic string) PublishOption

// Publish[T] sends a command, always with a replyTo (auto or overridden).
// Default replyTo: <topic>.reply.<uuid>.
func Publish[T BrainkitMessage](rt Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error)

// Emit[T] is fire-and-forget. No replyTo, no response expected.
func Emit[T BrainkitMessage](rt Runtime, ctx context.Context, msg T) error
```

### 2.3 Subscribe

```go
// Unwraps success envelopes automatically; error envelopes invoke the
// handler with zero T and non-nil metadata so callers can inspect via
// sdk.DecodeEnvelope(msg.Payload).
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string,
    handler func(T, Message)) (func(), error)
```

### 2.4 Reply / Stream

```go
// Reply publishes a terminal envelope (done=true).
func Reply(rt Runtime, ctx context.Context, msg Message, payload any) error

// SendChunk publishes an intermediate frame (done=false). Use for streaming.
func SendChunk(rt Runtime, ctx context.Context, msg Message, payload any) error
```

Both require `rt` to satisfy `Replier`. Both read `replyTo` + `correlationId` from `msg.Metadata`. Return `ErrNotReplier` or `ErrNoReplyTo` on failure.

### 2.5 Cross-namespace

```go
// Routes to the specified Kit's namespace. rt must implement
// CrossNamespaceRuntime.
func PublishTo[T BrainkitMessage](rt Runtime, ctx context.Context,
    targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error)
```

### 2.6 Addressing `.ts` services

```go
// Convention: "my-agent.ts" + "ask" → "ts.my-agent.ask"
// "nested/svc" + "rpc" → "ts.nested.svc.rpc"
func SendToService(rt Runtime, ctx context.Context, service, topic string,
    payload any, opts ...PublishOption) (PublishResult, error)

func ResolveServiceTopic(service, topic string) string
```

---

## 3. Envelopes

Wire shape for every bus response:

```go
type Envelope struct {
    Ok    bool            `json:"ok"`
    Data  json.RawMessage `json:"data,omitempty"`
    Error *EnvelopeError  `json:"error,omitempty"`
}

type EnvelopeError struct {
    Code    string         `json:"code"`
    Message string         `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

func EnvelopeOK(data any) Envelope
func EnvelopeErr(code, message string, details map[string]any) Envelope
func EncodeEnvelope(e Envelope) ([]byte, error)
func DecodeEnvelope(payload []byte) (Envelope, error)
func IsEnvelope(payload []byte) bool
func FromEnvelope(e Envelope) error    // rehydrates a typed Go error
func ToEnvelope(data any, err error) Envelope
```

Success envelopes carry the raw response JSON in `Data`. Error envelopes carry a machine-readable `Code` (`NOT_FOUND`, `VALIDATION_ERROR`, `TIMEOUT`, …) plus structured `Details`. `FromEnvelope` reconstructs the exact typed error (see §8).

The `SubscribeTo[T]` helper unwraps envelopes transparently; only raw subscribers (`rt.SubscribeRaw`) need to call `DecodeEnvelope` manually.

---

## 4. `brainkit.Kit`

```go
package brainkit

type Kit struct { /* opaque */ }

// New creates a runtime from Config. Zero-value Transport defaults to
// Memory() — an in-process GoChannel, no disk side-effects, no plugins.
// Use EmbeddedNATS(), NATS(url), AMQP(url), Redis(url) for real pub/sub.
func New(cfg Config) (*Kit, error)

func RuntimeID() string // process-level identity shared by all Kits in the same OS process
```

### 4.1 `sdk.Runtime` methods

```go
func (k *Kit) PublishRaw(ctx, topic, payload) (correlationID string, err error)
func (k *Kit) SubscribeRaw(ctx, topic, handler) (cancel func(), err error)
func (k *Kit) PublishRawTo(ctx, targetNS, topic, payload) (correlationID string, err error)
func (k *Kit) SubscribeRawTo(ctx, targetNS, topic, handler) (cancel func(), err error)
func (k *Kit) ReplyRaw(ctx, replyTo, correlationID, payload, done bool) error
```

### 4.2 Lifecycle

```go
func (k *Kit) Close() error                    // drains for 5s, then closes
func (k *Kit) Shutdown(ctx context.Context) error // graceful drain
func (k *Kit) Alive(ctx context.Context) bool
func (k *Kit) Ready(ctx context.Context) bool
func (k *Kit) IsDraining() bool
func (k *Kit) ShutdownSignal() <-chan struct{}
```

### 4.3 Identity + introspection

```go
func (k *Kit) Namespace() string
func (k *Kit) CallerID() string
func (k *Kit) Logger() *slog.Logger
func (k *Kit) TransportKind() string // "memory", "embedded", "nats", "amqp", "redis"
func (k *Kit) PresenceTransport() transport.Presence
func (k *Kit) Remote() *transport.RemoteClient
func (k *Kit) Tracer() *tracing.Tracer
func (k *Kit) Tools() *tools.ToolRegistry
func (k *Kit) Store() types.KitStore
func (k *Kit) SecretStore() secrets.SecretStore
func (k *Kit) Audit() *audit.Recorder
func (k *Kit) HasCommand(topic string) bool
```

### 4.4 Deployments

```go
type DeployResult struct {
    Name      string
    Version   string
    Source    string
    Resources []sdk.ResourceInfo
}

type DeploymentInfo struct {
    Name    string
    Version string
    Source  string
    Status  string
}

func (k *Kit) Deploy(ctx context.Context, pkg Package) (DeployResult, error)
func (k *Kit) Teardown(ctx context.Context, name string) error
func (k *Kit) Get(ctx context.Context, name string) (DeploymentInfo, bool, error)
func (k *Kit) List(ctx context.Context) ([]DeploymentInfo, error)
```

`Deploy` hot-replaces an existing deployment with the same name. Internally calls `package.deploy`/`package.teardown`/`package.info`/`package.list` bus topics.

### 4.5 Packages

```go
type Package struct {
    Name    string
    Version string
    Entry   string
    Files   map[string]string
}

// Inline — name, entry filename, single source string.
func PackageInline(name, entry, source string) Package

// Directory containing manifest.json + source files. Bundled via esbuild
// at deploy time.
func PackageFromDir(dir string) (Package, error)

// Single .ts file. Name = filename stem, imports resolved at deploy.
func PackageFromFile(path string) (Package, error)
```

### 4.6 Accessors (`Providers`, `Storages`, `Vectors`, `Secrets`)

```go
func (k *Kit) Providers() *Providers
func (k *Kit) Storages()  *Storages
func (k *Kit) Vectors()   *Vectors
func (k *Kit) Secrets()   *Secrets

type Providers struct{ /* opaque */ }
func (p *Providers) Register(name string, typ AIProviderType, config any) error
func (p *Providers) Unregister(name string)
func (p *Providers) List() []ProviderInfo
func (p *Providers) Get(name string) (AIProviderRegistration, bool)
func (p *Providers) Has(name string) bool

type Storages struct{ /* opaque */ }
func (s *Storages) Register(name string, typ StorageType, config any) error
func (s *Storages) Unregister(name string)
func (s *Storages) List() []StorageInfo
func (s *Storages) Get(name string) (types.StorageRegistration, bool)
func (s *Storages) Has(name string) bool

type Vectors struct{ /* opaque */ }
// symmetric: Register / Unregister / List / Get / Has over VectorStoreType

type Secrets struct{ /* opaque */ }
func (s *Secrets) Set(ctx, name, value string) error
func (s *Secrets) Get(ctx, name string) (string, error)
func (s *Secrets) Delete(ctx, name string) error
func (s *Secrets) List(ctx) ([]SecretMeta, error)
func (s *Secrets) Rotate(ctx, name, newValue string) error
```

Without `Config.SecretKey`, `Secrets.*` return a cleartext-env-only fallback; mutators return `"no secret store configured"` error. `Rotate` is `Set`; restart-on-rotation is handled by the plugins module when present.

### 4.7 Tools

```go
type TypedTool[T any] = tools.TypedTool[T]
type RegisteredTool     = tools.RegisteredTool
type GoFuncExecutor     = tools.GoFuncExecutor

func RegisterTool[T any](k *Kit, name string, tool TypedTool[T]) error
func (k *Kit) RegisterRawTool(t RegisteredTool) error
```

`RegisterTool` is the typed path; `RegisterRawTool` is used by modules that proxy non-Go executors (MCP, plugin tools).

### 4.8 Modules

```go
type Module interface {
    Name() string
    Init(k *Kit) error
    Close() error
}

type ModuleStatus = string
const (
    ModuleStatusStable ModuleStatus = "stable"
    ModuleStatusBeta   ModuleStatus = "beta"
    ModuleStatusWIP    ModuleStatus = "wip"
)

type StatusReporter interface {
    Status() ModuleStatus
}

// Commands
type CommandSpec = engine.CommandSpec
func Command[Req sdk.BrainkitMessage, Resp any](
    handler func(context.Context, Req) (*Resp, error),
) CommandSpec
func (k *Kit) RegisterCommand(spec CommandSpec)

// Lookup / control
func (k *Kit) Module(name string) (Module, bool)
func (k *Kit) CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
func (k *Kit) ProbeAll()
func (k *Kit) ReportError(err error, ctx ErrorContext)

// Hooks modules use to wire into the kit
func (k *Kit) SetScheduleHandler(h ScheduleHandler)
func (k *Kit) SetAuditStore(s AuditStore)
func (k *Kit) SetAuditVerbosity(v AuditVerbosity)
func (k *Kit) SetTraceStore(store TraceStore)
func (k *Kit) SetPluginChecker(pc deploy.PluginChecker)
func (k *Kit) SetPluginRestarter(r PluginRestarter)
func (k *Kit) HarnessRuntime() any
```

Modules extend a `Kit` by registering bus commands during `Init`. They are evaluated in order from `Config.Modules` and closed in reverse order on `Kit.Close`. See `go-config.md` for the bundled module constructors.

---

## 5. `Call` — synchronous request/response

```go
type CallOption func(*callConfig)

func WithCallTimeout(d time.Duration) CallOption
func WithCallTo(peerName string) CallOption          // peer name → namespace (via topology module or raw NS)
func WithCallMeta(meta map[string]string) CallOption
func WithCallBuffer(n int) CallOption                // CallStream only
func WithCallBufferPolicy(p BufferPolicy) CallOption // CallStream only
func WithCallNoCancelSignal() CallOption             // disables _brainkit.cancel emission

type BufferPolicy = caller.BufferPolicy
const (
    BufferBlock       = caller.BufferBlock
    BufferDropNewest  = caller.BufferDropNewest
    BufferDropOldest  = caller.BufferDropOldest
    BufferError       = caller.BufferError
)

// Deadline rule: ctx must carry a deadline, or WithCallTimeout must be
// set. Missing both → *caller.NoDeadlineError (Code "VALIDATION_ERROR").
func Call[Req sdk.BrainkitMessage, Resp any](
    k *Kit, ctx context.Context, req Req, opts ...CallOption,
) (Resp, error)

// Streams intermediate chunks via onChunk(); final reply decoded into Resp.
// Returning a non-nil error from onChunk finalizes with that error.
// Default buffer: 64 slots, BufferBlock.
func CallStream[Req sdk.BrainkitMessage, Chunk any, Resp any](
    k *Kit, ctx context.Context, req Req,
    onChunk func(Chunk) error, opts ...CallOption,
) (Resp, error)

// Lower-level access to the shared-inbox reply router.
func (k *Kit) Caller() *caller.Caller
```

Resp types of `json.RawMessage` short-circuit the decode and return raw bytes.

### 5.1 Generated `CallXxx` wrappers

For every request/response pair in `sdk/*_messages.go`, `cmd/sdkgen` emits a saturated wrapper at the package top level:

```go
func CallToolCall(k *Kit, ctx context.Context, msg sdk.ToolCallMsg,
    opts ...CallOption) (sdk.ToolCallResp, error) {
    return Call[sdk.ToolCallMsg, sdk.ToolCallResp](k, ctx, msg, opts...)
}
```

All 62 wrappers (one per bus command in `docs/bus-topics.md`):

```
CallAgentDiscover         CallAgentGetStatus        CallAgentList             CallAgentSetStatus
CallAuditPrune            CallAuditQuery            CallAuditStats
CallClusterPeers
CallGatewayRouteAdd       CallGatewayRouteList      CallGatewayRouteRemove    CallGatewayStatus
CallKitEval               CallKitHealth             CallKitSend               CallKitSetDraining
CallMcpCallTool           CallMcpListTools
CallMetricsGet
CallPackageDeploy         CallPackageDeployInfo     CallPackageListDeployed   CallPackageTeardown
CallPeersList             CallPeersResolve
CallPluginListRunning     CallPluginManifest        CallPluginRestart         CallPluginStart
CallPluginStatus          CallPluginStop
CallProviderAdd           CallProviderRemove
CallRegistryHas           CallRegistryList          CallRegistryResolve
CallScheduleCancel        CallScheduleCreate        CallScheduleList
CallSecretsDelete         CallSecretsGet            CallSecretsList           CallSecretsRotate
CallSecretsSet
CallStorageAdd            CallStorageRemove
CallTestRun
CallToolCall              CallToolList              CallToolResolve
CallTraceGet              CallTraceList
CallVectorAdd             CallVectorRemove
CallWorkflowCancel        CallWorkflowList          CallWorkflowRestart       CallWorkflowResume
CallWorkflowRuns          CallWorkflowStart         CallWorkflowStartAsync    CallWorkflowStatus
```

Regenerate with:

```
go run ./cmd/sdkgen -messages ./sdk -out ./sdk/typed_gen.go -call-out ./call_gen.go
```

---

## 6. Typed messages

Every `sdk.*Msg` type satisfies `BrainkitMessage` and declares its topic via `BusTopic()`. Responses are plain structs with no shared base type.

Catalog: see `docs/bus-topics.md` (generated from `sdk/*_messages.go`).

Representative subset:

```go
// Package lifecycle
type PackageDeployMsg struct {
    Path     string            `json:"path,omitempty"`     // dir/file path (bundled via esbuild)
    Manifest json.RawMessage   `json:"manifest,omitempty"` // inline name+entry
    Files    map[string]string `json:"files,omitempty"`    // inline file map
}
func (PackageDeployMsg) BusTopic() string { return "package.deploy" }

type PackageDeployResp struct {
    Name      string             `json:"name"`
    Version   string             `json:"version"`
    Source    string             `json:"source"`
    Resources []sdk.ResourceInfo `json:"resources"`
}

// Runtime introspection
type KitEvalMsg struct {
    Code string `json:"code"`
    Mode string `json:"mode,omitempty"` // "script" (default), "ts", "module"
}
func (KitEvalMsg) BusTopic() string { return "kit.eval" }

type KitHealthMsg struct{}
func (KitHealthMsg) BusTopic() string { return "kit.health" }
```

`KitEvalMsg.Mode` is whitelisted to `script`, `ts`, `module`. `ts` transpiles via esbuild before evaluation.

---

## 7. Custom topic calls (`.ts` mailboxes)

Every `.ts` deployment publishes its `bus.on(localTopic)` handlers under `ts.<name>.<localTopic>`. To call them from Go:

```go
raw := json.RawMessage(`{"prompt":"hi"}`)
reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx,
    sdk.CustomMsg{Topic: "ts.my-agent.ask", Payload: raw},
    brainkit.WithCallTimeout(30*time.Second),
)
```

`sdk.CustomMsg.MarshalJSON` serializes only the payload — the `.ts` subscriber receives `msg.payload` with the inner object directly (no `Topic` wrapper).

---

## 8. Errors

### 8.1 Sentinels (`errors.Is`)

```go
var ErrNoReplyTo          error // message has no replyTo metadata
var ErrNotReplier         error // runtime does not implement Replier
var ErrNotCrossNamespace  error // runtime does not implement CrossNamespaceRuntime
```

### 8.2 Typed errors (`errors.As`)

All implement `sdkerrors.BrainkitError` (`error` + `Code() string` + `Details() map[string]any`). Re-exported as type aliases from `sdk.*`.

| Type | Code | Fields |
|---|---|---|
| `NotFoundError` | `NOT_FOUND` | `Resource, Name` |
| `AlreadyExistsError` | `ALREADY_EXISTS` | `Resource, Name, Hint` |
| `ValidationError` | `VALIDATION_ERROR` | `Field, Message` |
| `TimeoutError` | `TIMEOUT` | `Operation` |
| `WorkspaceEscapeError` | `WORKSPACE_ESCAPE` | `Path` |
| `NotConfiguredError` | `NOT_CONFIGURED` | `Feature` |
| `TransportError` | `TRANSPORT_ERROR` | `Operation, Cause` |
| `PersistenceError` | `PERSISTENCE_ERROR` | `Operation, Source, Cause` |
| `DeployError` | `DEPLOY_ERROR` | `Source, Phase, Cause` |
| `BridgeError` | `BRIDGE_ERROR` | `Function, Cause` |
| `CompilerError` | `COMPILER_ERROR` | `Cause` |
| `CycleDetectedError` | `CYCLE_DETECTED` | `Depth` |
| `DecodeError` | `DECODE_ERROR` | `Topic, Cause` |
| `BusError` | any unknown | `Code_, Message, Details_` |

`sdk.BusError` is the fallback carrier when an envelope's `Code` doesn't map to a known typed error — all three fields are still accessible via `Code()`/`Error()`/`Details()`.

### 8.3 `brainkit.Call` errors

From `internal/bus/caller` (re-exported):

| Type | Code | Trigger |
|---|---|---|
| `*caller.NoDeadlineError` | `VALIDATION_ERROR` | ctx has no deadline and no `WithCallTimeout` |
| `*caller.CallTimeoutError` | `CALL_TIMEOUT` | deadline elapsed before terminal reply |
| `*caller.CallCancelledError` | `CALL_CANCELLED` | ctx cancelled (not deadline) |
| `*caller.DecodeError` | `CALL_DECODE_ERROR` | reply payload won't unmarshal into Resp |
| `*caller.BufferOverflowError` | `CALL_BUFFER_OVERFLOW` | stream buffer full, `BufferError` policy |
| `*caller.HandlerFailedError` | `HANDLER_FAILED` | remote handler exhausted retries |
| `caller.ErrCallerClosed` | — | call on a closed `Caller` |

---

## 9. Context keys (`sdk/ctxkeys`)

```go
// Propagates correlationId across local goroutines (e.g. when a handler
// fans out work through its own Publish calls).
func ctxkeys.WithPublishMeta(ctx context.Context,
    correlationID, replyTo string) context.Context
```

Used internally by `Publish`/`PublishTo`. Callers rarely interact directly.

---

## 10. Error handling pattern

```go
reply, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{Name: "echo"})
if err != nil {
    var nf *sdk.NotFoundError
    var vl *sdk.ValidationError
    switch {
    case errors.As(err, &nf):
        // nf.Resource, nf.Name
    case errors.As(err, &vl):
        // vl.Field, vl.Message
    case errors.Is(err, sdk.ErrNotCrossNamespace):
        // runtime doesn't support WithCallTo
    }
}
```

Every typed error is round-trippable through an envelope: `ToEnvelope(nil, err)` preserves `Code` + `Details`; `FromEnvelope(env)` rehydrates the concrete `*T`.

---

## 11. Minimal examples

### Embedded Kit, typed call

```go
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "demo",
    Transport: brainkit.Memory(), // zero-config default
    Providers: []brainkit.ProviderConfig{brainkit.OpenAI(os.Getenv("OPENAI_API_KEY"))},
})
defer kit.Close()

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := brainkit.CallToolList(kit, ctx, sdk.ToolListMsg{})
```

### Deploy a `.ts` package, call its mailbox

```go
pkg := brainkit.PackageInline("echo", "echo.ts", `
    bus.on("ask", (msg) => msg.reply({ text: msg.payload.prompt }));
`)
if _, err := kit.Deploy(ctx, pkg); err != nil { log.Fatal(err) }

reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx,
    sdk.CustomMsg{Topic: "ts.echo.ask", Payload: json.RawMessage(`{"prompt":"hi"}`)},
    brainkit.WithCallTimeout(5*time.Second),
)
```

### Streaming chunks

```go
resp, err := brainkit.CallStream[sdk.CustomMsg, sdk.StreamEvent, sdk.CustomMsg](
    kit, ctx, sdk.CustomMsg{Topic: "ts.agent.chat", Payload: req},
    func(ev sdk.StreamEvent) error {
        fmt.Println(ev.Type, string(ev.Data))
        return nil
    },
    brainkit.WithCallTimeout(60*time.Second),
    brainkit.WithCallBuffer(256),
    brainkit.WithCallBufferPolicy(brainkit.BufferDropOldest),
)
```

### Custom bus command on a Kit

```go
mod := myModule{}
kit, _ := brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{&mod},
})

// Inside myModule.Init(k *Kit):
k.RegisterCommand(brainkit.Command(func(ctx context.Context,
    req MyMsg) (*MyResp, error) {
    return &MyResp{Echo: req.Text}, nil
}))
```
