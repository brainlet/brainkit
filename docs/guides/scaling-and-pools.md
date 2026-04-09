# Scaling

brainkit scales horizontally by running multiple Kit replicas on the same transport. The transport's consumer group mechanism distributes messages across replicas. No load balancer needed — the bus handles it.

## Identity Hierarchy

Every brainkit deployment has four identity levels:

```
Cluster     (logical group — configured)
  └── Runtime    (physical process — auto UUID)
       └── Kit        (namespace — configured)
            └── .ts        (deployment — a service)
```

| Level | k8s Analogy | How Set | Scope |
|-------|-------------|---------|-------|
| Cluster | Cluster | `Config.ClusterID` (default: `"default"`) | Runtimes that discover each other |
| Runtime | Node | Auto UUID per process (`brainkit.RuntimeID()`) | All Kits in the same Go process |
| Kit | Namespace | `Config.Namespace` (default: `"user"`) | Own deployments, schedules, tools |
| .ts | Pod | `kit.deploy` bus command | A TypeScript service |

Every published message carries `clusterID`, `runtimeID`, `namespace`, and `callerID` in its metadata. Receivers can tell where a message came from — same process, same cluster, or remote.

## Consumer Groups = Namespace

The transport's consumer group is derived from the Kit's namespace. All Kit instances with the **same namespace** on the same transport **compete** for messages (round-robin). Different namespaces are independent.

```
           NATS / Redis / AMQP
          /        |          \
     Kit "agents"  Kit "agents"  Kit "agents"    ← same namespace = competing
     (machine 1)   (machine 2)   (machine 3)

     Kit "gateway"                               ← different namespace = independent
     (machine 1)
```

This happens automatically — no configuration beyond setting `Namespace` on each Kit.

## Commands vs Events

The bus distinguishes two message patterns:

- **Commands** (competing): `kit.deploy`, `tools.call`, `providers.add`, etc. — ONE replica handles each. Uses the shared consumer group.
- **Events** (fan-out): `kit.deployed`, `plugin.started`, `bus.handler.failed`, etc. — ALL replicas receive. Uses a unique consumer group per instance.

Each transport creates two subscribers:
- `Subscriber` — shared consumer group (namespace). For commands.
- `FanOutSubscriber` — unique group per instance. For events.

## Deployment Propagation

When Kit-1 deploys `support-agent.ts`, Kit-2 and Kit-3 need it too.

```
Kit-1: receives kit.deploy command (competing → only Kit-1 handles)
       ↓ deploys locally, persists to shared KitStore
       ↓ emits kit.deployed event (fan-out → all replicas receive)

Kit-2: receives kit.deployed event
       ↓ checks RuntimeID — "not me"
       ↓ loads deployment from shared KitStore
       ↓ deploys locally

Kit-3: same as Kit-2
```

Requires a shared KitStore (Postgres). In-memory or local SQLite stores don't support cross-instance propagation. Same pattern for teardown via `kit.teardown.done` event.

## Schedule Deduplication

A cron schedule should fire on exactly ONE replica. brainkit uses claim-based INSERT:

```sql
INSERT OR IGNORE INTO schedule_fires (schedule_id, fire_time, claimed_at)
VALUES (?, ?, ?)
```

First replica to INSERT wins. Others get a conflict, skip. Fire time is truncated to 100ms — fine enough for sub-second schedules, coarse enough for replica dedup.

Without a shared KitStore, every replica fires (acceptable for single-instance deployments).

## Gateway Pattern

Gateways (Telegram, Discord, Slack) don't need to scale. They're thin edge processes:

```
External Platform → Gateway → bus.publish → consumer group distributes
                                            /        |        \
                                       Kit-1      Kit-2      Kit-3
                                    (process)  (process)  (process)
```

The gateway receives external events and publishes them as bus messages. The bus distributes processing across worker replicas. Gateway itself is stateless glue — a `net/http` handler doing `sdk.Publish`.

For HA: two gateway instances behind a load balancer. NATS JetStream's `TrackMsgId` (already enabled) deduplicates if both receive the same webhook.

## InstanceManager

`InstanceManager` manages pools of Kit instances within a single Go process. Two modes:

### Sharded Mode (default)

Each instance gets a different namespace. Workload isolation — no message sharing.

```go
im := brainkit.NewInstanceManager()
im.SpawnPool("workers", brainkit.PoolConfig{
    Base: brainkit.Config{
        Namespace: "workers",
        Transport: "nats",
        NATSURL:   "nats://localhost:4222",
    },
    InitialCount: 3,
    Mode: brainkit.PoolSharded, // workers-workers-0, workers-workers-1, workers-workers-2
})
```

### Replicated Mode

All instances share the same namespace. Consumer group distributes messages. Horizontal scaling.

```go
im.SpawnPool("agents", brainkit.PoolConfig{
    Base: brainkit.Config{
        Namespace: "agents",
        Transport: "nats",
        NATSURL:   "nats://localhost:4222",
    },
    InitialCount: 3,
    Mode: brainkit.PoolReplicated, // all namespace "agents", competing consumers
})
```

### Auto-Scaling

```go
im.SpawnPool("workers", brainkit.PoolConfig{
    Base:         cfg,
    InitialCount: 3,
    Min:          1,
    Max:          10,
    Strategy:     brainkit.NewThresholdStrategy(100, 10), // up at 100 pending, down at 10
})

// Run evaluation loop
ticker := time.NewTicker(30 * time.Second)
go func() {
    for range ticker.C {
        im.EvaluateAndScale()
    }
}()
```

### Strategies

**StaticStrategy** — maintains a fixed count:
```go
brainkit.NewStaticStrategy(5) // always 5 instances
```

**ThresholdStrategy** — scales on pending message count:
```go
brainkit.NewThresholdStrategy(100, 10) // up at 100, down at 10
```

**Custom** — implement the interface:
```go
type ScalingStrategy interface {
    Evaluate(metrics transport.MetricsSnapshot, pool PoolInfo) ScalingDecision
}
```

## Cross-Kit Communication

Multiple Kits on the same transport communicate via `sdk.PublishTo`:

```go
// Kit A sends a deploy command to Kit B's namespace
sdk.PublishTo(kitA, ctx, "kit-b-namespace", sdk.KitDeployMsg{
    Source: "task.ts",
    Code:   tsCode,
})
```

Works across machines if both connect to the same transport cluster (NATS, Redis, AMQP). The `runtimeID` in message metadata tells you whether a message is local or remote.

## Memory Considerations

Each Kit instance has its own QuickJS heap (~50-80MB with Mastra bundle). A pool of 5 instances uses ~400MB.

For memory-constrained environments:
- Use shared tool registries (`SharedTools` in PoolConfig)
- Keep `Max` bounded
- Use replicated mode (fewer instances, consumer groups share load)
