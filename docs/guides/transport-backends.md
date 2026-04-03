# Transport Backends

brainkit supports 6 Watermill transport backends. GoChannel (in-process) is the default. External transports enable multi-Kit communication and plugin subprocesses.

## The Six Backends

| Backend | Type String | Topic Sanitizer | Container | Use Case |
|---------|------------|-----------------|-----------|----------|
| GoChannel | `"memory"` | none | none | Default. Single-Kit, fastest. |
| SQLite | `"sql-sqlite"` | dots → underscores | none | Persistent bus on disk, single-node. |
| NATS JetStream | `"nats"` | dots → dashes | `nats:latest -js` | Multi-Kit, plugins, production. |
| AMQP (RabbitMQ) | `"amqp"` | slashes → dashes | `rabbitmq:management` | Existing RabbitMQ infra. |
| Redis Streams | `"redis"` | none | `redis:latest` | Existing Redis infra. |
| PostgreSQL | `"sql-postgres"` | dots → underscores | `postgres:16` | Existing Postgres infra. |

## Configuration

Transports are configured on Node via `MessagingConfig`:

```go
n, err := kit.NewNode(kit.NodeConfig{
    Kernel: kit.KernelConfig{...},
    Messaging: kit.MessagingConfig{
        Transport:   "nats",
        NATSURL:     "nats://localhost:4222",
        NATSName:    "my-app",        // durable prefix for JetStream consumers
    },
})
```

Full config:

```go
type MessagingConfig struct {
    Transport   string // "memory", "nats", "amqp", "redis", "sql-postgres", "sql-sqlite"
    NATSURL     string // "nats://localhost:4222"
    NATSName    string // durable consumer prefix
    AMQPURL     string // "amqp://guest:guest@localhost:5672/"
    RedisURL    string // "redis://localhost:6379/0"
    PostgresURL string // "postgres://user:pass@localhost:5432/brainkit?sslmode=disable"
    SQLitePath  string // "/tmp/brainkit-bus.db" or ":memory:"
}
```

## Topic Sanitizers

Each transport has characters that are invalid in its topic/subject/table naming system. The transport's sanitizer transforms logical topics before publishing and subscribing. This is automatic — user code works with logical names.

### NATS

Dots are NATS subject delimiters. All dots, slashes, @, and spaces become dashes:

```
tools.call → tools-call
ts.my-service.greet → ts-my-service-greet
plugin.tool.acme/plugin@1.0.0/echo → plugin-tool-acme-plugin-1-0-0-echo
```

### SQL (Postgres + SQLite)

Topics become table names. Dots, slashes, @, spaces become underscores:

```
tools.call → tools_call
ts.my-service.greet → ts_my_service_greet
```

### AMQP

Dots are native routing key delimiters — preserved. Slashes, @, spaces become dashes:

```
tools.call → tools.call (unchanged)
plugin.tool.acme/plugin@1.0.0/echo → plugin.tool.acme-plugin-1.0.0-echo
```

### GoChannel + Redis

No sanitization needed — accept any string.

## NATS JetStream Details

NATS uses JetStream with auto-provisioning. On first subscribe, a JetStream stream is created for the subject. This can be slow — 1-5 seconds per stream.

```go
// internal/messaging/transport.go — NATS config
publisher, err := wmnats.NewPublisher(wmnats.PublisherConfig{
    URL:               url,
    SubjectCalculator: natsSubjectCalc, // dots → dashes
    JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
})

subscriber, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
    URL:               url,
    QueueGroupPrefix:  durablePrefix,
    SubscribersCount:  1,
    CloseTimeout:      15 * time.Second,
    AckWaitTimeout:    30 * time.Second,
    SubscribeTimeout:  30 * time.Second,
    JetStream: wmnats.JetStreamConfig{
        AutoProvision: true,
        DurablePrefix: durablePrefix,
        TrackMsgId:    true,
    },
})
```

`NewNode` waits up to 2 minutes for `router.Running()` to account for JetStream stream provisioning. If it times out, you get `TimeoutError{Operation: "router start (NATS JetStream provisioning)"}`.

**NATS topic gotcha:** The original NATS JetStream hanging issue was caused by dots in stream names. The `TopicSanitizer` (dots → dashes) fixes this. If you ever bypass the sanitizer, streams with dots in their names will hang during auto-provisioning.

## Transport Matrix Testing

Every operation is tested on every backend. The test matrix in `test/transport/matrix_test.go` runs 12 operations × 6 backends = 72 subtests:

| Operation | What it tests |
|-----------|-------------|
| `tools_call` | Tool invocation round-trip |
| `tools_list` | Tool listing |
| `tools_resolve` | Tool metadata lookup |
| `fs_write_read` | File write + read round-trip |
| `fs_mkdir_list_stat_delete` | Full filesystem lifecycle |
| `agents_list_empty` | Agent registry query |
| `kit_deploy_teardown` | .ts deployment + teardown |
| `async_correlation` | CorrelationID-based response routing |
| `kit_redeploy` | Atomic teardown + deploy |
| `registry_has_list` | Provider registry queries |

Container-based backends (NATS, AMQP, Redis, Postgres) use testcontainers-go with Podman. The test helper `testutil.AllBackends(t)` returns only available backends — GoChannel and SQLite always, container backends only if Podman is running.

## Backend Readiness

Some backends need time after container start before they can process messages (SQL table creation, AMQP queue binding). The test helper `WaitForBackendReady` probes with a pub/sub round-trip:

```go
// internal/testutil/backend.go
func WaitForBackendReady(t *testing.T, transport *messaging.Transport) {
    for attempt := 0; attempt < 5; attempt++ {
        // Publish probe message → subscribe → wait for round-trip
        // Retries with increasing delay
    }
}
```

## Choosing a Backend

| Scenario | Recommended |
|----------|-------------|
| Single Kit, development | `"memory"` (default) |
| Single Kit, persistent bus | `"sql-sqlite"` |
| Multi-Kit, plugins | `"nats"` |
| Existing RabbitMQ | `"amqp"` |
| Existing Redis | `"redis"` |
| Existing Postgres | `"sql-postgres"` |
| Production, new infra | `"nats"` (best JetStream durability + competing consumers) |

NATS is the only backend tested with plugins and cross-Kit communication. The others work for single-Kit transport but haven't been validated for plugin subprocess flows.
