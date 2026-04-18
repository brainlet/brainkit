# Changelog

## Unreleased

### Session 08 Bundle B — modules/harness (WIP)

Move the harness agent-orchestration layer out of `internal/harness`
into `modules/harness` as a brainkit.Module. Marked WIP — only the
`Instance` interface + frozen `Event` / `EventType` set are stable.

Added:
- `modules/harness/` — the pre-move package (bridge, config, display,
  harness, runtime, schema, types) now under `modules/`.
- `modules/harness/module.go` — `Module` wrapping the inner Harness.
  `Init` creates the Harness from `Kit.HarnessRuntime()` when the
  Kit has a JS bridge; otherwise no-ops. Reports
  `brainkit.ModuleStatusWIP`.
- `modules/harness/instance.go` — frozen `Instance` interface
  (`SendMessage / Abort / Steer / FollowUp / Subscribe /
  CurrentThread / CurrentMode / Close`) + `Event` / `EventType`
  enum with the 6 frozen values. `instanceAdapter` maps the WIP
  `*Harness` onto `Instance` so downstream consumers import the
  interface, not the struct.
- `modules/harness/README.md` — WIP banner, frozen event table,
  "not frozen" summary.
- `modules/harness/doc.go` — package overview + usage example.
- `modules/harness/module_test.go` — lifecycle assertion for
  `Init(Close)` via a Kit built with the module.
- `(*Kernel).HarnessRuntime()` / `(*Kit).HarnessRuntime() any` — kit
  side adapter implementing the harness `Runtime` interface. `any`
  in the public signature keeps `quickjs-go` out of brainkit's
  exported surface.

Removed:
- `internal/harness/` — moved verbatim to `modules/harness/`.

### Session 08 Bundle A — modules/topology

Cross-kit routing ergonomics split out into a dedicated module. The
`peers.list` / `peers.resolve` bus commands move from `modules/discovery`
into `modules/topology`; discovery is reduced to a Provider library
(presence + self-registration). `WithCallTo(name)` now consults the
topology module to map peer names onto namespaces.

Added:
- `modules/topology/` — `Module{Resolve, Peers, Namespaces}` with
  `Config{Peers []Peer, Discovery ProviderSource}`. Static peers +
  optional presence provider (usually `modules/discovery.Module`).
  Registers `peers.list` / `peers.resolve` bus commands.
- `modules/topology.ProviderSource` — narrow interface discovery
  satisfies, so topology reads the provider lazily and module init
  order doesn't matter.
- `(*brainkit.Kit).Module(name) (Module, bool)` — general lookup for
  cross-module coordination. `WithCallTo` uses it to find topology;
  any module with a duck-typed `Resolve(string) (string, error)` can
  participate.
- `topology.Peer` alias over `discovery.Peer` — single shape across
  the two modules.
- Four new tests in `test/suite/cross/topology.go`: static resolve,
  bus peers.list, no-module raw-namespace fallback, topology resolve
  error surfacing.

Changed:
- `brainkit/call.go` — `WithCallTo(name)` resolves via topology when
  the module is wired; falls back to raw namespace when absent.
- `modules/discovery/module.go` — dropped `handleList` / `handleResolve`
  and their `RegisterCommand` calls. Added `Provider()` accessor so
  topology can read the live provider without type-asserting.
- `test/suite/cross/discovery.go` — bus tests now wire
  `busDiscoveryModules()` (discovery + topology pair); the static
  `peers.list` test drives topology directly.
- `test/suite/cross/TEST_MAP.md` — documents the new topology row +
  updated discovery wiring.

Migration:

Before (session 05):
```go
brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{
        discovery.NewModule(discovery.ModuleConfig{Type: "static", StaticPeers: ...}),
    },
})
```

After:
```go
d := discovery.NewModule(discovery.ModuleConfig{Type: "bus"})
brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{
        d,
        topology.NewModule(topology.Config{
            Peers:     []topology.Peer{{Name: "analytics", Namespace: "analytics-prod"}},
            Discovery: d,
        }),
    },
})
```

### Session 07 — modules/plugins

The subprocess plugin supervisor, WebSocket endpoint, and plugin.*
lifecycle commands move out of `internal/engine` into a stand-alone
`brainkit.Module`. Core no longer knows about plugins; Kits that don't
wire the module run without a WS server or subprocess supervisor.

Added:
- `modules/plugins/` package:
  - `manager.go` — subprocess supervisor (ported verbatim from
    `internal/engine/plugin_manager.go`).
  - `ws_server.go` — plugin WebSocket endpoint (ported from
    `internal/engine/plugin_ws.go`).
  - `handlers.go` — `lifecycleDomain` with plugin.start / stop /
    restart / list / status bus handlers (ported from
    `internal/engine/handlers_plugins.go`, renamed).
  - `module.go` — `NewModule(Config{Plugins, Store})`. Init launches
    configured plugins, restores running plugins from the Store, wires
    the plugin.* bus commands (incl. plugin.manifest), attaches
    itself as both the deploy.PluginChecker (for
    `requires.plugins` gating) and engine.PluginRestarter (for
    secrets-rotation restart). Close stops plugins + WS server and
    detaches both hooks.
  - `types.go` — narrow `Store` interface (LoadRunningPlugins /
    SaveRunningPlugin / DeleteRunningPlugin / LoadInstalledPlugins).
    brainkit.KitStore satisfies it structurally.
  - `doc.go` — package overview + usage example.
- Kit surface additions to support module-owned plugins:
  - `(*Kit).Audit() *audit.Recorder` — for modules to record events
    directly (WS server emits plugin.registered / health.changed).
  - `(*Kit).SecretStore() secrets.SecretStore` — `$secret:NAME`
    resolution in plugin env.
  - `(*Kit).Store() types.KitStore` — installed-plugin lookup.
  - `(*Kit).Tools() *tools.ToolRegistry` — register plugin-backed tools.
  - `(*Kit).Tracer() *tracing.Tracer` — plugin tool spans.
  - `(*Kit).Remote() *transport.RemoteClient` — WS server's raw
    PublishRawWithMeta / SubscribeRawFanOut / ResolvedTopic paths.
  - `(*Kit).ShutdownSignal() <-chan struct{}` — restart backoff abort.
  - `(*Kit).TransportKind() string` — normalized transport type so
    modules can refuse "memory" (plugins need real networking).
  - `(*Kit).SetPluginChecker(pc)` + `(*Kit).SetPluginRestarter(r)` —
    the module installs these hooks at Init.
  - `brainkit.PluginRestarter` alias over `engine.PluginRestarter`.
- `(*Kernel)` mirror accessors for each of the above, plus
  `Kernel.pluginChecker` / `Kernel.pluginRestarter` storage fields.

Changed:
- `internal/engine/kernel.go` — `packageDeployDomain` built with a
  stable closure reading `kernel.pluginChecker`. The old Node-capturing
  `pluginCheckerImpl` is gone.
- `internal/engine/handlers_package_deploy.go` — `resolvePluginChecker`
  and `newSecretChecker` now always return non-nil checkers
  (`denyAllPluginChecker`, `denyAllSecretChecker`) when the backing
  subsystem is absent. `internal/deploy.DeployPackage` drops its
  `if plugins != nil && secrets != nil` guard — the callers provide
  denyAll fallbacks instead.
- `internal/transport/transport.go` — `Transport.Kind` records the
  normalized transport type; `Kernel.TransportKind()` + `Kit.TransportKind()`
  expose it to modules so they can refuse unsupported configurations.
- `internal/engine/node.go` — stripped all plugin fields (`plugins`,
  `pluginLifecycle`) and lifecycle methods (StartPlugin, StopPlugin,
  RestartPlugin, ListRunningPlugins, restoreRunningPlugins,
  processPluginManifest, pluginToolTopic, mustMarshalJSON).
- `internal/engine/catalog.go` — removed plugin.start / stop /
  restart / list / status / manifest nodeCommand entries; module
  registers them.
- `internal/engine/metrics.go` — plugin details no longer emitted
  from the engine snapshot.
- `internal/types/config.go` — `NodeConfig.Plugins` removed.
- `brainkit.Config.Plugins` field removed; `brainkit.PluginConfig`
  alias stays for callers that build config lists.
- `internal/audit/recorder_test.go` — switched to an in-memory test
  store (the SQLite store moved to `modules/audit/stores` last
  session; the cross-package cycle blocked the original import).

Fixed:
- Package deploy now validates `requires.secrets` even when no plugins
  module is loaded. Previously `ValidateDeps` was guarded by
  `if plugins != nil && secrets != nil`, so if the plugin checker was
  nil both plugin AND secret validation silently no-op'd. Secrets
  validation now runs against a deny-all secret checker when the
  secret store is absent, so `requires.secrets` on a no-store Kit
  fails with a clear "secret X is not set" error instead of
  deploying a broken package.
- `modules/plugins.Module.Init` / `StartPlugin` reject the memory
  transport up front (WS control plane can't bind and plugin→bus
  traffic can't flow in-process). Previously the module accepted the
  config silently and failed later at listener bind time.

Removed:
- `internal/engine/plugin_manager.go` — moved to module.
- `internal/engine/plugin_ws.go` — moved to module.
- `internal/engine/handlers_plugins.go` — moved to module.

Migration:

Before:
```go
brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url),
    Plugins: []brainkit.PluginConfig{{Name: "x", Binary: "./x"}},
})
```

After:
```go
import pluginsmod "github.com/brainlet/brainkit/modules/plugins"

brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url),
    Modules: []brainkit.Module{
        pluginsmod.NewModule(pluginsmod.Config{
            Plugins: []brainkit.PluginConfig{{Name: "x", Binary: "./x"}},
            Store:   kitStore, // optional — enables restart-survival
        }),
    },
})
```

Without the module: the plugin.* bus commands are absent, no WS
server is started, and every `requires.plugins` package deploy entry
fails with "plugin X is not running".

### Session 06 Bundle B — modules/audit

Audit log query + storage split out of core. The `Recorder` (event
producer) stays in `internal/audit` and is nil-safe without a store; the
new `modules/audit` attaches a store at Init time, registers the
audit.query / audit.stats / audit.prune bus commands, and ships two
backends (`stores.SQLite`, `stores.Postgres`). Two duplicate SQLite audit
store implementations (`internal/audit/store.go`, `internal/store/auditstore.go`)
collapse into one.

Added:
- `modules/audit/` package:
  - `module.go` — `NewModule(Config{Store, Verbose, OwnStore})`. Init
    calls `(*Kit).SetAuditStore(store)` so every subsystem's Record
    calls start persisting, applies Verbose via `SetAuditVerbosity`,
    and registers the three bus handlers.
  - `handlers.go` — `domain` wraps the module's Store. Each handler
    is nil-safe on a missing store (returns empty / false), so a Kit
    that wires the module with no store still answers the bus
    commands with empty results.
  - `types.go` — re-exports `audit.Store / Event / Query / Verbosity`
    under `audit.` (the module package name) for ergonomic callers.
  - `stores/sqlite.go` — consolidated SQLite audit store (columns
    `timestamp, type, ...` — the pre-module schema in
    `internal/audit/store.go` won out).
  - `stores/postgres.go` — Postgres audit store built on the shared
    sqlc-generated `internal/store/sqlgen/postgres` Queries. Creates
    the `audit_events` table on Open (no longer relies on KitStore
    to bring it along).
- `(*audit.Recorder).SetStore(s)` / `SetVerbosity(v)` — module-facing
  hooks to swap the store and verbosity post-construction.
- `(*Kit).SetAuditStore(s)` / `(*Kit).SetAuditVerbosity(v)` — Kit-level
  delegates. `brainkit.AuditStore`, `brainkit.AuditVerbosity`, and the
  `AuditVerbosityNormal/Verbose` consts are re-exported in `types.go`
  so external callers can name the argument types without importing
  `internal/audit`.
- `(*Kernel).SetAuditStore` / `SetAuditVerbosity` — engine delegates.

Changed:
- `internal/engine/kernel.go` — `Recorder` is now always created in
  `initAudit` (no store = no-op). Removed the `auditStore` field — the
  module owns the reference and the handlers read it directly.
- `internal/engine/kernel_init.go` — `initAudit` dropped its
  conditional "no AuditStore → stay nil" branch; Recorder always
  exists.
- `internal/engine/catalog.go` — audit.query / stats / prune
  nodeCommand entries removed (module registers them).
- `internal/engine/handlers_audit.go` — **deleted**; `AuditDomain`
  moved verbatim into `modules/audit/handlers.go` (renamed to
  `domain`, unexported).
- `internal/types/config.go` — dropped `KernelConfig.AuditStore` and
  `KernelConfig.AuditVerbose`. No top-level `brainkit.Config.AuditStore`
  existed, so no migration impact there.
- `test/suite/env.go` — always appends `auditmod.NewModule(Config{})`
  to Config.Modules so the audit.* bus commands are registered in
  every test; no store is wired by default (tests stay permissive,
  matching the legacy FSRoot-less behavior).

Removed:
- `internal/audit/store.go` — dead code, had no callers.
- `internal/audit/store_test.go` — tested the deleted SQLite store.
- `internal/store/auditstore.go` — replaced by
  `modules/audit/stores/sqlite.go`.
- `internal/store/auditstore_pg.go` — replaced by
  `modules/audit/stores/postgres.go`.
- `internal/store/factory.NewAuditStore` — no external callers; the
  module-builders (`stores.NewSQLite`, `stores.NewPostgres`) are the
  public surface now.
- `internal/store/store_test.go` — audit store tests + factory audit
  branch dropped (`TestFactory_SQLite` still covers KitStore).

Migration:

Before:
```go
audit, _ := store.NewSQLiteAuditStore(path) // internal/store
cfg := types.KernelConfig{AuditStore: audit}
```

After:
```go
import (
    auditmod  "github.com/brainlet/brainkit/modules/audit"
    auditstores "github.com/brainlet/brainkit/modules/audit/stores"
)

auditStore, _ := auditstores.NewSQLite(path)
kit, _ := brainkit.New(brainkit.Config{
    Modules: []brainkit.Module{
        auditmod.NewModule(auditmod.Config{Store: auditStore, OwnStore: true}),
    },
})
```

Without the module: the Recorder still runs (nil-safe), no events are
persisted, and audit.query / stats / prune bus commands are absent.

### Session 06 Bundle A — modules/schedules

Scheduling is now a stand-alone `brainkit.Module`. Persisted cron-like
schedules, the QuickJS bus.schedule / bus.unschedule bridges, and the
schedule.* bus commands all move out of the kernel. The kernel retains
the QuickJS job pump (generic Promise/microtask driver — required for
all .ts code, not just user schedules).

Added:
- `internal/types/scheduling.go` — `types.ScheduleHandler` (three-method
  interface: Schedule / Unschedule / List) dispatched to by the kernel's
  bus.schedule bridge and the schedule.* bus commands.
- `brainkit.ScheduleHandler` alias over `types.ScheduleHandler` so the
  public Kit.SetScheduleHandler signature has a nameable godoc type.
- `(*Kit).SetScheduleHandler(h)`, `(*Kit).HasCommand(topic)`,
  `(*Kit).Logger()` — module-facing hooks for attaching a scheduler,
  rejecting schedule-to-command-topic attempts, and emitting diagnostics.
- `modules/schedules/` package:
  - `scheduler.go` — Scheduler owns the live schedule set, timers, and
    fire/restore logic. Uses the Kit's ErrorHandler for persistence
    failures (same surface as the pre-module path).
  - `module.go` — `NewModule(Config{Store})` satisfies `brainkit.Module`.
    Init wires the Scheduler into the Kit, registers schedule.create /
    cancel / list bus commands, and restores persisted schedules.
    Close stops timers and detaches the handler.
  - `handlers.go` — thin bus handlers delegating to the Scheduler.
  - `types.go` — narrow `Store` interface (SaveSchedule / LoadSchedules
    / DeleteSchedule / ClaimScheduleFire). `brainkit.KitStore` satisfies
    it structurally.

Changed:
- `internal/engine/bridges_scheduling.go` — bus.schedule /
  bus.unschedule bridges now dispatch through `Kernel.scheduleHandler`.
  Without a handler (module absent), they throw NOT_CONFIGURED so .ts
  code that calls `bus.schedule(...)` errors cleanly instead of silently
  no-oping.
- `internal/engine/kernel.go` — removed `schedules map` field and
  `scheduleEntry` type; added `scheduleHandler types.ScheduleHandler`.
  `ScheduleCleanup` hook into DeploymentManager now routes through
  the attached handler (tested via testTeardownCancelsSchedules).
- `internal/engine/kernel_init.go` / `kernel_shutdown.go` — no more
  kernel-owned schedule restore or timer teardown; the module owns both.
- `internal/engine/kernel_scheduling.go` — reduced to the QuickJS job
  pump (`startJobPump` + `processScheduledJobs`). Not schedule-specific
  — drives Promise microtasks always.
- `internal/engine/catalog.go` — schedule.create / cancel / list
  nodeCommand entries gone; the module registers them.
- `internal/engine/health.go` / `metrics.go` — schedule count reads
  through the attached handler (0 when module absent).
- `test/suite/env.go` — always appends `schedulesmod.NewModule(Config)`
  to Config.Modules; passes the KitStore as Store when persistence is
  enabled so existing test assertions (restart survives, multi-replica
  dedup) keep working unchanged.
- `test/suite/persistence/schedule.go`, `store.go`, `backend_matrix.go`
  — tests that construct Kit directly (not via env.go) now include
  `schedulesmod.NewModule({Store: store})` explicitly.

Added test: `test/suite/scheduling/no_module.go` —
`testNoModuleThrowsNotConfigured` builds a Kit without the schedules
module and asserts that a .ts `bus.schedule(...)` call lands as a
not-configured error on the reply envelope.

Migration:

Before:
```go
brainkit.New(brainkit.Config{ Store: kitStore })
// bus.schedule worked automatically
```

After:
```go
import schedulesmod "github.com/brainlet/brainkit/modules/schedules"

brainkit.New(brainkit.Config{
    Store: kitStore,
    Modules: []brainkit.Module{
        schedulesmod.NewModule(schedulesmod.Config{Store: kitStore}),
    },
})
```

`KitStore` implements `schedulesmod.Store` structurally, so the two
Store references share the same value. Without the module, `.ts`
`bus.schedule(...)` calls raise NOT_CONFIGURED and the `schedule.*`
bus commands are absent.

### Session 05 Checkpoint 6 — modules/discovery (full extraction)

Last kernel module extracted. Peer discovery — static + bus — is now a
stand-alone `brainkit.Module` with its own bus commands, removing the
last `engine.Node` coupling to a module-specific field.

Added:
- `internal/transport/presence.go` — `transport.Presence` interface
  (`PublishRawGlobal`, `SubscribeRawFanOutGlobal`). Lives next to the
  implementation so the Kit surface and discovery module can both
  import it without a cycle via `brainkit` or `internal/engine`.
  `*transport.RemoteClient` satisfies it by structural match.
- `(*Kit).PresenceTransport() transport.Presence` — narrow, purpose-
  built accessor for modules that need cluster-wide pub/sub.
- `(*Kit).Namespace() string`, `(*Kit).CallerID() string` — delegate to
  Kernel; modules use these for self-identification at Init time.
- `(*Kernel).Remote() *transport.RemoteClient` — backing accessor for
  `Kit.PresenceTransport`.
- `modules/discovery/module.go` — `*Module` satisfies
  `brainkit.Module`. `NewModule(ModuleConfig{Type, StaticPeers,
  Heartbeat, TTL, Name})` builds the provider (`static` / `bus` /
  disabled). `Init(*Kit)` wires the provider, registers the self-peer
  using `Kit.CallerID()` / `Kit.Namespace()`, and installs
  `peers.list` / `peers.resolve` via `brainkit.Command`. `Close`
  tears the provider down.
- `modules/discovery.ModuleConfig` + `modules/discovery.PeerConfig` —
  module-local types replacing the former `types.DiscoveryConfig` /
  `types.PeerConfig`.

Removed:
- `brainkit.Config.Discovery`, `brainkit.DiscoveryConfig`,
  `brainkit.PeerConfig` — gone from the root package. Use
  `Modules: []brainkit.Module{discovery.NewModule(...)}` instead.
- `types.NodeConfig.Discovery`, `types.DiscoveryConfig`,
  `types.PeerConfig` — no internal equivalents remain.
- `engine.Node.discovery` field + `NewNode`'s wiring switch + `Start`'s
  provider registration + `Shutdown`'s `discovery.Close`.
- `peers.list` / `peers.resolve` `nodeCommand` entries in
  `internal/engine/catalog.go` — the module now owns them.
- `modules/discovery/aliases.go` (aliases to deleted `types` structs)
  and the module-local `PresenceTransport` interface duplicated in
  `discovery.go`.

Migration:

Before:
```go
brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url),
    Discovery: brainkit.DiscoveryConfig{
        Type: "bus", Heartbeat: time.Second, TTL: 5 * time.Second,
    },
})
```

After:
```go
import "github.com/brainlet/brainkit/modules/discovery"

brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url),
    Modules: []brainkit.Module{
        discovery.NewModule(discovery.ModuleConfig{
            Type: "bus", Heartbeat: time.Second, TTL: 5 * time.Second,
        }),
    },
})
```

For `"static"` mode, `StaticPeers` takes `[]discovery.PeerConfig`
(module-local type — previously `brainkit.PeerConfig`).

The self-peer name defaults to a per-instance UUID (matching the old
`Node.nodeID` behavior) so replicas sharing a `CallerID` stay distinct
on the bus. Override via `discovery.ModuleConfig.Name` when a stable
identifier is required.

### Session 05 Checkpoint 5 — modules/tracing

Fifth module extracted. Durable trace storage + `trace.get` / `trace.list`
bus commands move out of core. The in-memory Tracer and ring-buffer store
stay in `internal/tracing` so span creation is always available (no-op
when no store is attached).

Added:
- `modules/tracing/` package:
  - `sqlite_store.go` — moved from `internal/tracing/sqlite_store.go` via
    `git mv`. Re-exports core `Span` / `TraceQuery` / `TraceSummary` /
    `TraceStore` types so existing method signatures stay intact.
  - `module.go` — `*Module` satisfies `brainkit.Module` with
    `Config{Store}`. `Init(*Kit)` calls `(*Kit).SetTraceStore(store)`
    and registers `trace.get` / `trace.list`. `Close` shuts the store.
- `(*Tracer).SetStore(store)` / `(*Tracer).Store()` — used by modules to
  swap the tracer's durable backend at Init.
- `(*Kit).SetTraceStore(store)` + `(*Kernel).SetTraceStore(store)` —
  public entry points for the swap.

Removed:
- `internal/engine/handlers_tracing.go` (+ `TracingDomain` +
  `newTracingDomain` + `kernel.tracingDomain` field +
  `newTracingDomain(cfg.TraceStore)` wire + `trace.get` / `trace.list`
  catalog entries — all replaced by the module's registrations).
- Root `brainkit.NewSQLiteTraceStore` — use
  `modules/tracing.NewSQLiteTraceStore` instead. `brainkit.NewMemoryTraceStore`
  stays for test ergonomics.

Changed:
- `test/suite/env.go`: when `cfg.Tracing` is set, it now constructs a
  `MemoryTraceStore` AND appends `tracingmod.New({Store: store})` to
  Config.Modules so `trace.get` / `trace.list` are wired.
- `test/suite/tracing/spans.go`: `tracingEnv` helper explicitly includes
  the tracing module for its fresh Kit.

### Session 05 Checkpoint 4 — modules/probes

Fourth module extracted. Wraps the periodic probe loop as a Kit-scoped
Module with opt-in activation.

Added:
- `modules/probes/module.go` — `Config{Interval, ProbeOnRegister}`. Init
  optionally runs an initial probe sweep and, if `Interval > 0`, starts a
  ticker goroutine that calls `kit.ProbeAll()` periodically. Close stops
  the loop. `Status()` reports `ModuleStatusBeta`.
- `(*Kit).ProbeAll()` — exported so modules and on-demand callers can
  trigger a full sweep.

Removed:
- `internal/engine/kernel_probing.go` — `startPeriodicProbing()`. The
  kernel.go call site was removed in Bundle B; this deletes the
  now-unused method. Per-resource probes stay in core.

### Session 05 Checkpoint 3 — modules/workflow

Third module extracted. Wraps the 8 Mastra-style workflow bus commands
(start / startAsync / status / resume / cancel / list / runs / restart) as a
Kit-scoped Module.

Added:
- `modules/workflow/module.go` — `*Module` satisfies `brainkit.Module`;
  `New()` constructs it. `Status()` reports `brainkit.ModuleStatusBeta`.
- `modules/workflow/handlers.go` — 8 handlers re-routed through
  `(*Kit).CallJS`.
- `(*Kit).CallJS(ctx, fn, args)` + `(*Kernel).CallJS(ctx, fn, args)` —
  exported so modules can dispatch to named JS functions registered on
  `globalThis` without reaching into internal engine APIs.

Changed:
- `test/suite/env.go` always includes `workflow.New()` in every TestEnv's
  Config.Modules — the shared `env.Kit` can exercise workflow bus
  commands.
- Workflow tests that construct their own Kit (`storage.go`,
  `concurrent.go`) explicitly add `workflow.New()` to `Config.Modules`.

Removed:
- `internal/engine/handlers_workflows.go` (8 handlers + `jsonOrNull`
  helper — moved to `modules/workflow/handlers.go`).
- Workflow catalog entries in `internal/engine/catalog.go`.

Known follow-up: `test/suite/plugins/metrics_plugin_test.go` still fails
(pre-existing, from Bundle B) because the audit store is nil without
explicit wiring. The audit module in a later checkpoint owns that path.

### Session 05 Checkpoint 2 — modules/gateway

Second module extracted into `modules/`. Gateway now satisfies
`brainkit.Module` directly — no adapter layer.

Added:
- `modules/gateway/module.go` — `(*Gateway).Name / Init / Close` methods
  on the existing `Gateway` type. `Init(k *Kit)` sets the runtime and
  calls `Start`; `Close` calls `Stop`. `Status()` reports
  `brainkit.ModuleStatusStable`.
- `(*Gateway).SetRuntime(rt)` — standalone users (no Kit) set the
  runtime before `Start`. `Init` uses this internally.

Changed:
- `gateway.New(cfg)` — drops the `rt sdk.Runtime` parameter. The
  Runtime is wired via `Init` (module path) or `SetRuntime`
  (standalone path).
- `gateway/` moved to `modules/gateway/` (11 files, history preserved
  via `git mv`).
- All gateway test callers migrated:
  - `bkgw.New(k, cfg); gw.Start()` → `bkgw.New(cfg); gw.Init(k)`
  - Two route-table-only tests (`testRouteTable`,
    `testRouteReplacement`) use `SetRuntime` without Start because
    they don't serve HTTP.

### Session 05 Checkpoint 1 — modules/mcp

First module extracted from `internal/engine` into a standalone package under
`modules/`, exercising the public Module / Command surface built in the
prereq commit.

Added:
- `modules/mcp/` package:
  - `client.go` (moved from `internal/mcp/client.go`, `package mcp`,
    `New` renamed to `NewManager` to avoid collision with the module's `New`).
  - `module.go`: `*Module` satisfies `brainkit.Module`. `New(map[string]ServerConfig)` constructs it.
    `Init(*Kit)` connects every server, registers discovered tools via
    `(*Kit).RegisterRawTool`, and wires the `mcp.listTools` / `mcp.callTool`
    bus commands via `(*Kit).RegisterCommand(brainkit.Command(...))`.
  - `Status()` reports `brainkit.ModuleStatusStable`.
- Public Kit surface needed for modules:
  - `(*Kit).RegisterRawTool(RegisteredTool) error` — registers a pre-built tool.
  - `(*Kit).ReportError(err, ErrorContext)` — forwards through the Kit's
    ErrorHandler.
  - `brainkit.RegisteredTool` / `brainkit.GoFuncExecutor` / `brainkit.ErrorContext`
    type aliases.
- Transport-finalization lifecycle:
  - `engine.Kernel.StartRouter(ctx)` + `engine.Node.StartRouter(ctx)` —
    installs host bindings and starts the Watermill router. `brainkit.New`
    now sets `DeferRouterStart` on both paths and invokes `StartRouter`
    only after brainkit.Modules have Init'd, so late-registered commands
    reach the router.

Removed:
- `internal/mcp/` (moved to `modules/mcp/`).
- `internal/engine/module_mcp.go`, `handlers_mcp.go`, `module_mcp_test.go`.
- `mcp.go` in the root + the `mcpModuleAdapter` shim from the prereq commit.
- `brainkit.NewMCPModule` — callers migrate to `modules/mcp.New`.

Changed:
- `cmd/brainkit/config`, `test/suite/env.go`, `test/fixtures/runner.go`,
  `test/suite/mcp/mcp_test.go` import `modules/mcp` and build
  `mcppkg.New(...)` instead of `brainkit.NewMCPModule(...)`.

### Session 05 prereq — public Module / Command surface

Preparatory commit for the module-extraction batch (session 05). Establishes
the public contract modules outside `internal/` need to satisfy, without
moving any existing module.

Added:
- `brainkit.Module` interface — `Name() / Init(k *Kit) error / Close() error`.
  Modules outside `internal/engine` (i.e. the forthcoming `modules/*` tree)
  satisfy this. `ModuleStatus` + `StatusReporter` for CLI listing.
- `brainkit.Command[Req, Resp](handler)` + `brainkit.CommandSpec` (alias over
  the opaque engine spec). Handler takes only `(context.Context, Req)` —
  modules capture Kit / Module state via closure.
- `(*Kit).RegisterCommand(spec)` — forwards to the kernel's per-instance
  catalog. Intended for `Module.Init`.
- `engine.MakeCommand` / exported `engine.CommandSpec` — the internal
  building blocks behind `brainkit.Command`.

Changed:
- `brainkit.New` now runs a second module init pass after the kernel is
  constructed: for every `cfg.Modules` entry satisfying `brainkit.Module`,
  it calls `Init(kit)` and tracks it for `Close()`.
- `engine.NewKernel`'s module loop skips (not errors on) non-`engine.Module`
  entries — the kit-scoped path handles them.
- `brainkit.NewMCPModule` wraps the existing `engine.MCPModule` in a small
  adapter that satisfies `brainkit.Module` (no-op Init/Close passthrough) +
  exposes an `unwrapEngineModule()` escape hatch so the real engine-scoped
  module keeps flowing through `engine.NewKernel`'s legacy init path.
  Adapter is deleted when `modules/mcp` lands.

Unchanged:
- Legacy `engine.Module` (`Init(*Kernel)`) stays internal. No existing
  module is migrated in this commit.

### Session 04 — Bundle B: Config cleanup + QuickStart

Closes session 04. Trims the public `Config` struct to its essential
fields and stops auto-wiring disk persistence / periodic probes /
implicit modules from `brainkit.New`. The batteries-included path
moves to `brainkit.QuickStart`.

Added:
- `brainkit/quickstart.go` — `QuickStart(namespace, fsRoot) (*Kit, error)`
  wires embedded NATS + SQLite `kit.db` + side-effect setup under
  `fsRoot`. Tracing + audit modules attach in session 05.
- `brainkit.NewPostgresStore(dsn)` — exposes the Postgres-backed
  KitStore factory (was reachable only via the removed
  `StoreBackend`/`StoreURL` Config fields).

Changed:
- Default `Config.Transport` is now `Memory()` (GoChannel, no disk
  side-effects, no plugins) — was `EmbeddedNATS()`. Opt in to
  embedded NATS explicitly with `Transport: brainkit.EmbeddedNATS()`
  or use `brainkit.QuickStart`.
- `cmd/brainkit/config` + `test/suite/env.WithMCP` + the fixtures
  runner inject MCP servers as an explicit `brainkit.NewMCPModule`
  entry in `Config.Modules` (no implicit MCP wiring from Config).

Removed:
- `Config.Tracing` / `Config.TraceStore` / `Config.TraceSampleRate`
  — tracing module (session 05) owns the real store. Tracer defaults
  to a nil-store no-op.
- `Config.AuditVerbose` — audit module (session 05) owns verbosity.
- `Config.StoreBackend` / `Config.StoreURL` — use explicit
  `brainkit.NewSQLiteStore(path)` / `brainkit.NewPostgresStore(dsn)`
  and pass to `Config.Store`.
- `Config.MCPServers` — construct `brainkit.NewMCPModule(...)` and
  pass via `Config.Modules`.
- FSRoot-triggered auto-create of the deployment SQLite store and
  audit SQLite store in `kernel_init`. FSRoot is now strictly the
  filesystem-polyfill sandbox root.
- `kernel.startPeriodicProbing()` call + the verbose-audit metrics
  snapshot goroutine — probes + tracing modules own these.

`brainkit.New(brainkit.Config{})` no longer writes to disk or opens
external transports. It's a minimal in-memory kernel.

Tests updated:
- `test/suite/persistence/store_backend.go` uses explicit
  `brainkit.NewSQLiteStore` / `brainkit.NewPostgresStore` instead of
  the removed `StoreBackend` / `StoreURL` fields.
- `testStoreBackendSQLiteAuditViaConfig` is skipped pending the
  session 05 audit module.
- `testMultiDeployOrderAndMetadata` expects the runtime source of
  a packageName-specified deploy to be `packageName+ext(entry)`
  (matches the bundling path).

### Session 04 — Bundle A: Package as the only deployment unit

Removes the legacy `kit.deploy` / `kit.teardown` / `kit.list` /
`kit.redeploy` / `kit.deploy.file` command surface. `package.deploy`
/ `package.teardown` / `package.list` / `package.info` are now the
canonical commands. `brainkit.Package` is the single deployment unit.

Added:
- `brainkit/package.go` — public `Package` type with builders
  `PackageInline(name, entry, source)`, `PackageFromFile(path)`,
  `PackageFromDir(dir)`. `(*Kit).Deploy(ctx, pkg)`,
  `(*Kit).Teardown(ctx, name)`, `(*Kit).Get(ctx, name)`,
  `(*Kit).List(ctx)` round-trip through the shared-inbox Caller.
- `PackageDeployMsg` inline path: `Path == ""` + `Files` set skips
  esbuild bundling and deploys the entry file directly. `Manifest`
  carries `{name, entry, version, requires}`.
- `PackageDeployResp.Resources []sdk.ResourceInfo` — parity with the
  old `KitDeployResp` shape.
- `PackageDeployDomain` now emits `kit.deployed` /
  `kit.teardown.done` events + records to the audit recorder,
  absorbing the former `LifecycleDomain` side effects.

Changed:
- `testutil.Deploy` / `DeployWithOpts` / `DeployWithResources` build
  `PackageDeployMsg` internally; signatures unchanged.
- `testutil.Teardown` / `ListDeployments` switched to
  `PackageTeardownMsg` / `PackageListDeployedMsg`. `ListDeployments`
  returns `[]sdk.DeployedPackageInfo`.
- `PackageDeployDomain.List` reads from the authoritative
  `DeploymentManager.ListDeployments()` so restored-from-store
  deployments remain visible.
- `PackageDeployDomain.Teardown` is idempotent for missing names
  (returns `Removed:false` with no error) — matches prior
  `kit.teardown` semantics, fixes deploy/teardown races.
- `cmd/brainkit/cmd/list.go` uses `PackageListDeployedMsg`.
- `internal/engine/runtime/test_runtime.js` `deploy` / `deployFile`
  / per-test teardown hooks publish `package.*` topics.

Removed:
- `sdk.KitDeployMsg` / `KitDeployResp` / `KitTeardownMsg` /
  `KitTeardownResp` / `KitListMsg` / `KitListResp` /
  `KitRedeployMsg` / `KitRedeployResp` / `KitDeployFileMsg` types
  and their catalog bindings.
- `internal/engine/handlers_lifecycle.go` (`LifecycleDomain`) —
  merged into `PackageDeployDomain`.

Kept:
- `sdk.KitDeployedEvent` / `sdk.KitTeardownedEvent` — events
  describe what happened and remain the stable propagation + audit
  subscription surface.

### Session 03 — Bundle C: `.ts` bus.call / bus.callTo + JS BrainkitError lift

Closes session 03. Adds request-reply from `.ts` handlers and lifts
JS-thrown `BrainkitError` instances into their typed Go counterparts
on the error path.

Added:
- `internal/engine/runtime/bus.js` — `bus.call(topic, data, {timeoutMs})`
  and `bus.callTo(namespace, topic, data, {timeoutMs})`. `timeoutMs`
  is REQUIRED (mirrors Go's deadline rule). Returns a Promise that
  resolves with the unwrapped reply data or rejects with a
  `BrainkitError` built from the wire envelope's `ok:false` branch.
- `internal/engine/bridges_bus.go` — new `__go_brainkit_bus_call`
  bridge backs the JS calls. Publishes via the Kit's shared-inbox
  Caller; envelope unwrap happens in the Go Caller; the resulting
  raw-data bytes or Go typed error are delivered back to JS as a
  resolved/rejected Promise.
- `internal/contract/contract.go:JSBridgeBusCall` constant
- `internal/engine/runtime/kit.d.ts` — typed `call<T>` / `callTo<T>`
  declarations with timeoutMs required
- `bus.js:subscribe` — user handler wrapper captures thrown
  `BrainkitError` into `globalThis.__pending_handler_err` (sync OR
  async rejection) so Go-side dispatch can surface the typed code
- `internal/engine/bridges_util.go:enrichHandlerErr` — reads the
  pending slot, synthesizes an envelope, decodes via
  `sdk.FromEnvelope`, and returns the matching typed Go error. Called
  from `bridges_bus.go` on both sync and async handler exceptions.
- `test/suite/bus/ts_call.go` — 6 tests: .ts→.ts happy path,
  timeoutMs-required rejection, remote BrainkitError propagation,
  timeout-elapsed CALL_TIMEOUT surfacing, Go→.ts round-trip, and
  Go→.ts-throw typed-error surface through envelope unwrap

Changed:
- `kit_runtime.js` — `scopedBus` exposes `call`/`callTo`; wrapped
  under `rewrapErrorsAsync` so Compartment code throws the local
  `BrainkitError` class on ok=false

Verification:
- `go build ./...` clean
- `go test ./test/suite/bus` — 6 new tests green
- Full `go test ./test/suite/... -short` green except three
  pre-existing flakes (GoChannel stream interleave, Podman cross,
  plugin timing)

Session 03 is complete. All three bundles — error envelope, eval
command collapse, `.ts` bus.call — shipped. `msg.stream.end` /
`msg.stream.error` envelope wrap is deferred to the gateway SSE
rewrite; it is explicitly out of Bundle C scope per the design.

### Session 03 — Bundle B: eval command collapse

Collapses three bus eval commands into one. `kit.eval` now dispatches
on `Mode` instead of having separate topics.

Deleted:
- `sdk.KitEvalTSMsg` / `KitEvalTSResp` / topic `kit.eval-ts`
- `sdk.KitEvalModuleMsg` / `KitEvalModuleResp` / topic `kit.eval-module`
- `sdk.PublishKitEvalTS` / `SubscribeKitEvalTSResp` / `PublishKitEvalModule`
  / `SubscribeKitEvalModuleResp` from `sdk/typed_gen.go`

Changed:
- `sdk.KitEvalMsg` gains `Source` + `Mode` fields. Mode ∈
  `{"script", "ts", "module"}`; when empty, inferred from Source's file
  extension (`.ts` → `ts`, else `script`)
- `internal/engine/catalog.go` — a single `kit.eval` kernelCommand
  binding dispatches on Mode:
  - `ts` → `kernel.EvalTS(Source, Code)` (raw TS in current context)
  - `module` → `kernel.EvalModule(Source, Code)` (ES module w/ imports)
  - `script` → deploy as temp `.ts`, read `globalThis.__module_result`
  - unknown mode → `*sdkerrors.ValidationError`
- `internal/testutil/bus_helpers.go` — `EvalTSErr` / `EvalModule`
  helpers build `KitEvalMsg{Mode:"ts"}` / `KitEvalMsg{Mode:"module"}`
  (public helper signatures unchanged)
- `cmd/brainkit/cmd/eval.go` — new `--mode` and `--source` flags; help
  documents the three modes

Verification:
- `go build ./...` clean
- `grep -rn "kit\.eval-ts\|kit\.eval-module\|KitEvalTS\|KitEvalModule"`
  returns no matches outside vendor
- Full `go test ./test/suite/... -short` green except pre-existing
  `cross/node_commands/plugin_list` (Podman) and a plugin timing flake
  (`TestPluginToolCallViaBusEmbedded/via_bus_command`, ~30% repro,
  pre-existing; unrelated to this bundle)

### Session 03 — Error envelope Bundle A (closes) — ResultMeta deletion sweep

Closes Bundle A. Deletes the legacy `ResultMeta` embed + helpers from
the SDK and migrates every reader to the wire-envelope-based error
detection path.

Deleted:
- `sdk.ResultMeta` struct + `SetError` / `SetErrorWithCode` /
  `ResultError` / `ResultErrorOf` helpers (from `sdk/bus_messages.go`)
- `ResultMeta` embed from 20 response-type files covering 44 embed
  sites: agent, gateway, kit, mcp, package (x2), plugin, provider,
  registry, schedule, secret, storage, testing, tool, tracing,
  vector, workflow messages
- `test/suite/bus/error_contract.go:testResultMetaIncludesCode` +
  its run.go registration

Changed:
- `sdk/helpers.go:SubscribeTo[T]` — the migration-era flattening of
  error envelopes into `{error,code,details}` shape is gone. Error
  envelopes now invoke the handler with a zero-T; callers inspect
  the failure via the `msg sdk.Message` 2nd callback arg (e.g.
  `suite.ResponseErrorMessage(msg.Payload)` or
  `sdk.DecodeEnvelope(msg.Payload)`).
- `cmd/brainkit/cmd/helpers.go:httpBusRequest` — `ResultErrorOf`
  call replaced with `sdk.DecodeEnvelope` + `sdk.FromEnvelope` so
  CLI surfaces typed errors from the bus envelope
- `internal/engine/node.go` plugin tool-call result path — same
  envelope-based error detection
- 20 test files migrated from `resp.Error` reads to
  `suite.ResponseErrorMessage(msg.Payload)`: agents (ai, surface),
  bus (publish), deploy (surface), registry (provider, storage,
  vector), tools (registry), scheduling (bus_commands), stress
  (concurrent, concurrency), workflows (commands, concurrent,
  developer, storage, run helper), plugins (tool_call_bus,
  metrics_plugin, ws_subscribe)
- `test/suite/workflows/run.go:wfPublishAndWait` — helper now
  returns `(Resp, sdk.Message)` so tests can inspect the envelope

Verification:
- `go build ./...` clean
- Full `go test ./test/suite/... -short` green except pre-existing
  `cross/node_commands/plugin_list` (Podman infra)

Bundle A is done. Bundles B (eval command collapse) and C (`.ts`
`bus.call` / `bus.stream` / `bus.callTo`) remain.

### Session 03 — Error envelope Bundle A (JS bridges + gateway + audit + test suite)

Second drop on Bundle A. Rewires the QuickJS bridge + gateway HTTP/WS
layer + audit recorder to the wire envelope contract. Adds the
dedicated `test/suite/envelope/` regression suite.

Added:
- `test/suite/envelope/` — round-trip suite: NOT_FOUND / VALIDATION_ERROR
  typed-error decode, unknown code → `*sdkerrors.BusError` carrier,
  wire-shape invariants (success has `ok:true` + `data`, error has
  `ok:false` + `error.code`/`message`), envelope metadata flag
  presence, and `brainkit.Call` typed-error surfacing across the full
  bus round trip

Changed:
- `internal/engine/bridges_util.go:throwBrainkitError` — the thrown JS
  Error now carries real `.code` and `.details` properties. Deleted
  the `[CODE] msg {{json}}` message-string encoding.
- `internal/engine/runtime/bridges.js:__kit_parseBridgeResponse` —
  reads the wire envelope `{ok,data,error}` and throws a
  `BrainkitError(message, code, details)` on `ok=false`; success
  unwraps `data`. Retains a legacy-shape fallback so non-migrated
  producers keep working.
- `internal/engine/runtime/kit_runtime.js` — deleted `_codeRe`,
  `_detailsRe`, `_parseError`; `rewrapErrors`/`rewrapErrorsAsync`
  now rely exclusively on the JS error's `.code` property (set by
  `throwBrainkitError`) to promote into the Compartment-visible
  `BrainkitError` class
- `internal/engine/bridges_bus.go` — `__go_brainkit_bus_reply` takes
  an optional 5th `envelope` arg; when true, stamps
  `metadata["envelope"]="true"` so the Caller unwraps
- `internal/engine/runtime/bus.js` — wiring in place for envelope
  replies (`msg.reply`/`msg.send`/`msg.stream.end`/`.error`); kept
  **not yet enabled** in this drop so the many raw-decode tests
  stay green. Flip lands with session 03 Bundle B/C.
- `gateway/gateway.go` — `mapHTTPStatus` now consults the wire
  envelope's `error.code` and maps via the full taxonomy table from
  `designs/08-errors.md` (`httpCodes` map). `sanitizeErrorPayload`
  handles both envelope and legacy shapes.
- `gateway/websocket.go` — unwraps success envelopes before writing
  to the WebSocket client so clients see clean JSON; error envelopes
  forward through unchanged.
- `internal/audit/recorder.go` — new `recordErr` helper merges
  `BrainkitError.Code()` / `Details()` into event data as
  `errorCode` / `errorDetails`; `ToolCallFailed`, `DeployFailed`,
  and `BusHandlerFailed` switched to it so the audit log stays
  machine-queryable.
- `test/suite/env.go:ResponseData` — tightened envelope detection:
  requires both `ok` AND (`data` or `error`) keys before unwrapping.
  Without this, user replies like `msg.reply({ok:true,attempt:2})`
  were being falsely unwrapped to `nil`.
- `test/suite/bus/async_diag.go`, `test/suite/bus/failure.go` —
  decode `.ts` replies via `suite.ResponseData` for robustness

Pending (still remaining in session 03):
- Enable `.ts` `msg.reply`/`msg.send`/`msg.stream.end`/`.error`
  envelope wrap — requires updating ~dozen tests that raw-decode
  `.ts` handler replies
- Delete `ResultMeta` + helpers from `sdk/bus_messages.go` +
  21 `sdk/*_messages.go` embeds (mechanical)
- Eval command collapse (Bundle B)
- `.ts` `bus.call` / `bus.stream` / `bus.callTo` (Bundle C)

### Session 03 — Error envelope Bundle A (partial)

Ships the wire envelope infrastructure and migrates every affected bus
consumer in-process to the new shape. Bundle A is end-to-end on the Go
side; .ts bus.js envelope wrapping + bridges_util/bridges.js/kit_runtime.js
rewrapErrors deletion + sdk/*_messages ResultMeta deletion remain for
the follow-up bundle within session 03.

Added:
- `sdk/envelope.go` — `Envelope{Ok, Data, Error}` + `EnvelopeError{Code,
  Message, Details}` + `EnvelopeOK`/`EnvelopeErr`/`EncodeEnvelope`/
  `DecodeEnvelope`/`IsEnvelope` helpers; `FromEnvelope`/`ToEnvelope`
  map between envelopes and typed Go errors
- `sdkerrors.BusError` — generic carrier for error envelopes whose
  `code` does not map to a known typed error; implements
  `BrainkitError` so `errors.As` still works
- `test/suite` helpers `ResponseCode`, `ResponseHasError`,
  `ResponseErrorMessage`, `ResponseErrorDetails`, `ResponseData` —
  accept both envelope and legacy payload shapes

Changed:
- `internal/transport/host.go` — command replies now go out as wire
  envelopes (`{ok:true, data}` / `{ok:false, error:{code,message,details}}`)
  with `metadata["envelope"]="true"` stamped; `SerializeBrainkitError`
  builds the envelope instead of a top-level `{error,code,details}` map
- `internal/engine/kernel_failure.go` `sendErrorResponse` — also emits
  envelope replies via the new `kernel_bus.go:replyEnvelope` helper, so
  JS-handler-throw responses reach the Caller as typed errors
- `internal/bus/caller/caller.go` — when the terminal reply carries
  `envelope=true` metadata, unwraps via `sdk.FromEnvelope` and returns
  either `env.Data` as the raw success payload or the reconstructed
  typed Go error; the Bundle C "sendErrorResponse wins race over
  HandlerFailedError" known-limitation is now fixed as a side effect
- `sdk/helpers.go:SubscribeTo` — decodes envelope-carrying replies
  before unmarshaling into T; error envelopes are flattened into the
  legacy `{error, code, details}` shape so responses still embedding
  `ResultMeta` keep getting populated during the migration

Tests:
- All bus/agents/cli/deploy/secrets/persistence/security/gateway/stress/
  tools/registry/scheduling/workflows/packages/tracing/mcp/plugins/fs/
  health suites green on memory. Bus suite: 94s.
- `call_stream_all_delivered` remains flaky ~20% due to the documented
  GoChannel chunk interleaving (pre-existing, not introduced by this
  bundle). Cross suite failure is the pre-existing Podman infra issue.

Known (carried from Bundle C → fixed here):
- The Bundle C "sendErrorResponse wins race over HandlerFailedError"
  note is resolved: both paths now emit envelopes, and the Caller
  unwraps the envelope into the correct typed error.

Pending (remainder of session 03 Bundle A and B/C):
- `.ts` `bus.js` `msg.reply`/`msg.send`/`stream.end` envelope wrap
- `internal/engine/bridges_util.go` `throwBrainkitError` → real JS
  error with `.code`/`.details` properties (delete `[CODE] msg {{json}}`)
- `internal/engine/runtime/bridges.js` envelope handling in
  `__kit_parseBridgeResponse`; delete `rewrapErrors`/`_codeRe`/
  `_detailsRe`/`_parseError` from `kit_runtime.js`
- Delete `ResultMeta` + `SetError`/`SetErrorWithCode`/`ResultError`/
  `ResultErrorOf` from `sdk/bus_messages.go` + every
  `sdk/*_messages.go` embed
- `gateway/handler.go` HTTP status via envelope taxonomy table
- `internal/audit/recorder.go` structured error recording
- `test/suite/envelope/` dedicated round-trip suite
- Eval command collapse (Bundle B)
- `.ts` `bus.call`/`bus.stream`/`bus.callTo` (Bundle C)

### Session 02 — Caller Bundle C (Checkpoint 3)

Cancellation signal, fail-fast subscription, correlationID-stamped
exhausted events, and a metrics surface. Closes session 02.

Added:
- `brainkit.WithCallNoCancelSignal()` — disables the best-effort
  `_brainkit.cancel` publish that otherwise fires when `ctx` is
  cancelled before a terminal reply arrives
- `caller.CancelTopic` (`_brainkit.cancel`) + `caller.CancelNotice`
  payload type (`correlationId`, `topic`, `reason`)
- `caller.HandlerFailedError` — typed error carrying `Topic`, `Retries`,
  `Cause`; implements `BrainkitError` with code `HANDLER_FAILED`
- `Caller.Snapshot()` returning `MetricsSnapshot` with counters:
  `Inflight`, `Completed`, `TimedOut`, `Cancelled`, `Unmatched`,
  `DecodeErrs`, `BufferOverflows`, `ChunksDelivered`, `ChunksDropped`,
  `FailedFast`
- `test/suite/bus/call_cancel_failfast.go` — 4 tests: cancel notice
  on ctx timeout, `WithCallNoCancelSignal` suppresses, exhausted
  event carries correlationID metadata, metrics snapshot sanity

Changed:
- `internal/bus/caller/caller.go`
  - `NewCaller` now subscribes to `bus.handler.exhausted` in addition
    to the inbox; unsub releases both on `Close`
  - `onFailure` handler matches `msg.Metadata["correlationId"]` against
    pending calls; on hit, finalizes with `*HandlerFailedError`
  - `Call` emits `CancelNotice` on `_brainkit.cancel` when ctx closes
    before a terminal reply (detached 500ms context so already-cancelled
    parent doesn't block the emit); skipped when `NoCancelSignal` set
  - `Metrics.FailedFast` incremented on `HandlerFailedError` finalize
- `internal/engine/kernel_failure.go`
  - `emitHandlerExhausted` now takes `correlationID` and publishes via
    `k.remote.PublishRawWithMeta` so the event carries
    `metadata["correlationId"]`
  - `handleHandlerFailure` reads `correlationID` from the failing
    message's metadata and threads it through
- `internal/engine/kernel_init.go`
  - Auto-generates `watermill.NewUUID()` when `cfg.RuntimeID` is empty
    so low-level Kernel consumers that don't set it still get a Caller

Known:
- When a retry policy is configured and `sendErrorResponse` publishes
  a JSON error payload to the Caller's inbox (done=true), that success
  reply typically wins the race vs. the `bus.handler.exhausted` event.
  `HandlerFailedError` via `onFailure` remains the signal for the
  no-replyTo path; a proper typed-error contract belongs to session 03
  (error envelope).

### Session 02 — Caller Bundle B (Checkpoint 2)

Typed streaming on top of the Caller. Per-pending drain goroutine +
bounded channel + overflow policy.

Added:
- `brainkit.CallStream[Req, Chunk, Resp]` — ordered per-chunk delivery
  through `onChunk` callback, final reply decoded into Resp
- `brainkit.BufferPolicy` (re-exported from `caller.BufferPolicy`) plus
  `BufferBlock`/`BufferDropNewest`/`BufferDropOldest`/`BufferError`
- `brainkit.WithCallBuffer(n)`, `brainkit.WithCallBufferPolicy(p)`
- `caller.BufferOverflowError` — typed failure when `BufferError`
  policy triggers
- New `Metrics` fields: `BufferOverflows`, `ChunksDelivered`,
  `ChunksDropped`
- `test/suite/bus/call_stream.go` — 5 tests: all-delivered, nil-callback
  rejection, BufferError overflow, BufferDropNewest under slow
  consumer, handler-error aborts

Changed:
- `internal/bus/caller/caller.go`
  - `Config` gains `StreamHandler`, `BufferSize`, `BufferPolicy` fields
  - `pendingCall` gains streaming fields + `sendMu` serializing stream
    sends with finalize's close (no "send on closed channel" panic)
  - `onInbox` distinguishes terminal (`done=true`) from chunk; terminal
    uses `LoadAndDelete` so late chunks drop cleanly
  - Stream path: per-pending bounded channel + drain goroutine;
    `drainDone` closed on exit so `Call` waits for all chunks to flush
    before returning
  - `finalize` acquires `sendMu`; `finalizeLocked` handles the inline
    `BufferError` path (caller already holds the lock)
- `internal/transport/host.go` — command replies now set
  `done=true` metadata so the Caller finalizes immediately on bus
  command responses instead of treating them as stream chunks

Known:
- `test/suite/bus/call_stream_all_delivered` uses `assert.ElementsMatch`
  rather than order-sensitive equality. Memory transport (watermill
  GoChannel) does not serialize Publish calls by default, so rapid-fire
  stream chunks can interleave. Each chunk carries a `seq` field for
  consumers that need strict ordering. NATS/AMQP/Redis preserve FIFO
  per subject and will deliver in order on the wire.

### Session 02 — Caller Bundle A (Checkpoint 1)

Foundation for `brainkit.Call`. Shared-inbox reply router per Kit;
metadata-keyed correlation; test helpers + gateway rewritten on top of it.

Added:
- `internal/bus/caller` package
  - `Caller` — single inbox subscription per Kit
    (`_brainkit.inbox.<runtimeID>`), `sync.Map` of pending calls,
    `onInbox` demux, `Close()` finalizes all pending with
    `ErrCallerClosed`
  - `Config{TargetNamespace, Metadata}` for cross-namespace + custom meta
  - Typed errors: `NoDeadlineError`, `CallTimeoutError`,
    `CallCancelledError`, `DecodeError` (all `BrainkitError`)
  - `Metrics`/`MetricsSnapshot` with atomic counters for
    inflight/completed/timedout/cancelled/unmatched/decodeErrs
- `brainkit.Call[Req, Resp]` generic — marshals, invokes `Caller.Call`,
  unmarshals; `json.RawMessage` short-circuits decode
- `brainkit.WithCallTimeout`, `WithCallTo`, `WithCallMeta`
- `Kit.Caller()` + `Kernel.Caller()` accessors
- `test/suite/bus/call.go` — 7 tests covering happy path, deadline gate,
  timeout/cancel errors, 50× concurrent demux, raw-payload short-circuit

Changed:
- `internal/engine/kernel_init.go` — constructs Caller after transport init;
  uses `Kernel` as its `sdk.Runtime` so inbox resolves into local namespace
- `internal/engine/kernel_shutdown.go` — calls `Caller.Close()` during
  shutdown, before storages/transport teardown
- `internal/testutil/bus_helpers.go` — `roundTrip` now delegates to
  `Caller.Call` via a `callerHolder` interface check on the runtime
  (no more per-call `subscribe + publish` dance)
- `gateway/handler.go` — `handleRequest` uses `Caller.Call`; typed-error
  switch for timeout vs cancel

Out of scope (later bundles):
- Streaming + backpressure (Bundle B)
- Cancellation emit + fail-fast via `bus.handler.exhausted` (Bundle C)
- Error envelope (session 03)

### Session 01 — Phase 0 Cleanup

Pure subtraction. No new API, no behavior changes — only removal of orphaned
code from prior feature deletions.

Removed:
- `test/suite/rbac/` domain (RBAC was removed previously; the stranded test
  suite still lived in-tree)
- `internal/engine/scaling.go` — `InstanceManager`, `PoolConfig`, `PoolMode`,
  `PoolSharded`, `PoolReplicated`, `pool`, `StaticStrategy`
- `internal/types/scaling.go` — `ScalingStrategy`, `ScalingDecision`,
  `PoolInfo` types
- Scaling re-exports in root `types.go`: `InstanceManager`, `PoolConfig`,
  `StaticStrategy`, `ScalingDecision`, `ScalingStrategy`, `PoolInfo`,
  `PoolMode`, `PoolSharded`, `PoolReplicated`, `NewInstanceManager`,
  `NewStaticStrategy`
- `Kit.HealthJSON` public method and `Kernel.HealthJSON` — the `kit.health`
  bus command marshals `Kernel.Health(ctx)` inline; `gateway/health.go`
  drops its `healthJSONer` branch and always uses the `alive + ready`
  fallback on `/health`
- `test/suite/stress/scaling.go` and its 7 pool/strategy tests
- `testStorageRuntimeScalingPool` in `test/suite/registry/storage_runtime.go`
- `testHealthJSON` in `test/suite/gateway/routes.go`
- `testConcurrencyRBACAssignCheckRace`, `testTimingRoleChangeWhileHandlerRunning`,
  `testBusRateLimitExceeds`, `testErrorContractBusNotConfiguredRBAC`,
  `testRolePreservedAcrossRestart` — all were RBAC-era stubs that only `t.Skip`
- `secDeployWithRole` helper in `test/suite/security/run.go`
- `role` parameter on `testutil.DeployWithOpts`
- `rbacOnly` field on `test/suite/bus/surface.go` `cmdTest`
- `rbac.assign` / `rbac.revoke` from the forbidden-topic list in
  `test/suite/security/bus_forgery.go`
- `docs/guides/scaling-and-pools.md` guide
- `test/campaigns/fullstack/nats_postgres_rbac_test.go`
- References to the removed symbols across `docs/`, `TEST_MAP.md`,
  `CLAUDE.md` files, and `internal/docs/FEATURES.md`

Changed:
- `MetricsSnapshot` moved from `internal/types/scaling.go` into
  `internal/types/types.go` (still the same struct; only the owning file
  changed)
- `NotConfiguredError` feature strings referencing `"rbac"` now use `"mcp"`
