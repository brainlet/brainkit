# Scaling

brainkit scales the same way Kubernetes does: identical replicas
join the same consumer group on a shared transport, and the transport
distributes work. There is no load balancer, no proxy, no sharding
layer. A scaled deployment is N processes, each running `brainkit.New`
with the same `Namespace` and `Transport`.

## Identity Hierarchy

```
Cluster        ← logical group (Config.ClusterID, default "default")
  └── Runtime      ← physical process (auto-assigned UUID)
       └── Kit         ← namespace (Config.Namespace)
            └── Package    ← deployed .ts package + its bus mailboxes
```

| Layer    | Config field               | Scope                                               |
| -------- | -------------------------- | --------------------------------------------------- |
| Cluster  | `Config.ClusterID`         | All runtimes that discover each other.              |
| Runtime  | auto (`RuntimeID` in meta) | One Go process, one `*Kit`.                         |
| Kit      | `Config.Namespace`         | Tool registry, deployments, schedules, gateway.     |
| Package  | `Package.Name`             | Single deployed `.ts` — a mailbox `ts.<pkg>.<topic>`. |

`Config.CallerID` (default: `Namespace`) stamps outbound messages so
the audit module can attribute traffic. Two replicas of the same Kit
can share `CallerID` without collision because each replica gets a
unique `RuntimeID` assigned at boot.

## The Bus Is the Network

```
           Transport (NATS / Redis / AMQP)
          /           |            \
     Kit "agents"  Kit "agents"  Kit "agents"
     (replica 1)   (replica 2)   (replica 3)
        |             |             |
     QuickJS       QuickJS       QuickJS
        |             |             |
          Shared Storage (Postgres / libsql / …)
```

Replicas of the same Kit share a queue group. The transport
distributes command messages round-robin — one replica handles each
request. Different Kits (different namespaces) get independent groups,
so a `gateway` Kit and a `workers` Kit on the same NATS cluster never
interfere with each other's consumer positions.

## Commands vs Events

Two message shapes produce two distribution patterns:

| Kind      | Topic style        | Fan-out              | Use case                  |
| --------- | ------------------ | -------------------- | ------------------------- |
| Command   | `pkg.verb`         | Queue group (one)    | Request/reply work        |
| Event     | `subject.past`     | Fan-out (all)        | Broadcast state changes   |

- **Commands** such as `package.deploy`, `tools.call`, `workflow.start`
  are published with a `replyTo`. The transport hands them to exactly
  one consumer in the queue group. Replies go back on the caller's
  inbox.
- **Events** such as `kit.deployed`, `plugin.started`,
  `bus.handler.failed`, `secrets.stored`, `plugin.registered` are
  broadcast. Every Kit on the same namespace receives the event.

`sdk.Publish` requests a reply topic; `sdk.Emit` does not. The router
picks the right distribution based on that.

## Deployment Propagation

A package deployed on one replica becomes visible on every replica in
the queue group through the shared persistent store:

```
Replica-1                        Replica-2                   Replica-3
  │                                  │                          │
  ├─ receives package.deploy         │                          │
  │  (queue group → one winner)      │                          │
  │                                  │                          │
  ├─ persists package + resources    │                          │
  │  to KitStore (shared DB)         │                          │
  │                                  │                          │
  ├─ emits kit.deployed ─────────────┼─ receives (fan-out) ─────┼─ receives (fan-out)
  │                                  │                          │
  │                                  ├─ loads package body      ├─ loads package body
  │                                  │  from store              │  from store
  │                                  ├─ re-evaluates locally    ├─ re-evaluates locally
  ▼                                  ▼                          ▼
```

The `kit.deployed` event carries `RuntimeID` in its metadata so each
replica skips events it emitted itself. A shared `Config.Store` (the
`KitStore` persists deployments + schedules + secrets) is required for
this to work; without it, only the replica that accepted the
`package.deploy` knows about the new package.

## Schedule Deduplication

Every replica of a Kit runs the same schedule loop. Scheduled fires
must not duplicate work. `modules/schedules` uses claim-based
deduplication:

```
Schedule "job-1" fires on all three replicas at 12:00:00.100:

Replica-1: ClaimScheduleFire("job-1", 12:00:00.100) → INSERT OK   ✓ → fires
Replica-2: ClaimScheduleFire("job-1", 12:00:00.100) → CONFLICT    ✗ → skips
Replica-3: ClaimScheduleFire("job-1", 12:00:00.100) → CONFLICT    ✗ → skips
```

`ClaimScheduleFire(id, fireTime)` is an atomic UPSERT on
`(schedule_id, fire_time)` in the schedules table. First writer wins.
Fire times are truncated to 100 ms so clock skew between replicas
still hits the same row. The implementation lives in
`modules/schedules/scheduler.go` (invocation) and
`modules/schedules/types.go` (the `Store` contract every backend
satisfies).

## Gateway Pattern

The gateway module terminates external HTTP/SSE/WS/webhook traffic
and converts it to bus calls. Gateways themselves do not scale —
they are thin translators. The workers behind them do.

```
Telegram / HTTP / WS ──► Gateway Kit(s) (1–2 behind an LB)
                              │
                              ▼
                    bus.call("pkg.handler", …)
                              │
                      ┌───────┼───────┐
                      ▼       ▼       ▼
                   Worker   Worker   Worker   ← same namespace, queue group
```

For HA, run two gateway Kits behind a load balancer. Queue-group
semantics of the worker Kits prevent double-processing because only
one worker answers any given bus call. See `examples/streaming/main.go`
for the SSE, WebSocket, and Webhook surface.

## Cross-Kit Calls Across Machines

```
Kit "agents"                    Kit "workers"
     │                                │
     ├── brainkit.Call(               │
     │     kit, ctx, req,             │
     │     WithCallTo("workers"))     │
     │    ─────────────────────────►  │
     │                                 ├── handles command
     │                                 ├── replies via caller inbox
     │ ◄───────────────────────────────┤
     ▼                                 ▼
```

Works identically across machines as long as both Kits connect to the
same transport cluster. `RuntimeID` metadata distinguishes local vs
remote when the audit module records the call. See
[cross-kit.md](cross-kit.md) for the resolution model.

## Plugins

Plugins are separate Go binaries. Each host Kit launches its plugin
processes and keeps a WebSocket control plane per plugin (NOT the bus
transport):

```
Host Kit                              Plugin Process
  │                                        │
  ├── plugins module WS server             │
  │   (:random)                            │
  │                                        │
  │ ◄──── WS connect ──────────────────────┤
  │ ◄──── manifest {tools, subs} ──────────┤
  │ ────── ack ────────────────────────────►
  │                                        │
  │ ──── tool_call {id, input}  ───────────►
  │ ◄── tool_result {id, result}  ─────────┤
  │                                        │
```

Plugins do not join the transport's consumer group. They only see the
topics the host forwards to them (subscriptions declared in the
plugin manifest plus replies for outbound bus calls the plugin makes).
Scaling plugins means scaling the host Kit — each host owns an
independent set of plugin subprocesses.

A plugin binary links `github.com/brainlet/brainkit/sdk/plugin` only;
no Watermill, no QuickJS, no library internals. See
`examples/plugin-author/main.go` for the minimum viable plugin.

## Bottlenecks to Watch

- **Single NATS JetStream stream per durable prefix.** Each Kit's
  inbox uses one durable consumer. Throughput is capped by
  JetStream's per-consumer rate; horizontal scaling adds more
  consumers in the group.
- **Shared storage for deployments + schedules.** A SQLite store on a
  single file serializes writes; switch to Postgres/libsql for
  multi-replica workloads.
- **QuickJS per process.** Each Kit owns one JS heap. Parallelism
  inside JS is cooperative (the job pump + scheduled tasks); compute
  scales by running more Kits, not by adding JS threads.
- **AI provider rate limits.** Scaling replicas increases the
  aggregate call volume to providers. Use the tracing module and
  provider-specific retry policies (`Config.RetryPolicies`) to shape
  traffic.

## See Also

- `modules/schedules/README.md` — claim-based fire dedup.
- `modules/plugins/README.md` — WebSocket control plane + transport
  requirements.
- `modules/topology/README.md` — naming peers in a fleet.
- [cross-kit.md](cross-kit.md) — cross-namespace routing.
- [error-handling.md](error-handling.md) — how failures propagate
  through a scaled deployment.
