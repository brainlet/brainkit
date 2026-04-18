# modules/discovery — beta

Peer presence + Provider abstraction. Static lists or bus-mode
heartbeats. The bus surface (`peers.list` / `peers.resolve`) lives
in `modules/topology`, which consumes a discovery Provider; pair the
two when you want cross-kit routing by peer name.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/modules/discovery"
    "github.com/brainlet/brainkit/modules/topology"
)

d := discovery.NewModule(discovery.ModuleConfig{
    Type:      "bus",
    Heartbeat: 10 * time.Second,
    TTL:       30 * time.Second,
})

brainkit.New(brainkit.Config{
    Transport: brainkit.NATS(url),
    Modules: []brainkit.Module{
        d,
        topology.NewModule(topology.Config{Discovery: d}),
    },
})
```

## Provider types

- `"static"` — fixed peer list from `ModuleConfig.StaticPeers`.
- `"bus"` — heartbeat + presence announcements over the transport.
- `""` — disabled (module is a no-op).

Standalone use (without Module lifecycle): call
`discovery.NewStaticFromConfig` or `discovery.NewBus` directly.
