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
    Modules: []module.Module{
        d,
        topology.NewModule(topology.Config{Discovery: d}),
    },
})
```

## Provider types

- `"static"` — fixed peer list from `ModuleConfig.StaticPeers`.
- `"bus"` — heartbeat + presence announcements over the transport.
- `""` — disabled (module is a no-op).

## Capabilities

- Requires: `brainkit.core.namespace` and
  `brainkit.core.presence_transport` when bus discovery is active.
- Optional: `brainkit.core.lifecycle_debug_registry`.
- Provides: `discovery.provider`.

## Runtime resources

Owns the `discovery.provider` capability/resource and, in bus mode, the
presence heartbeat subscription/announcement loop.
When lifecycle debug is available, it registers a scoped `discovery` component
with provider, peer-count, subscription, heartbeat, and TTL state.

## Hot unmount

Unmounting closes the provider, stops bus presence work, unregisters the
capability, and removes the resource lease. Provider close failures keep the
provider attached so a later unmount retry can finish cleanup. Bus-mode close
is context-aware: it cancels the heartbeat/eviction loops, waits for active
loops under the unmount context, and keeps the subscription attached when that
wait times out so retry can finish the cleanup. Bus-mode providers also reject
duplicate active registration instead of starting a second heartbeat/subscription
set over the same provider.

Standalone use (without Module lifecycle): call
`discovery.NewStaticFromConfig` or `discovery.NewBus` directly.
