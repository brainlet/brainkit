# Plugins

Plugins are separately-compiled Go binaries that extend a Kit's
capabilities at runtime. They connect to the host over localhost
WebSocket, register tools and bus subscriptions, and are
supervised by the `modules/plugins` manager (launch, restart,
shutdown, persistence).

Plugins require a non-memory transport — the WebSocket server
needs real networking and the `plugin.*` bus commands run over the
external transport the plugins share with the host. `EmbeddedNATS`
works out of the box.

See [`examples/plugin-author/`](../../examples/plugin-author/) for
a minimal plugin and
[`examples/plugin-host/`](../../examples/plugin-host/) for a live
round-trip host that builds the plugin from source.

## Plugin project layout

Plugins live in their own Go module with their own `go.mod`, so
they can ship as standalone binaries. Use `replace` directives
while developing in-tree:

```
examples/plugin-author/
├── go.mod          # own module
├── main.go         # one file per plugin is plenty
└── README.md
```

Minimal `go.mod`:

```go
module github.com/your-org/my-plugin

go 1.26

require github.com/brainlet/brainkit/sdk v1.0.0-rc.1
```

Plugins depend only on the `brainkit/sdk` tree:

- `sdk/plugin` — the author-facing framework
  (`bkplugin.New`, `bkplugin.Tool`, `bkplugin.On`).
- `sdk` — message types, the `Runtime` interface.
- `sdk/pluginws` — WebSocket wire-protocol types.

No QuickJS, no Watermill, no esbuild.

## Write a plugin

```go
package main

import (
    "context"
    "encoding/json"
    "log"

    bkplugin "github.com/brainlet/brainkit/sdk/plugin"
)

type EchoIn  struct { Text   string `json:"text"` }
type EchoOut struct { Echoed string `json:"echoed"` }

func main() {
    p := bkplugin.New("brainlet", "plugin-author", "0.1.0",
        bkplugin.WithDescription("Minimal brainkit plugin example"))

    bkplugin.Tool(p, "echo", "Echo the input text back.",
        func(_ context.Context, _ bkplugin.Client, in EchoIn) (EchoOut, error) {
            return EchoOut{Echoed: in.Text}, nil
        })

    bkplugin.On[json.RawMessage](p, "demo.events",
        func(_ context.Context, payload json.RawMessage, _ bkplugin.Client) {
            log.Printf("received demo.events: %s", string(payload))
        })

    if err := p.Run(); err != nil {
        log.Fatalf("plugin run: %v", err)
    }
}
```

Build a flat binary:

```bash
cd plugin-author
go mod tidy
go build .
```

Output: an `./plugin-author` binary with no runtime dependencies
beyond its own Go module.

## Host a plugin

Wire `modules/plugins` on the host Kit and point it at the built
binary:

```go
import (
    "github.com/brainlet/brainkit"
    pluginsmod "github.com/brainlet/brainkit/modules/plugins"
)

kit, err := brainkit.New(brainkit.Config{
    Namespace: "plugin-host-demo",
    Transport: brainkit.EmbeddedNATS(),
    FSRoot:    "/var/lib/host",
    Modules: []brainkit.Module{
        pluginsmod.NewModule(pluginsmod.Config{
            Plugins: []brainkit.PluginConfig{{
                Name:         "demo",
                Binary:       "./examples/plugin-author/plugin-author",
                AutoRestart:  false,
                StartTimeout: 15 * time.Second,
            }},
        }),
    },
})
```

`brainkit.PluginConfig` fields most users touch:

| Field | Purpose |
|---|---|
| `Name` | Logical identifier on the bus. |
| `Binary` | Path to the plugin binary. |
| `AutoRestart` | Restart with exponential backoff on exit. |
| `StartTimeout` | How long to wait for the READY line. |
| `Env` | Extra env vars passed to the plugin process. |
| `Args` | Extra CLI args. |

## Wait for registration

Plugins announce themselves on the `plugin.registered` event.
Subscribe with `sdk.SubscribeTo`:

```go
unsub, err := sdk.SubscribeTo[sdk.PluginRegisteredEvent](
    kit, ctx, "plugin.registered",
    func(evt sdk.PluginRegisteredEvent, _ sdk.Message) {
        if evt.Name == "demo" {
            fmt.Printf("plugin ready: %s/%s@%s\n",
                evt.Owner, evt.Name, evt.Version)
        }
    })
defer unsub()
```

Once the event fires, the plugin's tools appear in the Kit's tool
registry alongside Go-registered tools and are callable via
`tools.call` from every surface.

## Call a plugin tool

Same `tools.call` topic as Go tools:

```go
resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
    Name:  "echo",
    Input: map[string]any{"text": "ping"},
}, brainkit.WithCallTimeout(10*time.Second))
// resp.Result == json.RawMessage(`{"echoed":"ping"}`)
```

From `.ts`:

```ts
const r = await bus.call("tools.call", {
    name:  "echo",
    input: { text: "ping" },
}, { timeoutMs: 10000 });
```

## Lifecycle and bus commands

The `modules/plugins` manager owns:

1. **Supervision** — launch, restart with exponential backoff,
   shutdown on Kit close, optional persistence of the plugin set to
   `Config.Store` for restart recovery.
2. **WebSocket server** — the plugin dials into a localhost
   endpoint for manifest handshake, tool-call dispatch, bus
   publish / subscribe bridging, and heartbeat.
3. **`plugin.*` bus commands** — `plugin.start`, `plugin.stop`,
   `plugin.restart`, `plugin.listRunning`, `plugin.status`,
   `plugin.manifest`.

All six commands have generated Call wrappers:
`brainkit.CallPluginStart`, `CallPluginStop`, `CallPluginRestart`,
etc.

## Host environment variables

The manager injects a small fixed set into the plugin process:

| Variable | Meaning |
|---|---|
| `BRAINKIT_PLUGIN_WS_URL` | WebSocket URL to dial back to the host. |
| `BRAINKIT_NAMESPACE` | Host Kit namespace. |
| `BRAINKIT_PLUGIN_NAME` | The plugin's configured name. |
| `BRAINKIT_PLUGIN_CONFIG` | JSON string of plugin-specific config. |

Additional `Env` in `PluginConfig` is merged on top. Values of the
form `$secret:NAME` are resolved against the Kit's secret store
before the process is launched.

## Protocol outline

1. Host starts the binary with injected env.
2. Plugin dials `BRAINKIT_PLUGIN_WS_URL`.
3. Plugin sends a manifest frame listing tools and bus
   subscriptions.
4. Host registers tools, returns `manifest.ack`.
5. Plugin prints `READY:<owner>/<name>@<version>` to stdout.
6. Host reads `READY` → plugin is operational. Tools appear in
   `tools.list`; subscriptions start receiving messages.
7. Host dispatches `tool.call` frames; plugin returns
   `tool.result`. Bus publishes from the plugin traverse the WS
   connection and get rebroadcast on the host transport.
8. On SIGTERM, the plugin's `Run()` returns and the process exits.

Details live in `sdk/pluginws/` (wire protocol) and
`modules/plugins/` (host supervisor). Most plugin authors never
touch either.

## Secrets rotation

When the plugins module is wired and you call
`kit.Secrets().Rotate(ctx, name, newValue)`, the manager checks
whether any plugin's `Env` references `$secret:name` and restarts
those plugins with the refreshed value. Nothing extra to wire —
`modules/plugins` registers a restarter with the Kit during `Init`.

## Cancellation

Tool calls plumb cancellation end-to-end:

- Go caller ctx cancelled → host sends a `tool.cancel` frame to
  the plugin → plugin's `ctx` in the tool handler is cancelled.
- Plugin returns promptly; the host finalizes the call with
  `*caller.CallCancelledError`.

Long-running tools should watch `ctx.Done()` and return early.
