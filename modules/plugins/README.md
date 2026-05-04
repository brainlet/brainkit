# modules/plugins — beta

Subprocess plugin supervisor + WebSocket control plane. Launches
configured plugin binaries, forwards bus events back over WS, and
installs the `plugin.*` lifecycle bus commands.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    bkmodule "github.com/brainlet/brainkit/module"
    pluginsmod "github.com/brainlet/brainkit/modules/plugins"
)

brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url), // required — plugins refuse memory
    Modules: []bkmodule.Module{
        pluginsmod.NewModule(pluginsmod.Config{
            Plugins: []pluginsmod.PluginConfig{{
                Name:   "metrics",
                Binary: "./bin/brainkit-plugin-metrics",
            }},
            Store: kitStore, // optional — restart-survival via KitStore
        }),
    },
})
```

## Bus commands

- `plugin.start` / `plugin.stop` / `plugin.restart` — dynamic
  lifecycle.
- `plugin.list` — running plugins + identity.
- `plugin.status` — health + restart counter.
- `plugin.manifest` — full manifest from a running plugin.

## Transport requirement

Plugins need real networking — the WS control plane binds TCP and
plugin→bus traffic flows over the external transport. The module
rejects `brainkit.Memory()` up front with a clear
`VALIDATION_ERROR`. Use `brainkit.EmbeddedNATS()` (default) or any
real transport.

## Capabilities

- Requires: `tools` module, non-memory transport,
  `brainkit.core.transport_kind`, `brainkit.core.shutdown_signal`,
  `brainkit.core.remote_client`, `brainkit.core.tool_registry`,
  `brainkit.core.tracer`, `brainkit.core.audit_recorder`,
  `brainkit.core.report_error`, `brainkit.core.plugin_checker_lease`,
  `brainkit.core.plugin_restarter_lease`, `brainkit.core.namespace`, and
  `brainkit.core.caller_id`.
- Uses when present: `brainkit.core.secret_store`,
  `brainkit.core.kit_store`, and
  `brainkit.core.lifecycle_debug_registry`.
- Provides: `brainkit.core.plugin_checker` and
  `brainkit.core.plugin_restarter` hooks to packages/secrets through scoped
  plugin checker/restarter leases.

## Runtime resources

Owns `plugins.manager`, `plugins.websocket`, subprocess plugin lifecycles,
plugin WebSocket sessions, replay timers, and plugin-owned tool leases.
When lifecycle debug is available, it registers a scoped `plugins` component
with manager stopping state, process/WebSocket counters, tool ownership, replay
timer, and hook lease counts. WebSocket debug includes listener state, serve
loop state, active handler count, ping loop count, registered connection count,
pending tool calls, subscriptions, and registered plugin tool count.

## Hot unmount

Unmounting stops managed plugin processes, closes the WebSocket control plane,
waits for accepted WebSocket handlers and heartbeat ping loops to drain, clears
replay timers, unregisters plugin-owned tools, and detaches plugin
checker/restarter hooks.

Close is retryable. If a plugin process does not drain before the close/kill
deadline, it remains tracked so a later close can observe and remove it after it
exits. If WebSocket close times out, the lazy WebSocket server handle remains
attached for retry and lifecycle inspection. Failed checker/restarter lease
detaches keep their lease handles active until a later close succeeds.
Lifecycle debug reports module closing state plus retained process, WebSocket,
tool, replay timer, and lease counters during teardown.

## Writing a plugin

See [`../../examples/plugin-author`](../../examples/plugin-author)
for a minimal standalone binary.
