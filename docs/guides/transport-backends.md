# Transport Backends

A Kit has exactly one transport. Pick it with one of the
`brainkit.Memory()` / `embeddednats.New()` / `nats.New(url)` /
`amqp.New(url)` / `redis.New(url)` constructors.

The root `brainkit` package keeps only the in-process memory transport linked.
Import backend-specific packages when a process uses embedded NATS, external
NATS, AMQP, or Redis:

- `github.com/brainlet/brainkit/transports/embeddednats`
- `github.com/brainlet/brainkit/transports/nats`
- `github.com/brainlet/brainkit/transports/amqp`
- `github.com/brainlet/brainkit/transports/redis`

The aggregate `github.com/brainlet/brainkit/transports` package still exists
for convenience, but it intentionally links every network backend.

| Backend | Constructor | Internal kind | Topic sanitizer |
|---|---|---|---|
| GoChannel | `brainkit.Memory()` (default) | `"memory"` | none |
| Embedded NATS | `embeddednats.New()` | `"embedded"` | dots → dashes |
| External NATS JetStream | `nats.New(url)` | `"nats"` | dots → dashes |
| AMQP (RabbitMQ) | `amqp.New(url)` | `"amqp"` | slashes → dashes |
| Redis Streams | `redis.New(url)` | `"redis"` | none |

Zero value for `Config.Transport` resolves to `brainkit.Memory()`.

## Memory

```go
brainkit.New(brainkit.Config{
    Namespace: "my-test",
    Transport: brainkit.Memory(),
    JSRuntime: true, // only needed for JS/TS deploy/eval tests
    FSRoot:    ".",
})
```

In-process GoChannel transport. Synchronous delivery, no disk, and no external
broker. A zero-value Kit does not start QuickJS; JS/TS tests opt into it with
`JSRuntime` or by mounting `modules/jsruntime`.

Limits: no cross-process communication, no plugins (the plugin
supervisor refuses `"memory"`), no NATS JetStream durability.

## Embedded NATS

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: embeddednats.New(),
    FSRoot:    "/var/lib/my-app",
})
```

Zero-config in-process NATS server with JetStream. Behaves like a
real NATS server but runs inside the Go process. Plugins work,
cross-Kit communication inside the same process works, every
transport feature (durable streams, ack policies, sanitizers)
matches external NATS.

JetStream stream data is persisted under
`<FSRoot>/nats-data/`. Empty `FSRoot` keeps state ephemeral.

Use `embeddednats.WithNATSName(name)` to override the durable consumer
prefix:

```go
embeddednats.New(embeddednats.WithNATSName("my-app-consumers"))
```

This is opt-in. The zero value of `TransportConfig` resolves to
`brainkit.Memory()` so importing brainkit alone stays light.

## External NATS JetStream

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: nats.New("nats://nats.example.com:4222",
        nats.WithNATSName("my-app")),
    FSRoot:    "/var/lib/my-app",
})
```

Connects to an external NATS server. Identical feature set to
embedded; pick external when multiple Kits across machines need to
share the bus.

JetStream streams are provisioned on first subscribe. The router
start waits up to 2 minutes for this to complete; if it times out
you get `*sdk.TimeoutError{Operation: "router start (NATS JetStream provisioning)"}`.

## AMQP (RabbitMQ)

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: amqp.New("amqp://guest:guest@rabbit.example.com:5672/"),
    FSRoot:    "/var/lib/my-app",
})
```

Useful when your infrastructure already runs RabbitMQ.

## Redis Streams

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: redis.New("redis://redis.example.com:6379/0"),
    FSRoot:    "/var/lib/my-app",
})
```

The Redis Streams-backed transport adapter. Useful when the rest of the
stack already runs Redis.

## Topic sanitizers

Every transport has invalid characters in its subject / routing
key / stream name rules. brainkit rewrites logical topics into
transport-legal ones automatically.

| Transport | Rule | Example |
|---|---|---|
| GoChannel / Redis | no rewrite | `tools.call` → `tools.call` |
| NATS (embedded + external) | dots → dashes | `tools.call` → `tools-call` |
| AMQP | slashes → dashes (dots preserved as routing key delimiters) | `plugin.tool.acme/x@1/echo` → `plugin.tool.acme-x@1-echo` |

Application code always speaks the logical topic. Sanitizers are
transparent.

## Picking a backend

| Scenario | Pick |
|---|---|
| Unit tests | `brainkit.Memory()` |
| Library embed, single process | `brainkit.Memory()` unless plugins/cross-process transport are needed |
| Plugins in a single process | `embeddednats.New()` |
| Multiple Kits across machines | `nats.New(url)` |
| Existing RabbitMQ | `amqp.New(url)` |
| Existing Redis | `redis.New(url)` |

Embedded NATS and external NATS are the only transports validated
against the plugin supervisor and cross-Kit flows. The others
carry the core bus surface but aren't part of the plugin /
cross-Kit matrix.

Modules receive the normalized transport kind as a capability and can refuse
configurations they cannot support. For example, `modules/plugins` rejects
`"memory"`.

## Transport matrix in tests

`test/transport/matrix_test.go` exercises every bus operation
against every backend. Container-backed backends (NATS, AMQP,
Redis) use testcontainers-go with Podman — the helper
`testutil.AllBackends(t)` returns only those that are reachable.
See [`../../test/transport/`](../../test/transport/) for the full
matrix.
