# modules/plugins — beta

Subprocess plugin supervisor + WebSocket control plane. Launches
configured plugin binaries, forwards bus events back over WS, and
installs the `plugin.*` lifecycle bus commands.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    pluginsmod "github.com/brainlet/brainkit/modules/plugins"
)

brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url), // required — plugins refuse memory
    Modules: []brainkit.Module{
        pluginsmod.NewModule(pluginsmod.Config{
            Plugins: []brainkit.PluginConfig{{
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

## Writing a plugin

See [`../../examples/plugin-author`](../../examples/plugin-author)
for a minimal standalone binary.
