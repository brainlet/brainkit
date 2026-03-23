# Pure Async Messaging Model + Streaming Formalization

**Goal:** Replace the PublishAwait synchronous-looking pattern with pure async pub/sub using reply-to addressing. Formalize streaming as a proper domain. Migrate all tests.

**Why:** PublishAwait was written for convenience but hides the async nature of the messaging system. It should never have existed. The messaging model should be pure pub/sub with explicit subscribe — the caller controls the lifecycle, not a blocking helper.

---

## 1. Core Messaging Model

### Publish (commands)

Every command publish returns a `PublishResult` containing all metadata. The `ReplyTo` field is always populated — if the caller doesn't specify one, the convention generates it.

```go
type PublishResult struct {
    MessageID     string // Watermill message UUID
    CorrelationID string // for response filtering
    ReplyTo       string // where responses will be sent (always populated for commands)
    Topic         string // where the message was published
}

type PublishOption func(*publishConfig)

// WithReplyTo overrides the auto-generated reply topic
func WithReplyTo(topic string) PublishOption

// Publish sends a typed command. Always generates a replyTo.
// Default convention: <topic>.reply.<uuid>
func Publish[T BrainkitMessage](rt Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error)
```

### Emit (events)

Events are fire-and-forget notifications. No replyTo, no response expected.

```go
// Emit sends an event. No replyTo, no response.
func Emit[T BrainkitMessage](rt Runtime, ctx context.Context, msg T) error
```

Events: `kit.deployed`, `kit.teardown.done`, `plugin.registered`, custom events via `bus.send`.

### Subscribe

```go
// SubscribeTo listens for typed messages on any topic (not derived from BusTopic).
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string, handler func(T, Message)) (cancel func(), err error)

// SubscribeRaw already exists, unchanged.
func (rt Runtime) SubscribeRaw(ctx context.Context, topic string, handler func(Message)) (cancel func(), err error)
```

### Usage Pattern

```go
// Send a command, subscribe to the reply
result, _ := sdk.Publish(rt, ctx, AiGenerateMsg{
    Model: "openai/gpt-4o-mini", Prompt: "hello",
})
cancel, _ := sdk.SubscribeTo[AiGenerateResp](rt, ctx, result.ReplyTo, func(resp AiGenerateResp, msg Message) {
    fmt.Println(resp.Text)
    cancel() // done, stop listening
})
```

No PublishAwait. No blocking. Pure async.

### Reply-To Convention

Default: `<topic>.reply.<uuid>`

Examples:
- `ai.generate` → `ai.generate.reply.a1b2c3d4`
- `tools.call` → `tools.call.reply.e5f6g7h8`
- `memory.createThread` → `memory.createThread.reply.i9j0k1l2`

Callers can override with `WithReplyTo("my-custom-topic")` for custom routing.

---

## 2. Handler Side

### Response Routing

The handler reads `replyTo` from inbound message metadata and publishes the response there. No more hardcoded `<topic>.result` convention.

```
Inbound message:
  topic: "ns.ai.generate"
  metadata:
    correlationId: "abc123"
    replyTo: "ns.ai.generate.reply.abc123"

Handler:
  1. Process request
  2. Read metadata["replyTo"]
  3. Publish response to "ns.ai.generate.reply.abc123"
     with metadata correlationId: "abc123"
```

If `replyTo` is missing from metadata (should not happen for commands), the handler logs a warning and drops the response. Commands always have replyTo.

### Host Changes

`internal/messaging/host.go` — `RegisterCommands` handler wrapper:
- Reads `replyTo` from inbound `message.Metadata`
- After handler returns, publishes result to `replyTo` topic (not `resultTopic`)
- `RawCommandBinding.ResultTopic` field removed — replaced by dynamic replyTo

### Catalog Changes

`kit/catalog.go` — `commandSpec`:
- Remove `resultTopic` field
- Remove `encodeFailure` field (errors go in the response payload via ResultMeta, published to replyTo like any response)
- Handler signature unchanged — still returns `(json.RawMessage, error)`
- Error responses still use `ResultMeta.Error` — the handler wraps errors, publishes to replyTo

---

## 3. Streaming

### ai.stream as Catalog Command

`ai.stream` is added to the command catalog. The handler:

1. Reads `replyTo` from metadata
2. Evaluates JS `embed.streamText()` via `evalDomain`
3. For each text delta: publishes a `StreamChunk` to `replyTo` with the same correlationID
4. Final chunk has `Done: true` and `Final` containing the complete response
5. No separate "final result" message — the done chunk IS the final message

### StreamChunk (unchanged)

```go
type StreamChunk struct {
    StreamID string          `json:"streamId"`
    Seq      int             `json:"seq"`
    Delta    string          `json:"delta,omitempty"`
    Done     bool            `json:"done"`
    Final    json.RawMessage `json:"final,omitempty"`
}
```

### Streaming Usage

```go
result, _ := sdk.Publish(rt, ctx, AiStreamMsg{
    Model: "openai/gpt-4o-mini", Prompt: "hello",
})
sdk.SubscribeTo[StreamChunk](rt, ctx, result.ReplyTo, func(chunk StreamChunk, msg Message) {
    if chunk.Done {
        // stream complete, chunk.Final has full response
    } else {
        fmt.Print(chunk.Delta)
    }
})
```

Same pattern from every surface. No special streaming API — it's just Publish + SubscribeTo on the reply topic.

### JS Bridge for Streaming

The `ai.stream` handler needs to bridge JS streaming to Go bus publishing. The handler:

1. Calls `embed.streamText()` in JS (returns a stream object with `textStream`)
2. Reads chunks from `textStream` via a Go goroutine polling `ctx.Schedule`
3. Each chunk is published to `replyTo` as a `StreamChunk`

This requires a new bridge pattern: JS produces chunks → Go publishes them. The handler runs async in the bridge's goroutine pool.

### WASM and Streaming

WASM calls `invokeAsync("ai.stream", ...)` which triggers the handler. The handler does the streaming internally (ai-sdk/mastra produce the chunks, not WASM). The invokeAsync callback receives the final result after streaming completes.

For WASM to receive individual chunks in real-time, it would need runtime subscribe capability — deferred to WASM bus parity (feature #2).

---

## 4. LocalInvoker (TS/WASM In-Process)

TS and WASM use the LocalInvoker which calls handlers directly without transport. The LocalInvoker:

1. Generates replyTo metadata (for consistency)
2. Calls the handler
3. Returns the response inline (no pub/sub round-trip)

The LocalInvoker is a short-circuit optimization — same result, no transport overhead. The `replyTo` is set in metadata for handlers that inspect it, but the response doesn't actually go through the bus.

For streaming via LocalInvoker: the handler publishes chunks to the bus (they DO go through transport) even though the request came through LocalInvoker. This means TS code calling `ai.stream()` produces chunks on the bus that Go/Plugin subscribers can receive.

---

## 5. Cross-Kit

Cross-Kit publishes to a remote namespace. The `replyTo` is set to a topic in the CALLER's namespace so the response routes back.

```
Kit A (namespace: "kit-a") calls Kit B (namespace: "kit-b"):

  Publish to: "kit-b.ai.generate"
  metadata:
    replyTo: "kit-a.ai.generate.reply.abc123"
    correlationId: "abc123"

Kit B handler processes, publishes response to:
  "kit-a.ai.generate.reply.abc123"

Kit A subscribes to "ai.generate.reply.abc123" in its own namespace.
```

The cross-Kit Publish variant:

```go
func PublishTo[T BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error)
```

ReplyTo defaults to caller's namespace: `<caller-namespace>.<topic>.reply.<uuid>`.

---

## 6. What Gets Removed

| Removed | Reason |
|---------|--------|
| `sdk.PublishAwait[Req, Resp]` | Replaced by Publish + SubscribeTo |
| `sdk.PublishAwaitTo[Req, Resp]` | Replaced by PublishTo + SubscribeTo |
| `cross.go` old `SubscribeTo` | Rewritten with new signature |
| `resultCh` buffer-16 hack | Was a PublishAwait race workaround |
| `commandSpec.resultTopic` | Handlers use replyTo from metadata |
| `commandSpec.encodeFailure` | Errors in ResultMeta, published to replyTo |
| `RawCommandBinding.ResultTopic` | Dynamic replyTo replaces it |
| `RawCommandBinding.EncodeFailure` | Same |
| All `<topic>.result` BusTopic methods on response types | No longer needed — responses go to replyTo |

### Response BusTopic Methods

Response types (`AiGenerateResp`, `ToolCallResp`, etc.) currently have `BusTopic() string` returning `"<domain>.result"`. These are no longer needed for routing (replyTo replaces them). However, they can stay for type identification if useful, or be removed entirely.

Decision: **Remove them.** Response types don't need BusTopic — they're never used as routing keys anymore. Only request types and event types need BusTopic.

---

## 7. Test Migration

Every test using `PublishAwait` changes to `Publish` + `SubscribeTo`. No test helpers, no shortcuts.

```go
// Before
resp, err := sdk.PublishAwait[AiGenerateMsg, AiGenerateResp](rt, ctx, req)
require.NoError(t, err)
assert.NotEmpty(t, resp.Text)

// After
result, err := sdk.Publish(rt, ctx, req)
require.NoError(t, err)
done := make(chan AiGenerateResp, 1)
cancel, err := sdk.SubscribeTo[AiGenerateResp](rt, ctx, result.ReplyTo, func(resp AiGenerateResp, msg Message) {
    done <- resp
})
require.NoError(t, err)
defer cancel()
select {
case resp := <-done:
    assert.NotEmpty(t, resp.Text)
case <-ctx.Done():
    t.Fatal("timeout")
}
```

This is more verbose but explicit. Every test shows exactly what's happening — publish, subscribe, wait, assert.

All 33 test files need updating. The pattern is mechanical.

---

## 8. Files Changed

| File | Change |
|------|--------|
| `sdk/helpers.go` | Delete PublishAwait/PublishAwaitTo. New: Publish, Emit, SubscribeTo, PublishOption, PublishResult |
| `sdk/cross.go` | Delete old functions. New: PublishTo (cross-Kit with replyTo) |
| `sdk/runtime.go` | Unchanged — PublishRaw/SubscribeRaw stay |
| `sdk/client.go` | Remove Client alias if still present |
| `sdk/messages/*.go` | Remove BusTopic from response types. Keep on request + event types. |
| `internal/messaging/host.go` | Read replyTo from metadata, publish response there |
| `internal/messaging/types.go` | Remove ResultTopic/EncodeFailure from RawCommandBinding |
| `kit/catalog.go` | Remove resultTopic/encodeFailure from commandSpec. Add ai.stream handler. |
| `kit/local_invoker.go` | Generate replyTo metadata, return response inline |
| `kit/handlers_ai.go` | Add Stream handler (publishes chunks to replyTo) |
| `kit/runtime/kit_runtime.js` | ai.stream() bridges chunks to Go for bus publishing |
| All 33 test files | Migrate PublishAwait → Publish + SubscribeTo |
| `test/TEST_COVERAGE.md` | Update |
| `FEATURES.md` | Update messaging model, mark streaming DONE |
