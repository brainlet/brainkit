# Changelog

## Unreleased

### Session 03 — Error envelope Bundle A (JS bridges + gateway + audit + test suite)

Second drop on Bundle A. Rewires the QuickJS bridge + gateway HTTP/WS
layer + audit recorder to the wire envelope contract. Adds the
dedicated `test/suite/envelope/` regression suite.

Added:
- `test/suite/envelope/` — round-trip suite: NOT_FOUND / VALIDATION_ERROR
  typed-error decode, unknown code → `*sdkerrors.BusError` carrier,
  wire-shape invariants (success has `ok:true` + `data`, error has
  `ok:false` + `error.code`/`message`), envelope metadata flag
  presence, and `brainkit.Call` typed-error surfacing across the full
  bus round trip

Changed:
- `internal/engine/bridges_util.go:throwBrainkitError` — the thrown JS
  Error now carries real `.code` and `.details` properties. Deleted
  the `[CODE] msg {{json}}` message-string encoding.
- `internal/engine/runtime/bridges.js:__kit_parseBridgeResponse` —
  reads the wire envelope `{ok,data,error}` and throws a
  `BrainkitError(message, code, details)` on `ok=false`; success
  unwraps `data`. Retains a legacy-shape fallback so non-migrated
  producers keep working.
- `internal/engine/runtime/kit_runtime.js` — deleted `_codeRe`,
  `_detailsRe`, `_parseError`; `rewrapErrors`/`rewrapErrorsAsync`
  now rely exclusively on the JS error's `.code` property (set by
  `throwBrainkitError`) to promote into the Compartment-visible
  `BrainkitError` class
- `internal/engine/bridges_bus.go` — `__go_brainkit_bus_reply` takes
  an optional 5th `envelope` arg; when true, stamps
  `metadata["envelope"]="true"` so the Caller unwraps
- `internal/engine/runtime/bus.js` — wiring in place for envelope
  replies (`msg.reply`/`msg.send`/`msg.stream.end`/`.error`); kept
  **not yet enabled** in this drop so the many raw-decode tests
  stay green. Flip lands with session 03 Bundle B/C.
- `gateway/gateway.go` — `mapHTTPStatus` now consults the wire
  envelope's `error.code` and maps via the full taxonomy table from
  `designs/08-errors.md` (`httpCodes` map). `sanitizeErrorPayload`
  handles both envelope and legacy shapes.
- `gateway/websocket.go` — unwraps success envelopes before writing
  to the WebSocket client so clients see clean JSON; error envelopes
  forward through unchanged.
- `internal/audit/recorder.go` — new `recordErr` helper merges
  `BrainkitError.Code()` / `Details()` into event data as
  `errorCode` / `errorDetails`; `ToolCallFailed`, `DeployFailed`,
  and `BusHandlerFailed` switched to it so the audit log stays
  machine-queryable.
- `test/suite/env.go:ResponseData` — tightened envelope detection:
  requires both `ok` AND (`data` or `error`) keys before unwrapping.
  Without this, user replies like `msg.reply({ok:true,attempt:2})`
  were being falsely unwrapped to `nil`.
- `test/suite/bus/async_diag.go`, `test/suite/bus/failure.go` —
  decode `.ts` replies via `suite.ResponseData` for robustness

Pending (still remaining in session 03):
- Enable `.ts` `msg.reply`/`msg.send`/`msg.stream.end`/`.error`
  envelope wrap — requires updating ~dozen tests that raw-decode
  `.ts` handler replies
- Delete `ResultMeta` + helpers from `sdk/bus_messages.go` +
  21 `sdk/*_messages.go` embeds (mechanical)
- Eval command collapse (Bundle B)
- `.ts` `bus.call` / `bus.stream` / `bus.callTo` (Bundle C)

### Session 03 — Error envelope Bundle A (partial)

Ships the wire envelope infrastructure and migrates every affected bus
consumer in-process to the new shape. Bundle A is end-to-end on the Go
side; .ts bus.js envelope wrapping + bridges_util/bridges.js/kit_runtime.js
rewrapErrors deletion + sdk/*_messages ResultMeta deletion remain for
the follow-up bundle within session 03.

Added:
- `sdk/envelope.go` — `Envelope{Ok, Data, Error}` + `EnvelopeError{Code,
  Message, Details}` + `EnvelopeOK`/`EnvelopeErr`/`EncodeEnvelope`/
  `DecodeEnvelope`/`IsEnvelope` helpers; `FromEnvelope`/`ToEnvelope`
  map between envelopes and typed Go errors
- `sdkerrors.BusError` — generic carrier for error envelopes whose
  `code` does not map to a known typed error; implements
  `BrainkitError` so `errors.As` still works
- `test/suite` helpers `ResponseCode`, `ResponseHasError`,
  `ResponseErrorMessage`, `ResponseErrorDetails`, `ResponseData` —
  accept both envelope and legacy payload shapes

Changed:
- `internal/transport/host.go` — command replies now go out as wire
  envelopes (`{ok:true, data}` / `{ok:false, error:{code,message,details}}`)
  with `metadata["envelope"]="true"` stamped; `SerializeBrainkitError`
  builds the envelope instead of a top-level `{error,code,details}` map
- `internal/engine/kernel_failure.go` `sendErrorResponse` — also emits
  envelope replies via the new `kernel_bus.go:replyEnvelope` helper, so
  JS-handler-throw responses reach the Caller as typed errors
- `internal/bus/caller/caller.go` — when the terminal reply carries
  `envelope=true` metadata, unwraps via `sdk.FromEnvelope` and returns
  either `env.Data` as the raw success payload or the reconstructed
  typed Go error; the Bundle C "sendErrorResponse wins race over
  HandlerFailedError" known-limitation is now fixed as a side effect
- `sdk/helpers.go:SubscribeTo` — decodes envelope-carrying replies
  before unmarshaling into T; error envelopes are flattened into the
  legacy `{error, code, details}` shape so responses still embedding
  `ResultMeta` keep getting populated during the migration

Tests:
- All bus/agents/cli/deploy/secrets/persistence/security/gateway/stress/
  tools/registry/scheduling/workflows/packages/tracing/mcp/plugins/fs/
  health suites green on memory. Bus suite: 94s.
- `call_stream_all_delivered` remains flaky ~20% due to the documented
  GoChannel chunk interleaving (pre-existing, not introduced by this
  bundle). Cross suite failure is the pre-existing Podman infra issue.

Known (carried from Bundle C → fixed here):
- The Bundle C "sendErrorResponse wins race over HandlerFailedError"
  note is resolved: both paths now emit envelopes, and the Caller
  unwraps the envelope into the correct typed error.

Pending (remainder of session 03 Bundle A and B/C):
- `.ts` `bus.js` `msg.reply`/`msg.send`/`stream.end` envelope wrap
- `internal/engine/bridges_util.go` `throwBrainkitError` → real JS
  error with `.code`/`.details` properties (delete `[CODE] msg {{json}}`)
- `internal/engine/runtime/bridges.js` envelope handling in
  `__kit_parseBridgeResponse`; delete `rewrapErrors`/`_codeRe`/
  `_detailsRe`/`_parseError` from `kit_runtime.js`
- Delete `ResultMeta` + `SetError`/`SetErrorWithCode`/`ResultError`/
  `ResultErrorOf` from `sdk/bus_messages.go` + every
  `sdk/*_messages.go` embed
- `gateway/handler.go` HTTP status via envelope taxonomy table
- `internal/audit/recorder.go` structured error recording
- `test/suite/envelope/` dedicated round-trip suite
- Eval command collapse (Bundle B)
- `.ts` `bus.call`/`bus.stream`/`bus.callTo` (Bundle C)

### Session 02 — Caller Bundle C (Checkpoint 3)

Cancellation signal, fail-fast subscription, correlationID-stamped
exhausted events, and a metrics surface. Closes session 02.

Added:
- `brainkit.WithCallNoCancelSignal()` — disables the best-effort
  `_brainkit.cancel` publish that otherwise fires when `ctx` is
  cancelled before a terminal reply arrives
- `caller.CancelTopic` (`_brainkit.cancel`) + `caller.CancelNotice`
  payload type (`correlationId`, `topic`, `reason`)
- `caller.HandlerFailedError` — typed error carrying `Topic`, `Retries`,
  `Cause`; implements `BrainkitError` with code `HANDLER_FAILED`
- `Caller.Snapshot()` returning `MetricsSnapshot` with counters:
  `Inflight`, `Completed`, `TimedOut`, `Cancelled`, `Unmatched`,
  `DecodeErrs`, `BufferOverflows`, `ChunksDelivered`, `ChunksDropped`,
  `FailedFast`
- `test/suite/bus/call_cancel_failfast.go` — 4 tests: cancel notice
  on ctx timeout, `WithCallNoCancelSignal` suppresses, exhausted
  event carries correlationID metadata, metrics snapshot sanity

Changed:
- `internal/bus/caller/caller.go`
  - `NewCaller` now subscribes to `bus.handler.exhausted` in addition
    to the inbox; unsub releases both on `Close`
  - `onFailure` handler matches `msg.Metadata["correlationId"]` against
    pending calls; on hit, finalizes with `*HandlerFailedError`
  - `Call` emits `CancelNotice` on `_brainkit.cancel` when ctx closes
    before a terminal reply (detached 500ms context so already-cancelled
    parent doesn't block the emit); skipped when `NoCancelSignal` set
  - `Metrics.FailedFast` incremented on `HandlerFailedError` finalize
- `internal/engine/kernel_failure.go`
  - `emitHandlerExhausted` now takes `correlationID` and publishes via
    `k.remote.PublishRawWithMeta` so the event carries
    `metadata["correlationId"]`
  - `handleHandlerFailure` reads `correlationID` from the failing
    message's metadata and threads it through
- `internal/engine/kernel_init.go`
  - Auto-generates `watermill.NewUUID()` when `cfg.RuntimeID` is empty
    so low-level Kernel consumers that don't set it still get a Caller

Known:
- When a retry policy is configured and `sendErrorResponse` publishes
  a JSON error payload to the Caller's inbox (done=true), that success
  reply typically wins the race vs. the `bus.handler.exhausted` event.
  `HandlerFailedError` via `onFailure` remains the signal for the
  no-replyTo path; a proper typed-error contract belongs to session 03
  (error envelope).

### Session 02 — Caller Bundle B (Checkpoint 2)

Typed streaming on top of the Caller. Per-pending drain goroutine +
bounded channel + overflow policy.

Added:
- `brainkit.CallStream[Req, Chunk, Resp]` — ordered per-chunk delivery
  through `onChunk` callback, final reply decoded into Resp
- `brainkit.BufferPolicy` (re-exported from `caller.BufferPolicy`) plus
  `BufferBlock`/`BufferDropNewest`/`BufferDropOldest`/`BufferError`
- `brainkit.WithCallBuffer(n)`, `brainkit.WithCallBufferPolicy(p)`
- `caller.BufferOverflowError` — typed failure when `BufferError`
  policy triggers
- New `Metrics` fields: `BufferOverflows`, `ChunksDelivered`,
  `ChunksDropped`
- `test/suite/bus/call_stream.go` — 5 tests: all-delivered, nil-callback
  rejection, BufferError overflow, BufferDropNewest under slow
  consumer, handler-error aborts

Changed:
- `internal/bus/caller/caller.go`
  - `Config` gains `StreamHandler`, `BufferSize`, `BufferPolicy` fields
  - `pendingCall` gains streaming fields + `sendMu` serializing stream
    sends with finalize's close (no "send on closed channel" panic)
  - `onInbox` distinguishes terminal (`done=true`) from chunk; terminal
    uses `LoadAndDelete` so late chunks drop cleanly
  - Stream path: per-pending bounded channel + drain goroutine;
    `drainDone` closed on exit so `Call` waits for all chunks to flush
    before returning
  - `finalize` acquires `sendMu`; `finalizeLocked` handles the inline
    `BufferError` path (caller already holds the lock)
- `internal/transport/host.go` — command replies now set
  `done=true` metadata so the Caller finalizes immediately on bus
  command responses instead of treating them as stream chunks

Known:
- `test/suite/bus/call_stream_all_delivered` uses `assert.ElementsMatch`
  rather than order-sensitive equality. Memory transport (watermill
  GoChannel) does not serialize Publish calls by default, so rapid-fire
  stream chunks can interleave. Each chunk carries a `seq` field for
  consumers that need strict ordering. NATS/AMQP/Redis preserve FIFO
  per subject and will deliver in order on the wire.

### Session 02 — Caller Bundle A (Checkpoint 1)

Foundation for `brainkit.Call`. Shared-inbox reply router per Kit;
metadata-keyed correlation; test helpers + gateway rewritten on top of it.

Added:
- `internal/bus/caller` package
  - `Caller` — single inbox subscription per Kit
    (`_brainkit.inbox.<runtimeID>`), `sync.Map` of pending calls,
    `onInbox` demux, `Close()` finalizes all pending with
    `ErrCallerClosed`
  - `Config{TargetNamespace, Metadata}` for cross-namespace + custom meta
  - Typed errors: `NoDeadlineError`, `CallTimeoutError`,
    `CallCancelledError`, `DecodeError` (all `BrainkitError`)
  - `Metrics`/`MetricsSnapshot` with atomic counters for
    inflight/completed/timedout/cancelled/unmatched/decodeErrs
- `brainkit.Call[Req, Resp]` generic — marshals, invokes `Caller.Call`,
  unmarshals; `json.RawMessage` short-circuits decode
- `brainkit.WithCallTimeout`, `WithCallTo`, `WithCallMeta`
- `Kit.Caller()` + `Kernel.Caller()` accessors
- `test/suite/bus/call.go` — 7 tests covering happy path, deadline gate,
  timeout/cancel errors, 50× concurrent demux, raw-payload short-circuit

Changed:
- `internal/engine/kernel_init.go` — constructs Caller after transport init;
  uses `Kernel` as its `sdk.Runtime` so inbox resolves into local namespace
- `internal/engine/kernel_shutdown.go` — calls `Caller.Close()` during
  shutdown, before storages/transport teardown
- `internal/testutil/bus_helpers.go` — `roundTrip` now delegates to
  `Caller.Call` via a `callerHolder` interface check on the runtime
  (no more per-call `subscribe + publish` dance)
- `gateway/handler.go` — `handleRequest` uses `Caller.Call`; typed-error
  switch for timeout vs cancel

Out of scope (later bundles):
- Streaming + backpressure (Bundle B)
- Cancellation emit + fail-fast via `bus.handler.exhausted` (Bundle C)
- Error envelope (session 03)

### Session 01 — Phase 0 Cleanup

Pure subtraction. No new API, no behavior changes — only removal of orphaned
code from prior feature deletions.

Removed:
- `test/suite/rbac/` domain (RBAC was removed previously; the stranded test
  suite still lived in-tree)
- `internal/engine/scaling.go` — `InstanceManager`, `PoolConfig`, `PoolMode`,
  `PoolSharded`, `PoolReplicated`, `pool`, `StaticStrategy`
- `internal/types/scaling.go` — `ScalingStrategy`, `ScalingDecision`,
  `PoolInfo` types
- Scaling re-exports in root `types.go`: `InstanceManager`, `PoolConfig`,
  `StaticStrategy`, `ScalingDecision`, `ScalingStrategy`, `PoolInfo`,
  `PoolMode`, `PoolSharded`, `PoolReplicated`, `NewInstanceManager`,
  `NewStaticStrategy`
- `Kit.HealthJSON` public method and `Kernel.HealthJSON` — the `kit.health`
  bus command marshals `Kernel.Health(ctx)` inline; `gateway/health.go`
  drops its `healthJSONer` branch and always uses the `alive + ready`
  fallback on `/health`
- `test/suite/stress/scaling.go` and its 7 pool/strategy tests
- `testStorageRuntimeScalingPool` in `test/suite/registry/storage_runtime.go`
- `testHealthJSON` in `test/suite/gateway/routes.go`
- `testConcurrencyRBACAssignCheckRace`, `testTimingRoleChangeWhileHandlerRunning`,
  `testBusRateLimitExceeds`, `testErrorContractBusNotConfiguredRBAC`,
  `testRolePreservedAcrossRestart` — all were RBAC-era stubs that only `t.Skip`
- `secDeployWithRole` helper in `test/suite/security/run.go`
- `role` parameter on `testutil.DeployWithOpts`
- `rbacOnly` field on `test/suite/bus/surface.go` `cmdTest`
- `rbac.assign` / `rbac.revoke` from the forbidden-topic list in
  `test/suite/security/bus_forgery.go`
- `docs/guides/scaling-and-pools.md` guide
- `test/campaigns/fullstack/nats_postgres_rbac_test.go`
- References to the removed symbols across `docs/`, `TEST_MAP.md`,
  `CLAUDE.md` files, and `internal/docs/FEATURES.md`

Changed:
- `MetricsSnapshot` moved from `internal/types/scaling.go` into
  `internal/types/types.go` (still the same struct; only the owning file
  changed)
- `NotConfiguredError` feature strings referencing `"rbac"` now use `"mcp"`
