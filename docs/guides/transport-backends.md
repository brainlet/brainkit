# Transport Backends

A Kit has exactly one transport. Pick it with one of the
`brainkit.Memory()` / `transports.EmbeddedNATS()` /
`transports.NATS(url)` / `transports.AMQP(url)` /
`transports.Redis(url)` constructors.

The root `brainkit` package keeps only the in-process memory transport linked.
Import `github.com/brainlet/brainkit/transports` when a process uses embedded
NATS, external NATS, AMQP, or Redis.

| Backend | Constructor | Internal kind | Topic sanitizer |
|---|---|---|---|
| GoChannel | `brainkit.Memory()` (default) | `"memory"` | none |
| Embedded NATS | `transports.EmbeddedNATS()` | `"embedded"` | dots → dashes |
| External NATS JetStream | `transports.NATS(url)` | `"nats"` | dots → dashes |
| AMQP (RabbitMQ) | `transports.AMQP(url)` | `"amqp"` | slashes → dashes |
| Redis Streams | `transports.Redis(url)` | `"redis"` | none |

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
    Transport: transports.EmbeddedNATS(),
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

Use `transports.WithNATSName(name)` to override the durable consumer
prefix:

```go
transports.EmbeddedNATS(transports.WithNATSName("my-app-consumers"))
```

This is opt-in. The zero value of `TransportConfig` resolves to
`brainkit.Memory()` so importing brainkit alone stays light.

## External NATS JetStream

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: transports.NATS("nats://nats.example.com:4222",
        transports.WithNATSName("my-app")),
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
    Transport: transports.AMQP("amqp://guest:guest@rabbit.example.com:5672/"),
    FSRoot:    "/var/lib/my-app",
})
```

Useful when your infrastructure already runs RabbitMQ.

## Redis Streams

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: transports.Redis("redis://redis.example.com:6379/0"),
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
| Plugins in a single process | `transports.EmbeddedNATS()` |
| Multiple Kits across machines | `transports.NATS(url)` |
| Existing RabbitMQ | `transports.AMQP(url)` |
| Existing Redis | `transports.Redis(url)` |

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
