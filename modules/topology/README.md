# modules/topology — beta

Cross-kit routing ergonomics. Static peer table + optional
`discovery.Provider`. Owns `peers.list` / `peers.resolve` bus
commands and the `WithCallTo(name)` resolution step.

## Usage

Static peers only:

```go
topology.NewModule(topology.Config{
    Peers: []topology.Peer{
        {Name: "analytics", Namespace: "analytics-prod"},
        {Name: "ingest",    Namespace: "ingest-prod"},
    },
})
```

With bus-mode discovery:

```go
d := discovery.NewModule(discovery.ModuleConfig{Type: "bus"})
topology.NewModule(topology.Config{Discovery: d})
```

## Surface

- `peers.list` → `PeersListResp{Peers, Namespaces}` — every known
  peer + deduplicated namespace set.
- `peers.resolve` → `PeersResolveResp{Namespace}` — map a peer name
  onto its namespace; errors on unknown names.
- `Resolve(name)` / `Peers()` / `Namespaces()` — programmatic
  access without a bus round trip.

## WithCallTo integration

`brainkit.WithCallTo(name)` consults this module through
`Kit.Module("topology")`. Without the module wired, name passes
through as the literal namespace.
