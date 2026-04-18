# Bus and Messaging

Every subsystem in brainkit — Go caller, deployed `.ts` handler,
plugin subprocess, HTTP gateway request — speaks to every other
subsystem by publishing messages. The bus is the only wire. A single
typed surface (`brainkit.Call`, `sdk.Publish`, `bus.call`, `bus.on`)
is exposed in Go, the SDK, and the JS runtime.

## Topic Model

A bus topic is a dotted string (`tools.call`, `ts.greeter.hello`,
`package.deploy`). Three kinds coexist:

- **Generated topics.** Every typed message in `sdk/*_messages.go`
  declares a `BusTopic()` string. The generator in
  `scripts/gen-bus-topics.go` writes `docs/bus-topics.md` from those
  declarations. 1.0-rc.1 ships ~75 topics covering
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

The Kit does not maintain a separate command catalog — the set of
topics is just the set of `BusTopic()` values plus whatever a
deployment or plugin registers at runtime.

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
`WithCallTimeout` is passed, it returns `*caller.NoDeadlineError`. This
is deliberate — nobody should wait on the bus forever.

### Generated wrappers

`call_gen.go` contains 62 type-safe shortcuts wired to the shipped
topics. They exist so a caller doesn't have to spell the types twice.

```go
resp, err := brainkit.CallKitHealth(kit, ctx, sdk.KitHealthMsg{})
route, err := brainkit.CallGatewayRouteAdd(kit, ctx, sdk.GatewayRouteAddMsg{...})
deployed, err := brainkit.CallPackageDeploy(kit, ctx, sdk.PackageDeployMsg{...})
```

Regenerate after adding a typed message with `make generate`. The
paired `sdk/typed_gen.go` file ships typed helpers usable from module
code (no Kit handle).

### Call options

```go
brainkit.WithCallTimeout(d time.Duration)        // absolute timeout
brainkit.WithCallTo(name string)                  // cross-namespace; see topology module
brainkit.WithCallMeta(map[string]string)          // extra message metadata
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
| `BufferError`      | Fail the call with `*caller.BufferOverflowError`.|

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
the runtime. Six primary methods plus three helpers:

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
bus.emit("app.ready", { at: Date.now() });

// Subscribe anywhere on the bus
const id = bus.subscribe("orders.completed", (msg) => { /* … */ });
bus.unsubscribe(id);

// Cross-namespace call (peer name or raw namespace)
await bus.callTo("analytics", "ts.report-svc.quarterly", { quarter: "Q4" },
    { timeoutMs: 10000 });

// Fire-and-forget to another service
bus.sendTo("other-service.ts", "topic", data);

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

`msg.send` emits a chunk; `msg.reply` emits the terminal reply. See
`examples/streaming/main.go` for the matched Go side.

## The Topic Catalog

`docs/bus-topics.md` is generated — it is the authoritative list of
typed topics shipped by core. Topic names in this doc are
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

Modules can register additional middlewares through
`kit.RegisterMiddleware(mw)` during `Init`.

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
- JS uses `bus.call`, `bus.on`, `bus.emit`, `bus.subscribe`,
  `bus.callTo`, `bus.sendTo`, `bus.schedule`.
- Every typed message carries its own topic via `BusTopic()`.
- Envelopes carry typed errors across the wire.
- Cross-Kit traffic flows through the same machinery with
  `WithCallTo`.
