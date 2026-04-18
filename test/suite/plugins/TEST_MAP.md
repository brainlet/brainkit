# plugins test map

Integration tests for `modules/plugins` — the subprocess plugin
supervisor + WS control plane. These tests build a real plugin
binary (nested modules under `internal/pluginfixtures/` and
`internal/pluginmetrics/`), boot it as a Kit subprocess, and
exercise the full round trip.

Unlike other suite domains this directory does **not** follow the
`Run(t, env)` pattern — each plugin scenario needs a dedicated
binary + fresh Kit, so tests are written as standalone `_test.go`
files. There's no `run.go`.

## Files

- `caller_test.go` — `TestPluginCallerExposesRouter` — plugin SDK
  Caller exposes the bus router object to plugin Go code.
- `cancel_test.go` — `TestPluginToolCancel` — cancel propagates
  from Kit to plugin via WS control plane.
- `metrics_plugin_test.go` — `TestMetricsPluginE2E` — internal
  metrics plugin round trip:
  - `snapshot` — snapshot bus message returns metrics
  - `audit_query` — audit.query works through the metrics plugin
  - `audit_stats` — audit.stats works through the metrics plugin
  - `health` — plugin health endpoint responds
  - `audit_prune` — audit.prune honors the retention window
- `no_module_test.go` — `TestPluginsNoModule` — Kit built without
  the plugins module still compiles and boots; plugin bus topics
  simply return `FEATURE_DISABLED`.
- `tool_call_bus_test.go` — `TestPluginToolCallViaBusEmbedded` —
  a plugin-registered tool invoked two ways:
  - `direct_executor` — in-process executor path
  - `via_bus_command` — bus `tools.call` command path
- `ws_subscribe_test.go` — `TestPluginWSSubscribe` — plugin can
  subscribe to bus topics over its WS control channel.

## Adding a test

1. Decide whether it needs a new plugin binary or can reuse one
   of the existing fixtures under
   `internal/pluginfixtures/` / `internal/pluginmetrics/`.
2. Add a new `<scenario>_test.go` file in this directory.
3. Update this file.
