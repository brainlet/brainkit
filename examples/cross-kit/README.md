# cross-kit

Two Kits in one process, routed by peer name through
`modules/topology` + `brainkit.WithCallTo`. Shares a single
in-process NATS server so the example has no external
dependencies.

## Run

```sh
go run ./examples/cross-kit
```

Expected output:

```
shared NATS at nats://127.0.0.1:XXXXX
orchestrator → analytics (WithCallTo):
  {
    "quarter": "Q4",
    "revenue": 1234567
  }
```

## What it shows

- **Shared NATS**: the example boots `nats-io/nats-server/v2`
  directly and hands its URL to both Kits via
  `brainkit.NATS(url)`. That's the pragmatic shape for
  two-Kits-one-process — `brainkit.EmbeddedNATS()` is per-Kit
  and doesn't expose its URL.
- **Topology**: the caller Kit wires `topology.NewModule` with a
  static `Peer{Name: "analytics", Namespace: "analytics-prod"}`
  entry. `WithCallTo("analytics")` resolves that name through
  the module before publishing.
- **Call routing**: the caller publishes to
  `ts.report-svc.quarterly` with `WithCallTo("analytics")`. The
  target kit (`analytics-prod`) has the `report-svc` deployment
  that handles `quarterly` and replies synchronously.

## Scaling to more Kits

Add more peers to the topology table, or swap in bus-mode
discovery so peers find each other via heartbeats:

```go
d := discovery.NewModule(discovery.ModuleConfig{
    Type:      "bus",
    Heartbeat: 10 * time.Second,
    TTL:       30 * time.Second,
})
topology.NewModule(topology.Config{Discovery: d})
```

## Production shape

Production clusters use an external NATS (real containers or
hosted) and drop the embedded server:

```go
kit, _ := brainkit.New(brainkit.Config{
    Namespace: "orchestrator",
    Transport: brainkit.NATS("nats://nats-cluster.internal:4222",
        brainkit.WithNATSName("brainkit-prod")),
    Modules: []brainkit.Module{topology.NewModule(...)},
})
```
