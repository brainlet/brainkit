# Transport Backends

A Kit has exactly one transport. Pick it with one of the
`brainkit.Memory()` / `brainkit.EmbeddedNATS()` /
`brainkit.NATS(url)` / `brainkit.AMQP(url)` / `brainkit.Redis(url)`
constructors.

| Backend | Constructor | `kit.TransportKind()` | Topic sanitizer |
|---|---|---|---|
| GoChannel | `brainkit.Memory()` | `"memory"` | none |
| Embedded NATS | `brainkit.EmbeddedNATS()` (default) | `"embedded"` | dots → dashes |
| External NATS JetStream | `brainkit.NATS(url)` | `"nats"` | dots → dashes |
| AMQP (RabbitMQ) | `brainkit.AMQP(url)` | `"amqp"` | slashes → dashes |
| Redis Streams | `brainkit.Redis(url)` | `"redis"` | none |

Zero value for `Config.Transport` resolves to
`brainkit.EmbeddedNATS()`.

## Memory

```go
brainkit.New(brainkit.Config{
    Namespace: "my-test",
    Transport: brainkit.Memory(),
    FSRoot:    ".",
})
```

In-process GoChannel transport. Synchronous delivery, no disk, no
goroutines beyond the QuickJS runtime itself. Use it for tests and
single-process demos.

Limits: no cross-process communication, no plugins (the plugin
supervisor refuses `"memory"`), no NATS JetStream durability.

## Embedded NATS

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: brainkit.EmbeddedNATS(),
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

Use `brainkit.WithNATSName(name)` to override the durable consumer
prefix:

```go
brainkit.EmbeddedNATS(brainkit.WithNATSName("my-app-consumers"))
```

This is the default — the zero value of `TransportConfig`
promotes to `EmbeddedNATS()` inside `brainkit.New`.

## External NATS JetStream

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: brainkit.NATS("nats://nats.example.com:4222",
        brainkit.WithNATSName("my-app")),
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
    Transport: brainkit.AMQP("amqp://guest:guest@rabbit.example.com:5672/"),
    FSRoot:    "/var/lib/my-app",
})
```

Useful when your infrastructure already runs RabbitMQ.

## Redis Streams

```go
brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: brainkit.Redis("redis://redis.example.com:6379/0"),
    FSRoot:    "/var/lib/my-app",
})
```

Watermill's Redis Streams driver. Useful when the rest of the
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
| Library embed, single process | `brainkit.EmbeddedNATS()` — default |
| Plugins in a single process | `brainkit.EmbeddedNATS()` |
| Multiple Kits across machines | `brainkit.NATS(url)` |
| Existing RabbitMQ | `brainkit.AMQP(url)` |
| Existing Redis | `brainkit.Redis(url)` |

Embedded NATS and external NATS are the only transports validated
against the plugin supervisor and cross-Kit flows. The others
carry the core bus surface but aren't part of the plugin /
cross-Kit matrix.

## Inspecting the running transport

```go
fmt.Println(kit.TransportKind())  // "memory" | "embedded" | "nats" | "amqp" | "redis"
```

Modules use this to refuse configurations they can't support (e.g.
`modules/plugins` rejects `"memory"`).

## Transport matrix in tests

`test/transport/matrix_test.go` exercises every bus operation
against every backend. Container-backed backends (NATS, AMQP,
Redis) use testcontainers-go with Podman — the helper
`testutil.AllBackends(t)` returns only those that are reachable.
See [`../../test/transport/`](../../test/transport/) for the full
matrix.
