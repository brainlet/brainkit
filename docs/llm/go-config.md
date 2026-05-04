# Go Config — API Reference

Dense reference for the brainkit Go configuration surface: `brainkit.Config`, provider / transport / storage / vector helpers, module constructors, persistence stores, and the `brainkit/server` composition layer. Canonical source: `github.com/brainlet/brainkit` at v1.0.0-rc.1.

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/server"

    "github.com/brainlet/brainkit/modules/audit"
    auditstores "github.com/brainlet/brainkit/modules/audit/stores"
    "github.com/brainlet/brainkit/modules/discovery"
    "github.com/brainlet/brainkit/modules/gateway"
    "github.com/brainlet/brainkit/modules/harness"
    "github.com/brainlet/brainkit/modules/mcp"
    pluginsmod "github.com/brainlet/brainkit/modules/plugins"
    "github.com/brainlet/brainkit/modules/probes"
    "github.com/brainlet/brainkit/modules/schedules"
    "github.com/brainlet/brainkit/modules/topology"
    "github.com/brainlet/brainkit/modules/tracing"
    "github.com/brainlet/brainkit/modules/workflow"
    "github.com/brainlet/brainkit/transports"
)
```

Two construction paths:

- `brainkit.New(brainkit.Config)` — bare runtime. Minimal modules, zero opinions.
- `server.New(server.Config)` — composed server: Kit + standard module set (gateway, tracing, probes, audit, optional plugins) behind one lifecycle.

Both take the same underlying `brainkit.ProviderConfig`, `brainkit.StorageConfig`, `brainkit.VectorConfig`, and `brainkit.TransportConfig` values — `server.Config` forwards them to the Kit verbatim.

---

## 1. `brainkit.Config`

```go
package brainkit

type Config struct {
    ClusterID       string                      // default "default"; peer-discovery scope
    Namespace       string                      // default "user"; per-Kit topic scope
    CallerID        string                      // default = Namespace; identity stamp on outbound messages

    Transport       TransportConfig             // zero-value = Memory() (see §3)
    FSRoot          string                      // deployment sandbox root
    Storages        map[string]StorageConfig    // named storage pool (§5)
    Vectors         map[string]VectorConfig     // named vector pool (§6)
    Providers       []ProviderConfig            // nil = env auto-detect (§2)
    EnvVars         map[string]string           // overrides os.Getenv for this Kit

    SecretKey       string                      // master key for encrypted secret store
    SecretStore     SecretStore                 // explicit store overrides SecretKey auto-create

    TraceStore      TraceStore                  // explicit span store; nil = no-op unless module attaches one
    TraceSampleRate float64                     // 0.0–1.0; default 1.0

    Store           KitStore                    // persistence for deploys + schedules + plugins

    Logger          *slog.Logger                // nil = slog.Default()
    LogHandler      func(LogEntry)              // tagged log entries from .ts + runtime
    ErrorHandler    func(error)                 // non-fatal errors

    MaxConcurrency  int                         // 0 = unlimited concurrent bus handlers
    JSRuntime       bool                        // enable embedded JS/TS deploy/eval runtime
    MaxStackSize    int                         // QuickJS stack bytes; default 1 MiB
    RetryPolicies   map[string]RetryPolicy      // per-topic retry (§10.1)

    Modules         []module.Module             // opt-in subsystems (§4)
}
```

### Defaults

| Field | Zero-value behavior |
|-------|---------------------|
| `ClusterID` | `"default"` |
| `Namespace` | `"user"` |
| `CallerID` | `Namespace` |
| `Transport` | `Memory()` — in-process GoChannel. `brainkit.QuickStart` flips this to `EmbeddedNATS()`. |
| `Providers` | Auto-detected from `os.Getenv` (`OPENAI_API_KEY` → openai, `ANTHROPIC_API_KEY` → anthropic, …) |
| `SecretKey` + `SecretStore` both unset | Env-only secret fallback (read from `os.Getenv`; writes return `NotConfiguredError`). |
| `Store` unset | No persistence — deployments, schedules, plugin state do not survive restart. |
| `Logger` | `slog.Default()` |
| `JSRuntime` | `false`; when true, root requests the registered `jsruntime` module. JS-dependent modules such as eval/packages/workflow also request it. |
| `MaxStackSize` | `1 * 1024 * 1024` |
| `TraceSampleRate` | `1.0` when a trace store is active |

Transport-connected Kits (`"embedded"`, `"nats"`, `"amqp"`, `"redis"`) resolve their JetStream / external connection during `brainkit.New`. For embedded NATS, JetStream persists under `<FSRoot>/nats-data` when `FSRoot != ""`; empty `FSRoot` gives ephemeral JetStream.

### `brainkit.New`

```go
func New(cfg Config) (*Kit, error)
```

Returns a `*Kit`. Use `Close()` for immediate shutdown, `Shutdown(ctx)` for graceful drain. Build errors are wrapped: `"brainkit: <reason>"`.

### `brainkit.QuickStart`

```go
func QuickStart(namespace, fsRoot string) (*Kit, error)
```

Bare-Kit shortcut: embedded NATS + SQLite at `<fsRoot>/kit.db`, no module set composed. For the batteries-included path including HTTP gateway, use `server/quickstart` (§13.3).

---

## 2. Providers (`ProviderConfig`)

```go
type ProviderConfig struct { /* unexported fields */ }
type ProviderOption func(*ProviderConfig)

func WithBaseURL(url string) ProviderOption
func WithHeaders(headers map[string]string) ProviderOption
```

Twelve constructors — each takes `apiKey string` plus optional `...ProviderOption`:

| Constructor | Provider key | Notes |
|-------------|--------------|-------|
| `OpenAI(key, opts...)` | `"openai"` | |
| `Anthropic(key, opts...)` | `"anthropic"` | |
| `Google(key, opts...)` | `"google"` | Gemini API |
| `Mistral(key, opts...)` | `"mistral"` | |
| `Groq(key, opts...)` | `"groq"` | |
| `DeepSeek(key, opts...)` | `"deepseek"` | |
| `XAI(key, opts...)` | `"xai"` | Grok |
| `Cohere(key, opts...)` | `"cohere"` | |
| `Perplexity(key, opts...)` | `"perplexity"` | |
| `TogetherAI(key, opts...)` | `"togetherai"` | |
| `Fireworks(key, opts...)` | `"fireworks"` | |
| `Cerebras(key, opts...)` | `"cerebras"` | |

The provider key is exposed to .ts code via `model("openai", "gpt-4o-mini")`. `WithBaseURL` overrides the OpenAI-compatible endpoint for self-hosted deployments; `WithHeaders` attaches custom HTTP headers.

```go
Providers: []brainkit.ProviderConfig{
    brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    brainkit.Anthropic(os.Getenv("ANTHROPIC_API_KEY"),
        brainkit.WithHeaders(map[string]string{"anthropic-beta": "tools-2024-05-16"}),
    ),
}
```

Runtime registration is available by mounting `modules/registry` and calling
the typed registry message wrappers:

```go
registrymsg.CallProviderAdd(kit, ctx, registrymsg.ProviderAddMsg{...})
registrymsg.CallProviderRemove(kit, ctx, registrymsg.ProviderRemoveMsg{...})
registrymsg.CallRegistryList(kit, ctx, registrymsg.RegistryListMsg{Category: "provider"})
registrymsg.CallRegistryHas(kit, ctx, registrymsg.RegistryHasMsg{Category: "provider", Name: name})
registrymsg.CallRegistryResolve(kit, ctx, registrymsg.RegistryResolveMsg{Category: "provider", Name: name})
```

---

## 3. Transports (`TransportConfig`)

```go
type TransportConfig struct { /* unexported fields */ }
type TransportOption func(*TransportConfig)
```

| Constructor | Backend | Notes |
|-------------|---------|-------|
| `Memory()` | GoChannel (`"memory"`) | Synchronous in-process. Tests and library-embedded use. |
| `EmbeddedNATS(opts...)` | In-process NATS JetStream (`"embedded"`) | Zero config; persistence under `<FSRoot>/nats-data` when `FSRoot` set. |
| `NATS(url, opts...)` | External NATS JetStream (`"nats"`) | Typical `url = "nats://host:4222"`. |
| `AMQP(url)` | RabbitMQ (`"amqp"`) | Typical `url = "amqp://user:pass@host:5672/"`. |
| `Redis(url)` | Redis Streams (`"redis"`) | Typical `url = "redis://host:6379"`. |

### Options

```go
func WithNATSName(name string) TransportOption // durable consumer prefix (NATS + EmbeddedNATS)
```

Zero-value `TransportConfig{}` is treated as `Memory()` by `brainkit.New`. Topic sanitization rules per backend:

| Backend | Sanitizer |
|---------|-----------|
| `Memory` | none |
| `EmbeddedNATS`, `NATS` | dots → dashes |
| `AMQP` | slashes → dashes |
| `Redis` | none |

`server.Config.Transport` rejects `Memory()` — nothing plugin- or cross-kit would work on an in-process channel.

---

## 4. Modules

`Config.Modules []module.Module` takes values from `github.com/brainlet/brainkit/module`:

```go
type Module interface {
    ID() string
    Mount(context.Context, module.Host) error
}

type module.Status = string
const (
    module.StatusStable module.Status = "stable"
    module.StatusBeta   module.Status = "beta"
    module.StatusWIP    module.Status = "wip"
)

type StatusReporter interface { Status() module.Status }
```

Configured modules are mounted in slice order after the router starts. Modules register bus topics through `module.Host.Commands().Handle(...)`, which returns scoped handles that are closed on unmount. `Close` is invoked in reverse mount order.

### Module catalog

| Module | Constructor | Status | Adds |
|--------|-------------|--------|------|
| agents | `agents.New()` | stable | `agents.list`, `agents.discover`, `agents.get-status`, `agents.set-status` |
| audit | `audit.NewModule(audit.Config{Store, Verbose, OwnStore})` | stable | `audit.query`, `audit.stats`, `audit.prune` |
| control | `control.New()` | stable | `kit.set-draining`, `kit.modules`, module mount/unmount/describe, `cluster.peers` |
| discovery | `discovery.NewModule(discovery.ModuleConfig{Type, StaticPeers, Heartbeat, TTL, Name})` | beta | Presence registration (static or bus) |
| eval | `eval.New()` | beta | `kit.eval` |
| gateway | `gateway.New(gateway.Config{Listen, Timeout, CORS, RateLimit, Stream, Middleware, NoHealth, NoBusAPI, Logger, Tracer})` | stable | HTTP: `/healthz`, `/readyz`, `POST /api/bus`, `POST /api/stream`, user routes |
| health | `health.New()` | stable | `kit.health` |
| harness | `harness.NewModule(harness.Config{Harness HarnessConfig})` | wip | Display/harness adapter (in flux) |
| jsruntime | `jsruntime.New()` | beta | Embedded JS/TS runtime capabilities |
| mcp | `mcp.New(map[string]mcp.ServerConfig{})` | stable | `mcp.listTools`, `mcp.callTool`; auto-registers MCP tools |
| messaging | `messaging.New()` | stable | `kit.send` |
| metrics | `metrics.New()` | stable | `metrics.get` |
| packages | `packages.New()` | stable | `package.deploy`, `package.teardown`, `package.list`, `package.info` |
| plugins | `pluginsmod.NewModule(pluginsmod.Config{Plugins []PluginConfig, Store})` | stable | `plugin.start/stop/restart/status/list`, subprocess supervision |
| probes | `probes.New(probes.Config{Interval, ProbeOnRegister})` | beta | Periodic provider / storage / vector probing |
| reference | `reference.New()` | stable | `kit.reference`, `kit.reference.list` |
| registry | `registry.New()` | stable | `registry.*`, `providers.*`, `storages.*`, `vectors.*` |
| schedules | `schedules.NewModule(schedules.Config{Store})` | beta | `schedules.create/list/cancel`; JS `bus.schedule` |
| secrets | `secrets.New()` | stable | `secrets.set/get/delete/list/rotate` |
| testing | `testing.New()` | beta | `test.run` |
| tools | `tools.New()` | stable | `tools.call`, `tools.resolve`, `tools.list` |
| topology | `topology.NewModule(topology.Config{Peers, Discovery})` | beta | `peers.list`, `peers.resolve`; `WithCallTo` resolver |
| tracing | `tracing.New(tracing.Config{Store TraceStore})` | beta | `trace.get`, `trace.list`; attaches a trace store |
| workflow | `workflow.New()` | stable | `workflow.start/startAsync/status/resume/cancel/list/runs/restart` |

### Module-specific helpers

```go
// tracing
tracing.NewSQLiteTraceStore(db *sql.DB, opts ...SQLiteTraceStoreOption) (*SQLiteTraceStore, error)
tracing.WithRetention(d time.Duration) SQLiteTraceStoreOption

// audit
auditstores.NewSQLite(dbPath string) (*auditstores.SQLite, error)
auditstores.NewPostgres(connStr string) (*auditstores.Postgres, error)

// discovery
discovery.NewStaticFromConfig(configs []PeerConfig) *Static
discovery.NewBus(BusConfig{Transport, Heartbeat, TTL}) *Bus

// modules/mcp
type ServerConfig struct { /* fields below */ }
```

### Discovery / topology configuration

```go
type discovery.ModuleConfig struct {
    Type        string         // "static" | "bus" | "" (disabled)
    StaticPeers []PeerConfig   // used when Type == "static"
    Heartbeat   time.Duration  // default 10s (bus mode)
    TTL         time.Duration  // default 30s (bus mode)
    Name        string         // self-peer override; "" = per-instance UUID
}

type discovery.Peer struct {
    Name, Namespace, Address string
    Meta                     map[string]string
}

type topology.Peer = discovery.Peer

type topology.Config struct {
    Peers     []Peer             // static list
    Discovery ProviderSource     // e.g. *discovery.Module
}
```

### MCP server config

```go
// modules/mcp
type ServerConfig struct {
    Command string            `json:"command,omitempty"` // stdio transport
    Args    []string          `json:"args,omitempty"`
    Env     map[string]string `json:"env,omitempty"`
    URL     string            `json:"url,omitempty"`     // SSE / HTTP transport
}
```

### Gateway config

```go
type gateway.Config struct {
    Listen     string                // ":8080"
    Timeout    time.Duration         // default 30s
    Middleware []Middleware
    CORS       *CORSConfig
    NoHealth   bool                  // disables /healthz + /readyz
    NoBusAPI   bool                  // disables POST /api/bus + /api/stream
    RateLimit  *RateLimitConfig
    Stream     *StreamConfig
    Logger     *slog.Logger          // nil = slog.Default()
    Tracer     Tracer                // optional; creates root spans for requests
}
```

---

## 5. Storage pool (`StorageConfig`)

```go
type StorageConfig struct {
    Type             string // "sqlite" | "postgres" | "mongodb" | "upstash" | "memory"
    Path             string // sqlite only
    ConnectionString string // postgres only
    URI              string // mongodb only
    DBName           string // mongodb only
    URL              string // upstash only
    Token            string // upstash only
}
```

Five constructors:

```go
brainkit.SQLiteStorage(path string) StorageConfig
brainkit.PostgresStorage(connStr string) StorageConfig
brainkit.MongoDBStorage(uri, dbName string) StorageConfig
brainkit.UpstashStorage(url, token string) StorageConfig
brainkit.InMemoryStorage() StorageConfig
```

```go
Storages: map[string]brainkit.StorageConfig{
    "main":  brainkit.SQLiteStorage("./data/app.db"),
    "cache": brainkit.UpstashStorage(os.Getenv("UPSTASH_URL"), os.Getenv("UPSTASH_TOKEN")),
}
```

Deployments reach the pool via `storage("main")` in `.ts` code.

Runtime registration is owned by `modules/registry`:

```go
registrymsg.CallStorageAdd(kit, ctx, registrymsg.StorageAddMsg{...})
registrymsg.CallStorageRemove(kit, ctx, registrymsg.StorageRemoveMsg{...})
registrymsg.CallRegistryList(kit, ctx, registrymsg.RegistryListMsg{Category: "storage"})
registrymsg.CallRegistryHas(kit, ctx, registrymsg.RegistryHasMsg{Category: "storage", Name: name})
registrymsg.CallRegistryResolve(kit, ctx, registrymsg.RegistryResolveMsg{Category: "storage", Name: name})
```

---

## 6. Vector pool (`VectorConfig`)

```go
type VectorConfig struct {
    Type             string // "sqlite" | "pgvector" | "mongodb"
    Path             string // sqlite only
    ConnectionString string // pgvector only
    URI              string // mongodb only
    DBName           string // mongodb only
}
```

Three constructors:

```go
brainkit.SQLiteVector(path string) VectorConfig
brainkit.PgVectorStore(connStr string) VectorConfig
brainkit.MongoDBVectorStore(uri, dbName string) VectorConfig
```

Deployments reach the pool via `vectorStore("name")` in `.ts` code. Runtime registration follows the same `modules/registry` shape:

```go
registrymsg.CallVectorAdd(kit, ctx, registrymsg.VectorAddMsg{...})
registrymsg.CallVectorRemove(kit, ctx, registrymsg.VectorRemoveMsg{...})
registrymsg.CallRegistryList(kit, ctx, registrymsg.RegistryListMsg{Category: "vectorStore"})
registrymsg.CallRegistryHas(kit, ctx, registrymsg.RegistryHasMsg{Category: "vectorStore", Name: name})
registrymsg.CallRegistryResolve(kit, ctx, registrymsg.RegistryResolveMsg{Category: "vectorStore", Name: name})
```

---

## 7. Persistence (`KitStore`)

```go
type KitStore interface {
    SaveDeployment(d PersistedDeployment) error
    LoadDeployments() ([]PersistedDeployment, error)
    LoadDeployment(source string) (PersistedDeployment, error)
    DeleteDeployment(source string) error

    SaveSchedule(s PersistedSchedule) error
    LoadSchedules() ([]PersistedSchedule, error)
    DeleteSchedule(id string) error
    ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error) // multi-replica dedup

    SaveInstalledPlugin(p InstalledPlugin) error
    LoadInstalledPlugins() ([]InstalledPlugin, error)
    DeleteInstalledPlugin(name string) error

    SaveRunningPlugin(p RunningPluginRecord) error
    LoadRunningPlugins() ([]RunningPluginRecord, error)
    DeleteRunningPlugin(name string) error

    Close() error
}
```

### Records

```go
type PersistedDeployment struct {
    Source       string
    Code         string
    Order        int
    DeployedAt   time.Time
    PackageName  string
    ArtifactKind DeployArtifactKind
}

type PersistedSchedule struct {
    ID         string
    Expression string
    Duration   time.Duration
    Topic      string
    Payload    json.RawMessage
    Source     string
    CreatedAt  time.Time
    NextFire   time.Time
    OneTime    bool
}

type InstalledPlugin struct {
    Name, Owner, Version, BinaryPath, Manifest string
    InstalledAt time.Time
}

type RunningPluginRecord struct {
    Name, Owner, Version, BinaryPath string
    Env        map[string]string
    Config     json.RawMessage
    StartOrder int
    StartedAt  time.Time
}
```

### Built-in implementations

```go
type SQLiteStore struct { DB *sql.DB }

func brainkit.NewSQLiteStore(path string) (*SQLiteStore, error)
func brainkit.NewPostgresStore(dsn string) (KitStore, error)
```

`NewSQLiteStore` uses `modernc.org/sqlite` (pure Go). WAL mode, `synchronous=NORMAL`, `busy_timeout=5000`. Creates the parent directory (`0755`) if missing.

Narrow persistence interfaces for modules (both satisfied by `*SQLiteStore` / the value returned by `NewPostgresStore`):

```go
// pluginsmod.Store
type Store interface {
    LoadRunningPlugins() ([]types.RunningPluginRecord, error)
    SaveRunningPlugin(r types.RunningPluginRecord) error
    DeleteRunningPlugin(name string) error
    LoadInstalledPlugins() ([]types.InstalledPlugin, error)
}

// schedules.Store
type Store interface {
    SaveSchedule(s types.PersistedSchedule) error
    LoadSchedules() ([]types.PersistedSchedule, error)
    DeleteSchedule(id string) error
    ClaimScheduleFire(scheduleID string, fireTime time.Time) (bool, error)
}
```

---

## 8. Secrets (`SecretStore`)

```go
type SecretStore interface {
    Get(ctx context.Context, name string) (string, error)
    Set(ctx context.Context, name, value string) error
    Delete(ctx context.Context, name string) error
    List(ctx context.Context) ([]SecretMeta, error)
    Close() error
}

type SecretMeta struct {
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
    Version   int       `json:"version"`
}
```

Priority: `Config.SecretStore` (explicit) > `Config.SecretKey` (auto-create `EncryptedKVStore`) > env-only fallback (`NotConfiguredError` on writes).

Runtime secret administration is owned by `modules/secrets`:

```go
secretmsg.CallSecretsSet(kit, ctx, secretmsg.SecretsSetMsg{...})
secretmsg.CallSecretsGet(kit, ctx, secretmsg.SecretsGetMsg{...})
secretmsg.CallSecretsDelete(kit, ctx, secretmsg.SecretsDeleteMsg{...})
secretmsg.CallSecretsList(kit, ctx, secretmsg.SecretsListMsg{})
secretmsg.CallSecretsRotate(kit, ctx, secretmsg.SecretsRotateMsg{...})
```

Plugin environment entries of the form `$secret:NAME` are resolved against this store when plugins boot; rotation restarts affected plugins when a plugin restarter module is wired (see `modules/plugins`).

---

## 9. Traces (`TraceStore`)

```go
type Span struct {
    TraceID, SpanID, ParentID string
    Name, Source              string
    StartTime                 time.Time
    Duration                  time.Duration
    Status                    string // "ok" | "error"
    Error                     string
    Attributes                map[string]string
}

type TraceStore interface {
    RecordSpan(span Span) error
    GetTrace(traceID string) ([]Span, error)
    ListTraces(query TraceQuery) ([]TraceSummary, error)
    Close() error
}

type TraceQuery struct {
    Since, Until   time.Time
    Source, Status string
    MinDuration    time.Duration
    Limit          int
}

type TraceSummary struct {
    TraceID   string
    RootSpan  string
    SpanCount int
    Duration  time.Duration
    Status    string
    StartTime time.Time
}
```

In-memory ring-buffer stores live in the internal tracing package and are used
by tests. Production code should pass an explicit store or mount the tracing
module.

Durable SQLite store:

```go
func tracing.NewSQLiteTraceStore(db *sql.DB, opts ...SQLiteTraceStoreOption) (*SQLiteTraceStore, error)
func tracing.WithRetention(d time.Duration) SQLiteTraceStoreOption // background cleanup loop
```

`brainkit.Config.TraceStore` attaches a store at startup. The tracing module can
attach or replace the store while mounted. `TraceSampleRate` (0.0-1.0) throttles
span recording when a store is active.

---

## 10. Retry, error, health types

### 10.1 RetryPolicy

```go
type RetryPolicy struct {
    MaxRetries      int
    InitialDelay    time.Duration
    MaxDelay        time.Duration
    BackoffFactor   float64
    DeadLetterTopic string
}

Config{
    RetryPolicies: map[string]brainkit.RetryPolicy{
        "workflow.*": {MaxRetries: 3, InitialDelay: time.Second, MaxDelay: 30 * time.Second, BackoffFactor: 2.0,
            DeadLetterTopic: "dlq.workflow"},
    },
}
```

Keys are glob patterns matched against bus topics. Applied to handler invocations; `DeadLetterTopic`, when non-empty, receives the failed message after retries are exhausted.

### 10.2 ErrorContext / LogEntry / ResourceInfo

```go
type ErrorContext struct {
    Operation string
    Component string
    Source    string
}

type LogEntry struct {
    Source, Level, Message string
    Time                   time.Time
}

type ResourceInfo struct {
    Type, ID, Name, Source string
    CreatedAt              int64
}
```

Internal runtime and module capabilities surface non-fatal errors through the configured handler.

### 10.3 HealthStatus / HealthCheck / KernelMetrics

```go
type HealthStatus struct {
    Healthy bool          `json:"healthy"`
    Status  string        `json:"status"`
    Uptime  time.Duration `json:"uptime"`
    Checks  []HealthCheck `json:"checks"`
}

type HealthCheck struct {
    Name    string        `json:"name"`
    Healthy bool          `json:"healthy"`
    Latency time.Duration `json:"latency,omitempty"`
    Error   string        `json:"error,omitempty"`
    Details any           `json:"details,omitempty"`
}

type KernelMetrics struct {
    ActiveHandlers    int64
    ActiveDeployments int
    ActiveSchedules   int
    PumpCycles        int64
    Uptime            time.Duration
    ActivePlugins     int
    Plugins           []PluginMetrics
    Bus               *MetricsSnapshot
    Transport         *TransportMetrics
}
```

Surfaced over the `kit.health` and `metrics.get` bus commands; `HealthStatus` and `KernelMetrics` are the decoded response types.
`Bus` contains logical-topic publish/handle/error counters plus handler duration summaries; `Transport` contains low-cardinality transport/router gauges.

---

## 11. Plugins and Schedules (inputs)

```go
// modules/plugins
type PluginConfig struct {
    Name            string
    Binary          string
    Args            []string
    Env             map[string]string
    Config          json.RawMessage
    AutoRestart     bool
    MaxRestarts     int
    StartTimeout    time.Duration
    ShutdownTimeout time.Duration
}

// modules/schedules
type ScheduleConfig struct {
    ID         string
    Expression string          // cron or "every 30s"
    Topic      string
    Payload    json.RawMessage
    Source     string          // audit attribution
}
```

`plugins.PluginConfig` feeds `modules/plugins.Config.Plugins`; `schedules.ScheduleConfig` is used by the `schedules.create` bus command via the schedules module handlers.

Secret interpolation: env values of the form `$secret:NAME` are resolved by the plugins module against the Kit secret store before the subprocess is started.

---

## 12. Package scaffolding `.d.ts` bundles

```go
// modules/packages owns package scaffolding and writes the embedded
// TypeScript declaration files into the generated package directory.
err := packages.ScaffoldPackage(dir, "name", "index.ts", source)
```

Each is a `string` whose value is the exact .d.ts text shipped with the runtime.

---

## 13. `brainkit/server` — composed runtime

`server` bundles an explicit Kit module set behind a single lifecycle. YAML loading, package boot hooks, standard module registration, and quickstart presets live in subpackages so importing core `server` stays light.

### 13.1 Config

```go
package server

type Config struct {
    Namespace    string                    // required
    Transport    brainkit.TransportConfig  // required; rejects Memory()
    FSRoot       string                    // required
    Store        brainkit.KitStore         // optional; configfile/quickstart install SQLite
    SecretKey    string                    // empty = cleartext fallback (warned)

    // Pass-through to brainkit.Config
    Providers []brainkit.ProviderConfig
    Storages  map[string]brainkit.StorageConfig
    Vectors   map[string]brainkit.VectorConfig

    Modules []module.Module                // explicit module set
    OnStart []server.StartHook             // optional startup hooks
}

type AuditConfig struct {
    Path    string // default: <FSRoot>/audit.db
    Verbose bool
}
```

Validation (`New` fails fast):

- `Namespace == ""` → error
- `Transport == TransportConfig{}` → error (explicit transport required)
- `FSRoot == ""` → error
- No module with ID `gateway` → error

### 13.2 Server

```go
type Server struct { /* unexported */ }

func New(cfg Config) (*Server, error)

func (s *Server) Start(ctx context.Context) error // runs OnStart hooks; blocks on ctx / SIGINT / SIGTERM
func (s *Server) Stop(ctx context.Context) error  // graceful drain via Kit.Shutdown
func (s *Server) Close() error                    // immediate shutdown
func (s *Server) Kit() *brainkit.Kit              // full underlying Kit
```

### 13.3 QuickStart

```go
func quickstart.New(namespace, fsRoot string, opts ...Option) (*server.Server, error)

type quickstart.Option func(*server.Config)
func quickstart.WithListen(addr string) Option                  // override :8080
func quickstart.WithSecretKey(key string) Option
func quickstart.WithPackages(pkgs ...packages.Package) Option
func quickstart.WithExtraModules(mods ...module.Module) Option
```

Defaults applied by `QuickStart`:

| Field | Value |
|-------|-------|
| `Transport` | `transports.EmbeddedNATS()` |
| `gateway` module | listen `":8080"` |
| standard command modules | on |
| `Store` | SQLite at `<fsRoot>/kit.db` |

### 13.4 YAML config

```go
func configfile.Load(path string) (server.Config, error)
```

Reads a YAML file, expands `$VAR` / `${VAR}` against `os.Getenv`, and projects onto `server.Config`. Import `server/standard` for built-in YAML module names. Unknown transport / provider types return errors; unknown storage / vector types default to `InMemoryStorage()` / `SQLiteVector(v.Path)`.

YAML shape:

```yaml
namespace: svc
fs_root: ./data
kit_store_path: ./data/kit.db        # optional; default <fs_root>/kit.db
secret_key: ${BRAINKIT_SECRET_KEY}

transport:
  type: nats                          # memory | embedded | nats | amqp | redis
  url: nats://nats:4222               # used by nats | amqp | redis
  nats_name: svc-prod                 # optional durable prefix

modules:
  jsruntime: {}
  gateway:
    listen: ":8080"
    timeout: 30s
  agents: {}
  reference: {}
  control: {}
  eval: {}
  health: {}
  messaging: {}
  metrics: {}
  registry: {}
  secrets: {}
  tools: {}
  packages: {}
  audit:
    path: ./data/audit.db
    verbose: false
  tracing:
    retention: 168h
  probes:
    interval: 60s

providers:
  - name: gpt          # label only; type + api_key are the wire fields
    type: openai       # openai | anthropic | google | mistral | groq | deepseek
                       # | xai | cohere | perplexity | togetherai | fireworks
                       # | cerebras
    api_key: ${OPENAI_API_KEY}

storages:
  main:
    type: sqlite       # sqlite | postgres | mongodb | upstash | memory
    path: ./data/app.db
  cache:
    type: upstash
    url: ${UPSTASH_URL}
    token: ${UPSTASH_TOKEN}

vectors:
  docs:
    type: pgvector     # sqlite | pgvector | mongodb
    connection_string: ${PG_DSN}

plugins:
  - name: postgres-mcp
    binary: /usr/local/bin/postgres-mcp
    env:
      PGURL: ${PG_DSN}

packages:
  - path: ./packages/api          # packages.FromDir(...)
```

Provider types accepted by `configfile.Load`: `openai`, `anthropic`, `google`, `mistral`, `groq`, `deepseek`, `xai`, `cohere`, `perplexity`, `togetherai`, `fireworks`, `cerebras`. Unknown types return `"server: unknown provider type %q"`.

---

## 14. Example compositions

### 14.1 Library-embedded Kit, no modules

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "app",
    Transport: brainkit.Memory(),
    FSRoot:    tmpDir,
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    },
})
if err != nil { return err }
defer kit.Close()
```

### 14.2 Standalone service with plugins, tracing, audit

```go
srv, err := server.New(server.Config{
    Namespace: "prod",
    Transport: transports.NATS("nats://nats:4222", transports.WithNATSName("prod")),
    FSRoot:    "/var/lib/brainkit",
    SecretKey: os.Getenv("BRAINKIT_SECRET_KEY"),

    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    },
    Storages: map[string]brainkit.StorageConfig{
        "main": brainkit.PostgresStorage(os.Getenv("PG_DSN")),
    },
    Vectors: map[string]brainkit.VectorConfig{
        "docs": brainkit.PgVectorStore(os.Getenv("PG_DSN")),
    },

    // Every module is constructed explicitly and passed via Modules.
    // The server validates that a "gateway" module is present.
    Modules: []module.Module{
        gateway.New(gateway.Config{Listen: ":8080"}),

        // Audit: open the SQLite store via the module's own stores
        // sub-package. Pass OwnStore:true so the module closes it at
        // shutdown.
        audit.NewModule(audit.Config{
            Store:    mustAuditStore("/var/lib/brainkit/audit.db"),
            OwnStore: true,
        }),

        // Tracing: open the SQLite-backed span store via the module's
        // NewSQLiteTraceStore + an *sql.DB you own. See the
        // module's Factory for an end-to-end example.
        tracing.New(tracing.Config{Store: mustTraceStore("/var/lib/brainkit/tracing.db")}),

        probes.New(probes.Config{Interval: 60 * time.Second, ProbeOnRegister: true}),

        // Plugins: pass the subprocess list; the module picks up the
        // KitStore from the Kit at Init time for persistence.
        plugins.NewModule(plugins.Config{Plugins: []plugins.PluginConfig{
            {Name: "pg-mcp", Binary: "/usr/local/bin/pg-mcp",
             Env: map[string]string{"PGURL": "$secret:PG_DSN"}},
        }}),
    },
    // mustAuditStore / mustTraceStore above are your helpers —
    // typically one-liners that call stores.NewSQLite / sql.Open +
    // tracing.NewSQLiteTraceStore and log.Fatal on error. For the
    // YAML-driven path, the registry factories construct these
    // stores for you from `modules.audit.path` / `modules.tracing.path`.

    OnStart: []server.StartHook{
        packageboot.Deploy(must(packages.FromDir("./packages/api"))),
    },
})
if err != nil { return err }
defer srv.Close()

ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
_ = srv.Start(ctx) // blocks until signal or ctx cancels
```

### 14.3 YAML-driven startup

```go
cfg, err := configfile.Load("/etc/brainkit/config.yaml")
if err != nil { log.Fatal(err) }
srv, err := server.New(cfg)
if err != nil { log.Fatal(err) }
defer srv.Close()
log.Fatal(srv.Start(ctx))
```

### 14.4 Custom module mix

```go
srv, err := server.New(server.Config{
    Namespace: "svc",
    Transport: transports.EmbeddedNATS(),
    FSRoot:    tmp,
    Modules: []module.Module{
        gateway.New(gateway.Config{Listen: ":8080"}),
        tracing.New(tracing.Config{Store: mustTraceStore(tmp + "/tracing.db")}),
        workflow.New(),
        schedules.NewModule(schedules.Config{Store: srvStore}),
        mcp.New(map[string]mcp.ServerConfig{
            "github": {URL: "https://mcp.github.com/sse"},
        }),
        topology.NewModule(topology.Config{
            Peers: []topology.Peer{{Name: "auth", Namespace: "auth"}},
        }),
    },
})
```

`server.Config` composes gateway + tracing + probes + audit + plugins automatically; anything beyond that (workflow, schedules, mcp, topology, discovery, harness) goes into `Extra` and is appended to the module slice in order.
