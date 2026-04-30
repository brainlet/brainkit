# Bus and Messaging

Every subsystem in brainkit — Go caller, deployed `.ts` handler,
plugin subprocess, HTTP gateway request — speaks to every other
subsystem by publishing messages. The bus is the only wire. A single
typed surface (`brainkit.Call`, `sdk.Publish`, `bus.call`, `bus.on`)
is exposed in Go, the SDK, and the JS runtime.

## Topic Model

A bus topic is a dotted string (`tools.call`, `ts.greeter.hello`,
`package.deploy`). Three kinds coexist:

- **Generated topics.** Typed messages in `sdk/**/*_messages.go` and
  module-owned `modules/**/*_messages.go` files declare a `BusTopic()`
  string. The generator in `scripts/gen-bus-topics.go` writes
  `docs/bus-topics.md` from those declarations. 1.0-rc.1 ships ~79
  topics covering
  `package.*`, `kit.*`, `plugin.*`, `workflow.*`, `audit.*`,
  `schedules.*`, `secrets.*`, `storages.*`, `vectors.*`,
  `providers.*`, `gateway.http.*`, `peers.*`, `cluster.peers`,
  `mcp.*`, `registry.*`, `trace.*`, `tools.*`, `test.run`.
- **Deployment mailboxes.** A `.ts` package `foo` that calls
  `bus.on("bar", …)` registers the handler at `ts.foo.bar`. The prefix
  is automatic — callers address the deployment by that topic.
- **Application events.** Anything else is free-form. A workflow can
  `sdk.Emit(ctx, MyEvent{…})` where `MyEvent.BusTopic()` returns
  `orders.completed` and any subscriber can receive it.

The Kit maintains a live command catalog for request/reply routing, and
modules publish a manifest that describes the topics they own or consume:
commands, emitted events, raw subscriptions, host capabilities, and generic
non-bus resources such as tools, hooks, schedulers, processes, runtimes, and
HTTP listeners.
The generated topic list still comes from `BusTopic()` declarations, but
runtime availability comes from which modules are mounted. Deployments and
plugins can add their own topics at runtime.

The control module exposes the live module catalog and linked-code runtime
module lifecycle through `kit.modules`, `kit.module.describe`,
`kit.module.mount`, and `kit.module.unmount`. These commands use the same
registered module factories as server YAML startup, so only modules compiled
into the running binary can be mounted.

## Typed Calls from Go

Every typed topic has two access paths:

### Generic `Call`

```go
reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](
    kit, ctx,
    sdk.CustomMsg{
        Topic:   "ts.greeter.hello",
        Payload: json.RawMessage(`{"name":"world"}`),
    },
    brainkit.WithCallTimeout(2*time.Second),
)
```

The generic takes a request type that implements `sdk.BrainkitMessage`
(one method: `BusTopic() string`) and a response type. The response can
be a concrete struct or `json.RawMessage` to skip decoding.

`Call` requires a deadline. If `ctx` has no deadline and no
`WithCallTimeout` is passed, it returns `*sdk.NoDeadlineError`. This
is deliberate — nobody should wait on the bus forever.

### Generated wrappers

Generated `typed_gen.go` files contain type-safe call shortcuts wired
to the shipped topics. They exist so a caller doesn't have to spell the
types twice, and they work with any `sdk.CallerRuntime`. SDK-owned
messages generate into `sdk/typed_gen.go`; module-owned messages
generate into the module package that owns them.

```go
resp, err := healthmod.CallKitHealth(kit, ctx, healthmod.KitHealthMsg{})
route, err := gatewaymsg.CallGatewayRouteAdd(kit, ctx, gatewaymsg.GatewayRouteAddMsg{...})
deployed, err := packagemsg.CallPackageDeploy(kit, ctx, packagemsg.PackageDeployMsg{...})
```

Regenerate after adding a typed message with `make generate`. Root
`brainkit` keeps only generic `Call` / `CallStream`; module-specific
typed shortcuts live with the package that owns the message types.

### Call options

```go
brainkit.WithCallTimeout(d time.Duration)        // absolute timeout
brainkit.WithCallTo(name string)                  // cross-namespace; see topology module
brainkit.WithCallMeta(map[string]string)          // extra message metadata
sdk.WithCallTimeout(d time.Duration)              // SDK typed-call timeout
brainkit.WithCallBuffer(n int)                    // stream buffer size (CallStream only)
brainkit.WithCallBufferPolicy(BufferBlock|...)    // stream overflow policy
brainkit.WithCallNoCancelSignal()                 // suppress _brainkit.cancel on ctx cancel
```

## Streaming: `CallStream`

When a handler emits chunks before a terminal reply, use
`CallStream`:

```go
chunks := []map[string]any{}
result, err := brainkit.CallStream[sdk.CustomMsg, map[string]any, struct {
    Done  bool `json:"done"`
    Total int  `json:"total"`
}](
    kit, ctx,
    sdk.CustomMsg{Topic: "ts.streaming-demo.count", Payload: json.RawMessage(`{"n":5}`)},
    func(chunk map[string]any) error { chunks = append(chunks, chunk); return nil },
    brainkit.WithCallTimeout(5*time.Second),
)
```

Chunks arrive in publish order. Returning a non-nil error from the
chunk callback finalizes the call with that error. The JS side emits
chunks with `msg.send(data)` and the terminal reply with
`msg.reply(data)`. See `examples/streaming/main.go`.

### Buffer policies

`CallStream` buffers chunks while the callback runs. Four overflow
policies are exported:

| Policy             | Behaviour                                        |
| ------------------ | ------------------------------------------------ |
| `BufferBlock`      | Back-pressure the producer (default, 64 slots).  |
| `BufferDropNewest` | Drop incoming chunks when the buffer is full.    |
| `BufferDropOldest` | Evict the oldest queued chunk.                   |
| `BufferError`      | Fail the call with `*sdk.BufferOverflowError`.   |

## Fire-and-Forget: `sdk.Emit`

Publishing an event with no expected reply uses `sdk.Emit`:

```go
err := sdk.Emit(kit, ctx, MyEvent{Kind: "ready"})
```

`Emit` publishes to `msg.BusTopic()` without a reply-to. Subscribers
are set up with `sdk.SubscribeTo[T](rt, ctx, topic, handler)`:

```go
unsub, _ := sdk.SubscribeTo[MyEvent](kit, ctx, "app.ready",
    func(ev MyEvent, _ sdk.Message) { /* … */ })
defer unsub()
```

Unlike `Call`, subscribe-based reads never decode error envelopes as
failures — an envelope with `Ok=false` is delivered to the handler with
`T` at its zero value, and the raw envelope is available on
`msg.Payload` (decode with `sdk.DecodeEnvelope`).

## Envelopes

Typed replies cross the wire wrapped in an envelope so errors survive
the round trip:

```go
type Envelope struct {
    Ok    bool            `json:"ok"`
    Data  json.RawMessage `json:"data,omitempty"`
    Error *EnvelopeError  `json:"error,omitempty"`
}
```

`sdk.EnvelopeOK`, `sdk.EnvelopeErr`, `sdk.EncodeEnvelope`,
`sdk.DecodeEnvelope`, `sdk.FromEnvelope`, and `sdk.ToEnvelope` bracket
the pattern. `FromEnvelope` maps a typed error envelope to one of the
typed error values in `sdk/sdkerrors` (NOT_FOUND → `*NotFoundError`,
TIMEOUT → `*TimeoutError`, and 11 others). See
[error-handling.md](error-handling.md).

Inside `Call`, envelope unwrapping happens automatically: a success
envelope's `Data` is decoded into `Resp`, an error envelope becomes a
typed Go error.

## Cross-Namespace Calls

Each Kit has a namespace (`Config.Namespace`). Every published topic is
prefixed with that namespace before hitting the transport. To call
across namespaces, pass `brainkit.WithCallTo(name)`:

```go
reply, _ := brainkit.Call[sdk.CustomMsg, json.RawMessage](
    caller, ctx,
    sdk.CustomMsg{Topic: "ts.report-svc.quarterly", Payload: q4},
    brainkit.WithCallTo("analytics"),
    brainkit.WithCallTimeout(10*time.Second),
)
```

If the `topology` module is loaded, `analytics` is resolved against
its peer table; otherwise, it is used as a raw namespace. The caller's
runtime must implement `sdk.CrossNamespaceRuntime` — every
`brainkit.Kit` does. See [cross-kit.md](cross-kit.md) and
`examples/cross-kit/main.go`.

## The JS Bus API

Inside a deployed `.ts` package, `bus` is a global object endowed by
the runtime. It covers publish/subscribe, fire-and-forget service
sends, request/reply, streaming request/reply, cross-namespace calls,
and schedules:

```typescript
// Publish typed, wait for reply
const resp = await bus.call("tools.call",
    { name: "weather", input: { city: "Paris" } },
    { timeoutMs: 2000 });

// Mailbox subscribe — auto-prefixed ts.<package>.<topic>
bus.on("demo", async (msg) => {
    msg.reply({ greeting: "hi " + msg.payload.name });
});

// Fire-and-forget
bus.publish("app.ready", { at: Date.now() });

// Subscribe anywhere on the bus
const id = bus.subscribe("orders.completed", (msg) => { /* … */ });
bus.unsubscribe(id);

// Cross-namespace call (peer name or raw namespace)
await bus.callTo("analytics", "ts.report-svc.quarterly", { quarter: "Q4" },
    { timeoutMs: 10000 });

// Fire-and-forget to another service
bus.sendTo("other-service.ts", "topic", data);

// Request/reply to another service through the shared caller
await bus.callService("other-service.ts", "topic", data,
    { timeoutMs: 10000 });

// Stream chunks from another service, then await its terminal reply
const chunks: any[] = [];
const final = await bus.callServiceStream("other-service.ts", "stream", data, {
    timeoutMs: 10000,
    onChunk: (chunk) => chunks.push(chunk),
});

// Schedule a publish via the schedules module
bus.schedule(cronSpec, topic, payload);
```

Inside a handler callback, `msg` exposes both ends of a streaming
reply:

```typescript
bus.on("count", (msg) => {
    const n = msg.payload.n || 3;
    for (let i = 1; i <= n; i++) msg.send({ tick: i });
    msg.reply({ done: true, total: n });
});
```

`msg.send` emits a chunk; `msg.reply` emits the terminal reply. Go
callers consume this with `brainkit.CallStream`; TypeScript callers
use `bus.callStream`, `bus.callServiceStream`, or `bus.callToStream`.
Each stream API uses the shared caller inbox, runs `onChunk` one chunk
at a time, and resolves only after the terminal reply and queued chunks
have drained. See `examples/streaming/main.go` for the matched Go side.

## The Topic Catalog

`docs/bus-topics.md` is generated — it is the authoritative list of
typed topics shipped by SDK and modules. Topic names in this doc are
cross-references, not duplicates. Run the generator after adding a new
typed message:

```bash
go run scripts/gen-bus-topics.go
```

## Transport Sanitizers

Different transports require different characters. `RemoteClient`
applies a transport-specific sanitizer before sending:

| Transport         | Kind         | Sanitizer          |
| ----------------- | ------------ | ------------------ |
| Memory (GoChannel)| `memory`     | none               |
| Embedded NATS     | `embedded`   | dots → dashes      |
| NATS JetStream    | `nats`       | dots → dashes      |
| AMQP (RabbitMQ)   | `amqp`       | slashes → dashes   |
| Redis Streams     | `redis`      | none               |

User code only ever sees the logical topic. Wire names are an
implementation detail — the sanitizer is symmetric on publish and
subscribe.

## Middleware

Three middlewares run on every inbound message:

- **DepthMiddleware** trips `CYCLE_DETECTED` at depth 16 (default). A
  handler that publishes to a topic that re-enters itself is caught in
  under 50ms.
- **CallerIDMiddleware** stamps the calling Kit's `CallerID` (from
  `Config.CallerID`, defaulting to `Namespace`) into message metadata
  so audit + tracing can attribute traffic.
- **MetricsMiddleware** records per-topic processing time and error
  counts surfaced via `kit.Status()` and the audit module.

Module-owned bus behavior is added through scoped commands and
subscriptions on `module.Host`; the core Watermill middleware stack is
fixed at router startup.

## Topic Namespace: `ts.<pkg>.<topic>`

Deployed packages live in their own mailbox namespace. When `greeter`
calls `bus.on("hello", …)`, the bus subscription is at
`ts.greeter.hello`. The deployment's own `bus.call("tools.call", …)`
does not get the prefix — only `bus.on` mailboxes are rewritten. This
lets a package call any typed topic by name while still receiving
messages on its own dedicated prefix.

## Summary

- `brainkit.Call` is the Go front door for typed request/reply.
- `brainkit.CallStream` adds chunked replies.
- `sdk.Publish` / `sdk.Emit` + `sdk.SubscribeTo` give low-level access.
- JS uses `bus.call`, `bus.callStream`, `bus.callService`,
  `bus.callServiceStream`, `bus.callTo`, `bus.callToStream`, `bus.on`,
  `bus.publish`, `bus.emit`, `bus.subscribe`, `bus.sendTo`,
  and `bus.schedule`.
- Every typed message carries its own topic via `BusTopic()`.
- Envelopes carry typed errors across the wire.
- Cross-Kit traffic flows through the same machinery with
  `WithCallTo`.
