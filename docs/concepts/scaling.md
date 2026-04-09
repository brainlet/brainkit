# Scaling

brainkit is an agent OS. A single Kit is one namespace — like a k8s namespace. Multiple Kits form a cluster. Scaling means running more Kits on the same transport.

## Identity Hierarchy

```
Cluster        ← logical group (configured)
  └── Runtime      ← physical process (auto UUID)
       └── Kit         ← namespace (configured)
            └── .ts         ← deployment (a service)
```

Maps to k8s:

| brainkit | k8s | Scope |
|----------|-----|-------|
| Cluster | Cluster | All runtimes that discover each other |
| Runtime | Node | One Go process, one or more Kits |
| Kit | Namespace | Own deployments, tools, schedules |
| .ts | Pod | One TypeScript service |

## The Bus Is the Network

```
           Transport (NATS / Redis / AMQP)
          /           |            \
     Kit "agents"  Kit "agents"  Kit "agents"
     (machine 1)   (machine 2)   (machine 3)
        |             |             |
     QuickJS       QuickJS       QuickJS
        |             |             |
     Shared Storage (Postgres)
```

Same namespace = competing consumers. The transport distributes messages round-robin. No load balancer. No proxy. The bus IS the network.

Different namespace = independent. A Kit named `"gateway"` gets its own consumer group — no interference with `"agents"`.

## Commands vs Events

Two message patterns on the bus:

```
COMMAND (competing):                    EVENT (fan-out):

  Publisher                               Publisher
      |                                       |
      ▼                                       ▼
  Consumer Group                         All Subscribers
  ┌─────────────┐                        ┌─────┬─────┬─────┐
  │ Kit-1       │ ◄── round-robin        │Kit-1│Kit-2│Kit-3│ ◄── broadcast
  │ Kit-2       │     (one handles)      │     │     │     │     (all receive)
  │ Kit-3       │                        └─────┴─────┴─────┘
  └─────────────┘
```

- **Commands**: `kit.deploy`, `tools.call`, `schedules.create` — ONE replica handles each
- **Events**: `kit.deployed`, `plugin.started`, `bus.handler.failed` — ALL replicas receive

## Deployment Propagation

```
Kit-1                          Kit-2                          Kit-3
  │                              │                              │
  ├─ receives kit.deploy         │                              │
  │  (competing → only Kit-1)    │                              │
  │                              │                              │
  ├─ deploys locally             │                              │
  ├─ persists to shared store    │                              │
  ├─ emits kit.deployed ────────►├─ receives (fan-out)          │
  │  (fan-out → all get it)      ├─ loads from store    ────────├─ receives (fan-out)
  │                              ├─ deploys locally             ├─ loads from store
  │                              │                              ├─ deploys locally
  │                              │                              │
  ▼                              ▼                              ▼
  All three now have the same .ts deployed
```

The `kit.deployed` event carries `RuntimeID` so each Kit skips events from itself. Requires shared KitStore (Postgres).

## Schedule Deduplication

```
Schedule fires on all 3 replicas simultaneously:

Kit-1: ClaimScheduleFire("job-1", 12:00:00.100) → INSERT OK ✓ → fires
Kit-2: ClaimScheduleFire("job-1", 12:00:00.100) → CONFLICT  ✗ → skips
Kit-3: ClaimScheduleFire("job-1", 12:00:00.100) → CONFLICT  ✗ → skips
```

Claim-based INSERT with `(schedule_id, fire_time)` as primary key. First INSERT wins. Fire time truncated to 100ms for dedup precision.

## Gateway Pattern

Gateways don't scale. The bus behind them does.

```
Telegram ──► Gateway (1-2 instances)
                 │
                 ▼
            bus.publish("telegram.message", payload)
                 │
         ┌───────┼───────┐
         ▼       ▼       ▼
      Kit-1   Kit-2   Kit-3     ← consumer group distributes
```

The gateway is a thin HTTP/WebSocket receiver. It converts external events to bus messages. The heavy work (AI calls, storage, orchestration) happens in worker Kits that scale via consumer groups.

For HA: two gateways behind a load balancer. Message dedup at the transport level prevents double-processing.

## Cross-Kit Communication

```
Kit "agents"                    Kit "workers"
     │                               │
     ├── sdk.PublishTo(             │
     │     "workers",               │
     │     msg)                     │
     │         ─────────────────────►│
     │                               ├── handles command
     │                               ├── responds via replyTo
     │◄──────────────────────────────┤
     │                               │
```

Works across machines if both connect to the same transport cluster. `runtimeID` in metadata tells you local vs remote.

## Plugin Architecture

Plugins connect to their host Kit via WebSocket — no transport dependency:

```
Host Kit                              Plugin Process
  │                                      │
  ├── WS server (:random)               │
  │                                      │
  │ ◄──── WS connect ───────────────────┤
  │ ◄──── manifest {tools} ─────────────┤
  │ ────── ack ─────────────────────────►│
  │                                      │
  │ ────── tool.call {id, input} ──────►│
  │ ◄──── tool.result {id, result} ─────┤
```

Plugin binary only needs: `coder/websocket` + `google/uuid`. No Watermill, no QuickJS. A Python script with a WebSocket client could be a brainkit plugin.
