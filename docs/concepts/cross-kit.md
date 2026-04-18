# Cross-Kit Communication

Multiple Kits can sit on the same transport and route calls between
each other by namespace. This is how brainkit scales from one process
(embedded `*Kit`) to a fleet (many Kits sharing NATS, Redis, or AMQP)
without changing the call surface. Every Kit implements
`sdk.CrossNamespaceRuntime`, so any code that has a Kit handle can
reach any other Kit's namespace.

## The Model

- **Namespace.** Each Kit has one, set via `Config.Namespace`. Every
  published topic is prefixed with that namespace before hitting the
  transport. When `orchestrator` publishes `ts.report-svc.quarterly`,
  the wire name is `orchestrator.ts.report-svc.quarterly` (sanitized
  per transport).
- **Target namespace.** `brainkit.WithCallTo(name)` switches the
  prefix to a target: the request goes to `analytics-prod` but the
  reply-to is still on `orchestrator`. Responses flow back through
  the caller's inbox.
- **Peer names.** Most deployments want to call peers by a short
  logical name (`"analytics"`, `"ingest"`) rather than the full
  namespace. The `topology` module provides the name → namespace
  table; `WithCallTo` consults it automatically.

## The Minimal Flow

```go
// Kit A — target
target, _ := brainkit.New(brainkit.Config{
    Namespace: "analytics-prod",
    Transport: brainkit.NATS(natsURL),
    FSRoot:    ".",
})
defer target.Close()

// Deploy a handler that answers the quarterly report topic.
_, _ = target.Deploy(ctx, brainkit.PackageInline("report-svc", "report.ts", `
    bus.on("quarterly", (msg) => {
        msg.reply({ revenue: 1234567, quarter: msg.payload.quarter });
    });
`))

// Kit B — caller (same NATS)
caller, _ := brainkit.New(brainkit.Config{
    Namespace: "orchestrator",
    Transport: brainkit.NATS(natsURL),
    FSRoot:    ".",
    Modules: []brainkit.Module{
        topology.NewModule(topology.Config{
            Peers: []topology.Peer{
                {Name: "analytics", Namespace: "analytics-prod"},
            },
        }),
    },
})
defer caller.Close()

reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](caller, ctx,
    sdk.CustomMsg{
        Topic:   "ts.report-svc.quarterly",
        Payload: json.RawMessage(`{"quarter":"Q4"}`),
    },
    brainkit.WithCallTo("analytics"),
    brainkit.WithCallTimeout(10*time.Second),
)
```

See `examples/cross-kit/main.go` for the full runnable source — it
boots a standalone NATS server and shares it between two Kits in the
same process. `examples/multi-kit/main.go` shows the same wiring with
an embedded NATS transport per Kit when you only need name resolution
(replies over two independent embedded transports do not route).

## How `WithCallTo` Resolves

`WithCallTo(name)` passes the name through `Kit.resolveTargetNS(name)`:

1. If the `topology` module is loaded, call `topology.Resolve(name)`.
   This returns the configured namespace or an error if the name is
   unknown.
2. If the module is not loaded, `name` is used as the raw namespace.
   This is handy for ad-hoc scripts where you know the namespace
   up front.

The generated `sdk.PeersResolveMsg` round-trip
(`sdk.PeersResolveMsg{Name: "analytics"}`) is a bus-level alternative;
it runs through the same table.

```go
resp, _ := brainkit.Call[sdk.PeersResolveMsg, sdk.PeersResolveResp](
    caller, ctx, sdk.PeersResolveMsg{Name: "analytics"})
// resp.Namespace == "analytics-prod"
```

Likewise, `sdk.PeersListMsg` enumerates every name + namespace the
topology module knows about.

## Topology Module

`modules/topology` owns `peers.list`, `peers.resolve`, and the
`WithCallTo` resolution hook.

### Static peers

```go
topology.NewModule(topology.Config{
    Peers: []topology.Peer{
        {Name: "analytics", Namespace: "analytics-prod"},
        {Name: "ingest",    Namespace: "ingest-prod"},
    },
})
```

### Dynamic peers via discovery

Pair with `modules/discovery` to refresh the peer table from the bus
or from a static list broadcast via heartbeats:

```go
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

Three discovery types ship:

| Type       | Source                                             |
| ---------- | -------------------------------------------------- |
| `"static"` | Fixed list from `ModuleConfig.StaticPeers`.        |
| `"bus"`    | Heartbeat + presence over the shared transport.   |
| `""`       | Disabled (discovery is a no-op).                   |

There is no mDNS/UDP multicast. Peer presence is propagated through
the same transport the Kits already share.

## Reaching In-JS

Deployed `.ts` code has the same affordance through `bus.callTo`:

```typescript
const resp = await bus.callTo("analytics",
    "ts.report-svc.quarterly",
    { quarter: "Q4" },
    { timeoutMs: 10000 });
```

`bus.sendTo(name, topic, payload)` is the fire-and-forget variant.
Both resolve through the host Kit's topology module.

## Cross-Namespace Low-Level API

Beneath the typed layer, the SDK offers raw access:

```go
type CrossNamespaceRuntime interface {
    Runtime
    PublishRawTo(ctx, targetNamespace, topic string, payload json.RawMessage) (string, error)
    SubscribeRawTo(ctx, targetNamespace, topic string, handler func(sdk.Message)) (func(), error)
}

// And typed convenience:
pr, err := sdk.PublishTo(kit, ctx, "analytics-prod",
    sdk.CustomMsg{Topic: "ts.report-svc.quarterly", Payload: q4})
```

These are the hooks modules use when they need to publish events into
other namespaces (for example, a discovery module broadcasting
presence across the fleet). Application code should prefer
`brainkit.Call(..., WithCallTo(...))` so it picks up retries, cancel
propagation, and audit middleware for free.

## Topic Prefix Rules

| Call                                  | Wire topic (approx.)                      |
| ------------------------------------- | ----------------------------------------- |
| `Call(kit, ctx, m)` (same Kit)        | `<self>.<m.BusTopic()>`                   |
| `Call(kit, ctx, m, WithCallTo("b"))`  | `<target>.<m.BusTopic()>`                 |
| reply from handler                    | `<caller>.<caller inbox topic>`           |
| `bus.on("t")` in package `p`          | `<deploying Kit ns>.ts.p.t` subscription  |

Sanitization (dots → dashes on NATS/embedded, slashes → dashes on
AMQP) happens transparently — both publish and subscribe go through
the same function so the logical topic always round-trips.

## Transport Requirements

Cross-Kit only works if both Kits share a transport that can route
between them:

- **Memory** and **EmbeddedNATS** default configurations are per-Kit
  and cannot see each other. Two Kits booted with
  `brainkit.EmbeddedNATS()` each spin up their own NATS server — they
  do not cross. `examples/multi-kit/main.go` intentionally shows only
  the resolution step under that constraint.
- **NATS JetStream** (`brainkit.NATS(url)`) pointed at the same server
  works out of the box. `examples/cross-kit/main.go` boots a standalone
  `nats-server/v2` in-process and shares its URL.
- **AMQP** and **Redis Streams** work the same way — both Kits need to
  point at the same broker.

For single-process tests, prefer a shared external NATS. For real
deployments, use a shared broker and let both Kits join the same
cluster.

## When to Add a Second Kit

Use a second Kit when the workloads have different lifetimes,
different resource profiles, or different blast radiuses:

- **Dev vs prod** on the same NATS cluster — different namespaces
  prevent crossover traffic.
- **Tenant isolation** — one Kit per tenant with a scoped tool
  registry and provider keys.
- **Domain boundaries** — `orders`, `payments`, `notifications` as
  separate Kits that collaborate over typed bus topics.

For lower-touch splits (e.g. dev ergonomics on one workstation), a
single Kit with multiple deployed packages is cheaper — each `.ts`
package is already isolated in its own SES Compartment on a shared
heap.

## See Also

- `examples/cross-kit/main.go` — two Kits on a shared external NATS.
- `examples/multi-kit/main.go` — topology-only demo in one process.
- `modules/topology/README.md` — peer table internals.
- `modules/discovery/README.md` — heartbeat + static discovery.
- [bus-and-messaging.md](bus-and-messaging.md) — the unified topic
  model `WithCallTo` builds on.
