# multi-kit

Two Kits in one process, routed by peer name through
`modules/topology`.

```sh
go run ./examples/multi-kit
```

Expected output:

```
analytics resolves to "analytics-prod"
```

## What it shows

- `topology.NewModule(topology.Config{Peers: ...})` registers a
  static peer table on the caller Kit.
- `brainkit.WithCallTo("analytics")` (not shown in this minimal
  demo) resolves `"analytics"` through the module to the target
  Kit's namespace before publishing.
- Without the topology module, `WithCallTo` treats its argument as
  the raw namespace — same codepath, different semantics.

## Scaling up

Pair topology with `modules/discovery` to learn peers from the bus
instead of a static list:

```go
d := discovery.NewModule(discovery.ModuleConfig{Type: "bus"})
topology.NewModule(topology.Config{Discovery: d})
```

Run both Kits on the same NATS cluster and they'll find each other
through heartbeats.
