# Error Handling

brainkit treats errors as first-class wire data. Every typed error has
a stable string code, carries structured details, and round-trips
through the bus envelope without losing its shape. Callers use
`errors.As` to inspect specific conditions and recover programmatically
instead of parsing messages.

## Two Layers

### Typed errors (`sdk/sdkerrors/`)

14 concrete types, each implementing `sdk.BrainkitError`:

```go
type BrainkitError interface {
    error
    Code() string
    Details() map[string]any
}
```

Every type is re-exported from `github.com/brainlet/brainkit/sdk` as a
type alias so callers only need one import:

| Type                   | Code                  | Emitted when                                          |
| ---------------------- | --------------------- | ----------------------------------------------------- |
| `*NotFoundError`       | `NOT_FOUND`           | A named resource does not exist.                      |
| `*AlreadyExistsError`  | `ALREADY_EXISTS`      | A resource with that name already exists.             |
| `*ValidationError`     | `VALIDATION_ERROR`    | Input fails validation (missing field, bad value).    |
| `*TimeoutError`        | `TIMEOUT`             | A discrete operation exceeded its deadline.           |
| `*WorkspaceEscapeError`| `WORKSPACE_ESCAPE`    | An fs path tries to escape `FSRoot`.                  |
| `*NotConfiguredError`  | `NOT_CONFIGURED`      | A feature was invoked without its required config.    |
| `*TransportError`      | `TRANSPORT_ERROR`     | Watermill/NATS/Redis/AMQP/embedded backend failed.    |
| `*PersistenceError`    | `PERSISTENCE_ERROR`   | `KitStore` (SQLite/libsql/…) operation failed.        |
| `*DeployError`         | `DEPLOY_ERROR`        | A deploy stage (transpile/eval/compartment) failed.   |
| `*BridgeError`         | `BRIDGE_ERROR`        | A Go↔JS bridge function returned an error.           |
| `*CompilerError`       | `COMPILER_ERROR`      | An inline compiler (future) fails.                    |
| `*CycleDetectedError`  | `CYCLE_DETECTED`      | Depth middleware trips at MaxDepth (default 16).      |
| `*DecodeError`         | `DECODE_ERROR`        | Reply payload cannot decode into the expected type.   |
| `*BusError`            | arbitrary code        | Fallback when code does not match a typed error.      |

### Sentinels (`sdk/errors.go`)

Three classic sentinels for boolean checks with `errors.Is`:

| Sentinel                    | When                                                  |
| --------------------------- | ----------------------------------------------------- |
| `sdk.ErrNoReplyTo`          | `Reply`/`SendChunk` on an emitted (fire-and-forget) message. |
| `sdk.ErrNotReplier`         | Runtime does not implement `sdk.Replier`.             |
| `sdk.ErrNotCrossNamespace`  | `PublishTo` on a runtime without cross-Kit support.   |

## The Envelope

Every typed bus reply is serialized as:

```json
{
  "ok": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "tool \"weather\" not found",
    "details": { "resource": "tool", "name": "weather" }
  }
}
```

`sdk.EnvelopeOK(data)` and `sdk.EnvelopeErr(code, msg, details)` build
the envelope. `sdk.ToEnvelope(data, err)` is the usual helper — if
`err` is a `BrainkitError` it preserves the code + details; any other
error collapses to `INTERNAL_ERROR`.

On the caller side, `sdk.DecodeEnvelope(payload)` + `sdk.FromEnvelope`
reconstruct the typed Go error. `brainkit.Call` does this for you:
envelope unwrapping happens before the response is decoded into the
requested generic type.

## Using `errors.As`

```go
reply, err := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](
    kit, ctx, sdk.ToolCallMsg{Name: "weather", Input: weatherInput},
    brainkit.WithCallTimeout(2*time.Second),
)
if err != nil {
    var nf *sdk.NotFoundError
    var ve *sdk.ValidationError
    var te *sdk.TimeoutError
    switch {
    case errors.As(err, &nf):
        return fmt.Errorf("missing tool %q", nf.Name)
    case errors.As(err, &ve):
        return fmt.Errorf("bad input: %s (%s)", ve.Message, ve.Field)
    case errors.As(err, &te):
        return fmt.Errorf("timed out: %s", te.Operation)
    default:
        return err
    }
}
```

`errors.As` works transparently across the envelope round-trip —
the caller does not need to know whether the error originated in the
same process or came back over NATS.

## Caller-Side Errors

`brainkit.Call` / `brainkit.CallStream` surface several caller-only
errors from `internal/bus/caller`:

| Error                      | Code                  | When                                                  |
| -------------------------- | --------------------- | ----------------------------------------------------- |
| `*caller.NoDeadlineError`  | `VALIDATION_ERROR`    | No `WithCallTimeout` and no ctx deadline.             |
| `*caller.CallTimeoutError` | `CALL_TIMEOUT`        | Deadline elapsed before the terminal reply.          |
| `*caller.CallCancelledError` | `CALL_CANCELLED`    | `ctx.Done()` fired for reasons other than timeout.    |
| `*caller.DecodeError`      | `DECODE_ERROR`        | Reply payload cannot decode into `Resp`.              |
| `*caller.BufferOverflowError` | `CALL_BUFFER_OVERFLOW` | `CallStream` buffer overflows with `BufferError`. |

These behave like any typed error — they implement `BrainkitError`, so
`errors.As` works and the code surfaces in logs/audit.

## Propagation Patterns

### Handler returns a typed error

In a Go module or tool implementation, return a typed error and let
the framework envelope it:

```go
brainkit.RegisterTool(kit, "weather", brainkit.TypedTool[WeatherInput]{
    Description: "Fetch weather.",
    Execute: func(ctx context.Context, in WeatherInput) (any, error) {
        if in.City == "" {
            return nil, &sdk.ValidationError{Field: "city", Message: "city is required"}
        }
        if _, ok := lookup(in.City); !ok {
            return nil, &sdk.NotFoundError{Resource: "city", Name: in.City}
        }
        return WeatherOutput{...}, nil
    },
})
```

The caller sees `*sdk.ValidationError` or `*sdk.NotFoundError` on the
other end of a `brainkit.CallToolCall` or a `bus.call("tools.call", …)`
from JS.

### Handler throws in JS

In a `.ts` handler, a thrown `Error` becomes a `BRIDGE_ERROR` by
default. To surface a semantic code, attach a `code` field:

```typescript
bus.on("lookup", (msg) => {
    const city = msg.payload.city;
    if (!city) {
        const e = new Error("city is required");
        (e as any).code = "VALIDATION_ERROR";
        (e as any).details = { field: "city", message: "city is required" };
        throw e;
    }
    msg.reply({ city, tempC: 18 });
});
```

The bus serializer reads `err.code` + `err.details` and builds a
proper error envelope.

### `Reply` vs fire-and-forget

```go
err := sdk.Reply(rt, ctx, inbound, response)
switch {
case errors.Is(err, sdk.ErrNoReplyTo):
    // Caller used sdk.Emit — no reply was expected.
case errors.Is(err, sdk.ErrNotReplier):
    // Programming error — this runtime cannot reply.
case err != nil:
    return err // transport issue
}
```

## What Stays as `fmt.Errorf`

Not every error deserves a type. These stay as wrapped strings:

- **Marshal failures.** `json.Marshal` errors on caller types.
- **Transport init.** Dialing NATS/Redis/AMQP at startup.
- **Config validation in `brainkit.New`.** Invalid provider ID, bad
  FSRoot — surfaced once at boot, not something the bus serializes.
- **Low-level store errors** where the typed `PersistenceError` would
  obscure the underlying driver message.

The rule: if a caller would act differently on this error than on
"something failed", give it a type. If the caller just logs it, a
wrapped error is fine.

## Error Propagation at Scale

- **Cycles** — if a package publishes to a topic it is subscribed to,
  the depth middleware returns `CycleDetectedError` within ~50 ms at
  depth 16. Adjust via custom middleware if needed.
- **Timeouts** — `brainkit.Call` requires a deadline. If you pass a
  `ctx` without a deadline and no `WithCallTimeout`, you get
  `*caller.NoDeadlineError` immediately rather than waiting forever.
- **Cross-namespace** — typed errors round-trip unchanged across
  namespaces. `*sdk.NotFoundError` on Kit A looks identical on Kit B.
- **Plugin errors** — bridged through the WebSocket control plane. A
  plugin tool that returns a typed error is enveloped on the plugin
  side and unwrapped on the host side before being re-enveloped for
  the original caller.

## See Also

- `sdk/sdkerrors/errors.go` — concrete implementations.
- `sdk/envelope.go` — envelope encoding/decoding and code mapping.
- `internal/bus/caller/errors.go` — caller-side typed errors.
- [bus-and-messaging.md](bus-and-messaging.md) — where envelopes live
  on the wire.
