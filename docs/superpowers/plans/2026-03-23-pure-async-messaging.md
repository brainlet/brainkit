# Pure Async Messaging Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace PublishAwait with pure async pub/sub using reply-to addressing. Formalize streaming as a catalog command. Migrate all 369 PublishAwait usages across 31 test files.

**Architecture:** Commands use `sdk.Publish` which returns a `PublishResult` with a `ReplyTo` topic. Handlers read `replyTo` from message metadata and publish responses there. Callers subscribe to `ReplyTo` via `sdk.SubscribeTo[T]`. Events use `sdk.Emit` (fire-and-forget). Streaming is a standard command where the handler publishes multiple `StreamChunk` messages to `replyTo`.

**Tech Stack:** Go generics, Watermill pub/sub, QuickJS (kit_runtime.js)

---

### Task 1: New SDK Types and Functions

**Files:**
- Modify: `sdk/helpers.go`
- Modify: `sdk/cross.go`
- Modify: `sdk/messages/bus.go`

- [ ] **Step 1: Rewrite `sdk/helpers.go`**

Delete everything in helpers.go. Replace with:

```go
package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
)

// PublishResult contains metadata about a published command.
type PublishResult struct {
	MessageID     string
	CorrelationID string
	ReplyTo       string
	Topic         string
}

type publishConfig struct {
	replyTo string
}

// PublishOption configures a Publish call.
type PublishOption func(*publishConfig)

// WithReplyTo overrides the auto-generated reply topic.
func WithReplyTo(topic string) PublishOption {
	return func(c *publishConfig) { c.replyTo = topic }
}

// Publish sends a typed command. Always generates a replyTo for response routing.
// Default convention: <topic>.reply.<uuid>
func Publish[T messages.BrainkitMessage](rt Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error) {
	cfg := publishConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	topic := msg.BusTopic()
	correlationID := uuid.NewString()
	replyTo := cfg.replyTo
	if replyTo == "" {
		replyTo = topic + ".reply." + correlationID
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return PublishResult{}, fmt.Errorf("marshal %T: %w", msg, err)
	}

	ctx = withPublishMeta(ctx, correlationID, replyTo)
	msgID, err := rt.PublishRaw(ctx, topic, payload)
	if err != nil {
		return PublishResult{}, err
	}

	return PublishResult{
		MessageID:     msgID,
		CorrelationID: correlationID,
		ReplyTo:       replyTo,
		Topic:         topic,
	}, nil
}

// Emit sends a fire-and-forget event. No replyTo, no response expected.
func Emit[T messages.BrainkitMessage](rt Runtime, ctx context.Context, msg T) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %T: %w", msg, err)
	}
	_, err = rt.PublishRaw(ctx, msg.BusTopic(), payload)
	return err
}

// SubscribeTo listens for typed messages on a specific topic.
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string, handler func(T, messages.Message)) (func(), error) {
	return rt.SubscribeRaw(ctx, topic, func(msg messages.Message) {
		var typed T
		if err := json.Unmarshal(msg.Payload, &typed); err != nil {
			return
		}
		handler(typed, msg)
	})
}
```

- [ ] **Step 2: Add `withPublishMeta` to `internal/messaging/context.go`**

Add a function that stamps correlationID + replyTo into context so PublishRaw can read them:

```go
// withPublishMeta stamps correlationID and replyTo into context for PublishRaw.
func WithPublishMeta(ctx context.Context, correlationID, replyTo string) context.Context {
	ctx = context.WithValue(ctx, correlationIDContextKey, correlationID)
	ctx = context.WithValue(ctx, replyToContextKey, replyTo)
	return ctx
}

func ReplyToFromContext(ctx context.Context) string {
	if ctx == nil { return "" }
	v, _ := ctx.Value(replyToContextKey).(string)
	return v
}
```

Add `replyToContextKey` to the context keys.

- [ ] **Step 3: Update `internal/messaging/client.go` `PublishRaw` to stamp replyTo metadata**

In `RemoteClient.PublishRaw`, after setting correlationId, add:

```go
if replyTo := ReplyToFromContext(ctx); replyTo != "" {
	wmsg.Metadata.Set("replyTo", replyTo)
}
```

Same for `PublishRawToNamespace`.

- [ ] **Step 4: Rewrite `sdk/cross.go`**

Delete old PublishAwaitTo/PublishTo/SubscribeTo. Replace with:

```go
package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
)

// PublishTo sends a typed command to a specific Kit's namespace.
func PublishTo[T messages.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error) {
	xrt, ok := rt.(CrossNamespaceRuntime)
	if !ok {
		return PublishResult{}, fmt.Errorf("runtime does not support cross-namespace operations")
	}

	cfg := publishConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	topic := msg.BusTopic()
	correlationID := uuid.NewString()
	replyTo := cfg.replyTo
	if replyTo == "" {
		replyTo = topic + ".reply." + correlationID
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return PublishResult{}, fmt.Errorf("marshal %T: %w", msg, err)
	}

	ctx = withPublishMeta(ctx, correlationID, replyTo)
	msgID, err := xrt.PublishRawTo(ctx, targetNamespace, topic, payload)
	if err != nil {
		return PublishResult{}, err
	}

	return PublishResult{
		MessageID:     msgID,
		CorrelationID: correlationID,
		ReplyTo:       replyTo,
		Topic:         topic,
	}, nil
}
```

- [ ] **Step 5: Build and verify compilation**

Run: `go build ./sdk/...`
Expected: FAIL — tests and other packages reference deleted PublishAwait. That's expected. The SDK layer is ready.

- [ ] **Step 6: Commit**

```bash
git add sdk/helpers.go sdk/cross.go internal/messaging/context.go internal/messaging/client.go
git commit -m "refactor(sdk): pure async Publish/Emit/SubscribeTo — delete PublishAwait"
```

---

### Task 2: Handler Side — replyTo Response Routing

**Files:**
- Modify: `internal/messaging/host.go`
- Modify: `internal/messaging/types.go` (if RawCommandBinding is there)
- Modify: `kit/catalog.go`

- [ ] **Step 1: Update `RawCommandBinding`**

Remove `ResultTopic` and `EncodeFailure` fields. The binding only needs `Name`, `Topic`, and `Handle`.

```go
type RawCommandBinding struct {
	Name   string
	Topic  string
	Handle func(context.Context, json.RawMessage) (json.RawMessage, error)
}
```

- [ ] **Step 2: Rewrite `host.go` `RegisterCommands`**

The handler wrapper reads `replyTo` from inbound message metadata. On success or error, publishes to `replyTo`.

```go
func (h *Host) RegisterCommands(bindings []RawCommandBinding) {
	for _, binding := range bindings {
		binding := binding
		commandTopic := h.resolvedTopic(binding.Topic)
		handlerName := rawHandlerName(binding.Name, binding.Topic)

		h.router.AddConsumerHandler(
			handlerName,
			commandTopic,
			h.sub,
			func(wmsg *message.Message) error {
				cmdCtx := withInboundMetadata(wmsg.Context(), wmsg, binding.Topic)
				payload, err := binding.Handle(cmdCtx, json.RawMessage(wmsg.Payload))

				replyTo := wmsg.Metadata.Get("replyTo")
				if replyTo == "" {
					// No replyTo — command cannot route response. Log and drop.
					if err != nil {
						log.Printf("[host] command %s failed with no replyTo: %v", binding.Topic, err)
					}
					return nil
				}

				// Build response — on error, wrap in ResultMeta-style error payload
				var responsePayload []byte
				if err != nil {
					if IsDecodeFailure(err) {
						return err
					}
					// Wrap error in a generic error response
					responsePayload, _ = json.Marshal(map[string]string{"error": err.Error()})
				} else if payload != nil {
					responsePayload = payload
				} else {
					return nil
				}

				result := message.NewMessage(watermill.NewUUID(), responsePayload)
				correlationID := wmsg.Metadata.Get("correlationId")
				if correlationID != "" {
					result.Metadata.Set("correlationId", correlationID)
				}

				resolvedReplyTo := replyTo
				if h.topicSanitizer != nil {
					resolvedReplyTo = h.topicSanitizer(replyTo)
				}
				return h.pub.Publish(resolvedReplyTo, result)
			},
		)
	}
}
```

- [ ] **Step 3: Update `kit/catalog.go`**

Remove `resultTopic` and `encodeFailure` from `commandSpec`. Remove `legacyResultSuffix`. Remove result topic validation panics. Remove `encodeFailure` wrappers from `kernelCommand`/`nodeCommand`.

The `commandSpec` becomes:

```go
type commandSpec struct {
	topic        string
	validate     func(json.RawMessage) error
	invokeKernel func(context.Context, *Kernel, json.RawMessage) (json.RawMessage, error)
	invokeNode   func(context.Context, *Node, json.RawMessage) (json.RawMessage, error)
}
```

Update `kernelCommand`/`nodeCommand` generics to not capture `Resp` type for result topic — only `Req` is needed now.

Update `BindingsForNode` and `commandBindingsForKernel` to build `RawCommandBinding` without ResultTopic/EncodeFailure.

- [ ] **Step 4: Build**

Run: `go build ./kit/... ./internal/messaging/...`
Expected: May still fail due to test references. Core handler pipeline compiles.

- [ ] **Step 5: Commit**

```bash
git add internal/messaging/host.go kit/catalog.go
git commit -m "refactor(handlers): replyTo response routing — remove resultTopic/encodeFailure"
```

---

### Task 3: Remove BusTopic from Response Types

**Files:**
- Modify: `sdk/messages/agents.go`
- Modify: `sdk/messages/ai.go`
- Modify: `sdk/messages/fs.go`
- Modify: `sdk/messages/kit.go`
- Modify: `sdk/messages/mcp.go`
- Modify: `sdk/messages/memory.go`
- Modify: `sdk/messages/plugin.go`
- Modify: `sdk/messages/registry.go`
- Modify: `sdk/messages/tools.go`
- Modify: `sdk/messages/vectors.go`
- Modify: `sdk/messages/wasm.go`
- Modify: `sdk/messages/workflows.go`

- [ ] **Step 1: Remove all `BusTopic()` methods from response types**

Delete every `func (XxxResp) BusTopic() string { return "xxx.result" }` line across all message files. Keep BusTopic on request types and event types.

Response types that lose BusTopic: `AgentRequestResp`, `AgentUnregisterResp`, `AgentGetStatusResp`, `AgentSetStatusResp`, `AgentListResp`, `AgentDiscoverResp`, `AgentMessageResp`, `AiGenerateResp`, `AiEmbedResp`, `AiEmbedManyResp`, `AiGenerateObjectResp`, `FsReadResp`, `FsListResp`, `FsStatResp`, `FsWriteResp`, `FsDeleteResp`, `FsMkdirResp`, `KitDeployResp`, `KitTeardownResp`, `KitRedeployResp`, `KitListResp`, `McpListToolsResp`, `McpCallToolResp`, `MemoryCreateThreadResp`, `MemoryGetThreadResp`, `MemoryListThreadsResp`, `MemorySaveResp`, `MemoryRecallResp`, `MemoryDeleteThreadResp`, `PluginManifestResp`, `PluginStateGetResp`, `PluginStateSetResp`, `RegistryHasResp`, `RegistryListResp`, `RegistryResolveResp`, `ToolListResp`, `ToolResolveResp`, `ToolCallResp`, `VectorUpsertResp`, `VectorQueryResp`, `VectorCreateIndexResp`, `VectorDeleteIndexResp`, `VectorListIndexesResp`, `WasmCompileResp`, `WasmRunResp`, `WasmDeployResp`, `WasmUndeployResp`, `WasmListResp`, `WasmGetResp`, `WasmRemoveResp`, `WasmDescribeResp`, `WorkflowRunResp`, `WorkflowResumeResp`, `WorkflowCancelResp`, `WorkflowStatusResp`.

- [ ] **Step 2: Build**

Run: `go build ./sdk/messages/...`
Expected: PASS — response types no longer implement BrainkitMessage interface.

- [ ] **Step 3: Commit**

```bash
git add sdk/messages/
git commit -m "refactor(messages): remove BusTopic from response types — responses go to replyTo"
```

---

### Task 4: Update Plugin SDK (serve.go)

**Files:**
- Modify: `sdk/serve.go`

- [ ] **Step 1: Replace PublishAwait in plugin manifest registration**

`serve.go` line 104 uses `PublishAwait` for manifest registration. Replace with Publish + SubscribeTo pattern:

```go
result, err := Publish(rt, context.Background(), messages.PluginManifestMsg{...})
if err != nil {
	return fmt.Errorf("sdk: publish manifest: %w", err)
}

regCh := make(chan error, 1)
cancel, err := SubscribeTo[messages.PluginManifestResp](rt, context.Background(), result.ReplyTo, func(resp messages.PluginManifestResp, msg messages.Message) {
	if resp.Error != "" {
		regCh <- fmt.Errorf("sdk: register manifest: %s", resp.Error)
	} else {
		regCh <- nil
	}
})
if err != nil {
	return fmt.Errorf("sdk: subscribe manifest result: %w", err)
}
defer cancel()

select {
case err := <-regCh:
	if err != nil {
		return err
	}
case <-time.After(30 * time.Second):
	return fmt.Errorf("sdk: manifest registration timeout")
}
```

- [ ] **Step 2: Update plugin tool call handler in `node.go`**

The plugin tool executor in `node.go` (`processPluginManifest`) uses its own correlationID + subscribe pattern. Update to use replyTo:

Replace the manual correlationID/subscribe/publish pattern with the new `Publish` + `SubscribeTo` pattern, or keep the low-level approach but read replyTo from the inbound metadata.

- [ ] **Step 3: Build**

Run: `go build ./sdk/... ./kit/...`

- [ ] **Step 4: Commit**

```bash
git add sdk/serve.go kit/node.go
git commit -m "refactor(plugin): migrate serve.go and node.go to pure async"
```

---

### Task 5: Add ai.stream Handler to Catalog

**Files:**
- Modify: `kit/handlers_ai.go`
- Modify: `kit/catalog.go`

- [ ] **Step 1: Add Stream method to AIDomain**

```go
func (d *AIDomain) Stream(ctx context.Context, req messages.AiStreamMsg) (*messages.StreamChunk, error) {
	// The stream handler is special — it publishes multiple chunks to replyTo.
	// The replyTo comes from the context (set by the handler wrapper).
	replyTo := messaging.ReplyToFromContext(ctx)
	correlationID := messaging.CorrelationIDFromContext(ctx)

	// Evaluate streaming in JS — collect chunks and publish each one
	code := fmt.Sprintf(`
		var req = globalThis.__pending_req;
		var model = req.model;
		var chunks = [];
		var stream = ai.stream({ model: model, prompt: req.prompt, messages: req.messages });
		var reader = stream.textStream.getReader();
		var seq = 0;
		var fullText = "";
		while (true) {
			var chunk = await reader.read();
			if (chunk.done) break;
			var delta = chunk.value || "";
			fullText += delta;
			chunks.push(JSON.stringify({ streamId: "%s", seq: seq, delta: delta, done: false }));
			seq++;
		}
		chunks.push(JSON.stringify({ streamId: "%s", seq: seq, delta: "", done: true, final: JSON.stringify({ text: fullText }) }));
		return JSON.stringify(chunks);
	`, correlationID, correlationID)

	raw, err := d.kit.evalDomain(ctx, req, "__ai_stream.ts", code)
	if err != nil {
		return nil, fmt.Errorf("ai.stream: %w", err)
	}

	// Parse chunks array and publish each to replyTo
	var chunkStrings []string
	json.Unmarshal(raw, &chunkStrings)

	for _, chunkJSON := range chunkStrings {
		if replyTo != "" {
			d.kit.publish(ctx, replyTo, json.RawMessage(chunkJSON))
		}
	}

	// Return the final chunk (handler wrapper will also publish to replyTo,
	// but for streaming we already published all chunks including the final one)
	return nil, nil // signal: already published responses directly
}
```

Note: The handler returns nil to signal "I already published responses." The host wrapper should handle this by not publishing an additional response if payload is nil.

- [ ] **Step 2: Register in catalog**

Add to `commandCatalog()` specs:

```go
kernelCommand(func(ctx context.Context, kernel *Kernel, req messages.AiStreamMsg) (*messages.StreamChunk, error) {
	return kernel.ai.Stream(ctx, req)
}),
```

Note: `kernelCommand` will need to be adjusted since `StreamChunk` is now the response type but streaming is special (handler publishes directly). This may need a `streamingCommand` variant.

- [ ] **Step 3: Build**

Run: `go build ./kit/...`

- [ ] **Step 4: Commit**

```bash
git add kit/handlers_ai.go kit/catalog.go
git commit -m "feat(streaming): add ai.stream handler to catalog — publishes chunks to replyTo"
```

---

### Task 6: Migrate Test Files — Core Tests

**Files:** All test files listed below. Pattern is mechanical — replace every `sdk.PublishAwait[Req, Resp](rt, ctx, req)` with Publish + SubscribeTo + channel + select.

The pattern for EVERY PublishAwait replacement:

```go
// Old:
resp, err := sdk.PublishAwait[ReqType, RespType](rt, ctx, req)
require.NoError(t, err)
// use resp

// New:
result, err := sdk.Publish(rt, ctx, req)
require.NoError(t, err)
done := make(chan RespType, 1)
unsub, err := sdk.SubscribeTo[RespType](rt, ctx, result.ReplyTo, func(resp RespType, msg messages.Message) {
	done <- resp
})
require.NoError(t, err)
defer unsub()
select {
case resp := <-done:
	// use resp
case <-ctx.Done():
	t.Fatal("timeout")
}
```

For `sdk.Publish` that currently returns `(string, error)` — the new `sdk.Publish` returns `(PublishResult, error)`. Update callers to use `result.CorrelationID` where they used the old string return.

For `sdk.PublishAwaitTo` (cross-Kit) — replace with `sdk.PublishTo` + `sdk.SubscribeTo`.

Migrate in this order (dependencies first):

- [ ] **Step 1: `test/helpers_test.go`** — no PublishAwait, but may need imports updated
- [ ] **Step 2: `test/go_direct_tools_test.go`** — 5 PublishAwait calls
- [ ] **Step 3: `test/go_direct_fs_test.go`** — ~15 PublishAwait calls
- [ ] **Step 4: `test/go_direct_agents_test.go`** — ~15 PublishAwait calls
- [ ] **Step 5: `test/go_direct_kit_test.go`** — ~10 PublishAwait calls
- [ ] **Step 6: `test/go_direct_wasm_test.go`** — ~20 PublishAwait calls
- [ ] **Step 7: `test/go_direct_ai_test.go`** — 4 PublishAwait calls
- [ ] **Step 8: `test/go_direct_memory_test.go`** — ~10 PublishAwait calls
- [ ] **Step 9: `test/go_direct_workflows_test.go`** — ~10 PublishAwait calls
- [ ] **Step 10: `test/go_direct_vectors_test.go`** — ~5 PublishAwait calls
- [ ] **Step 11: `test/go_direct_mcp_test.go`** — ~5 PublishAwait calls
- [ ] **Step 12: Build and run migrated tests**

Run: `go test ./test/ -run "TestGoDirect" -timeout 300s`
Expected: All pass with new async pattern.

- [ ] **Step 13: Commit**

```bash
git add test/go_direct_*.go test/helpers_test.go
git commit -m "test: migrate Go Direct tests to pure async Publish + SubscribeTo"
```

---

### Task 7: Migrate Test Files — Async, Streaming, WASM, Plugin

**Files:**
- `test/async_test.go`
- `test/streaming_test.go`
- `test/wasm_invokeAsync_test.go`
- `test/wasm_reply_test.go`
- `test/plugin_inprocess_test.go`
- `test/plugin_subprocess_test.go`
- `test/e2e_scenarios_test.go`
- `test/log_handler_test.go`
- `test/registry_integration_test.go`
- `test/probe_test.go`

- [ ] **Step 1: Migrate `async_test.go`** — rewrite completely since it tests the async pattern itself
- [ ] **Step 2: Migrate `streaming_test.go`** — use new Publish + SubscribeTo for stream chunks
- [ ] **Step 3: Migrate `wasm_invokeAsync_test.go`** — WASM tests use PublishAwait for setup (compile/run)
- [ ] **Step 4: Migrate `wasm_reply_test.go`** — same pattern
- [ ] **Step 5: Migrate `plugin_inprocess_test.go`**
- [ ] **Step 6: Migrate `plugin_subprocess_test.go`**
- [ ] **Step 7: Migrate `e2e_scenarios_test.go`**
- [ ] **Step 8: Migrate `log_handler_test.go`**
- [ ] **Step 9: Migrate `registry_integration_test.go`**
- [ ] **Step 10: Migrate `probe_test.go`**
- [ ] **Step 11: Build and run**

Run: `go test ./test/ -run "TestAsync|TestStreaming|TestWASM|TestPlugin|TestE2E|TestLog|TestRegistry|TestProbe" -timeout 300s`

- [ ] **Step 12: Commit**

```bash
git add test/async_test.go test/streaming_test.go test/wasm_*.go test/plugin_*.go test/e2e_*.go test/log_*.go test/registry_*.go test/probe_test.go
git commit -m "test: migrate async/streaming/wasm/plugin/e2e/registry/probe to pure async"
```

---

### Task 8: Migrate Test Files — Cross-Surface, Cross-Kit, Backend Matrix, Chains

**Files:**
- `test/backend_matrix_test.go`
- `test/crosskit_test.go`
- `test/chain_test.go`
- `test/cross_ts_go_test.go`
- `test/cross_wasm_go_test.go`
- `test/cross_ts_wasmmod_test.go`
- `test/cross_plugin_go_test.go`
- `test/cross_ts_plugin_test.go`
- `test/cross_wasmmod_plugin_test.go`
- `test/transport_compliance_test.go`

- [ ] **Step 1: Migrate `backend_matrix_test.go`** — 12 subtests × PublishAwait → Publish + SubscribeTo
- [ ] **Step 2: Migrate `crosskit_test.go`** — PublishAwaitTo → PublishTo + SubscribeTo
- [ ] **Step 3: Migrate `chain_test.go`**
- [ ] **Step 4: Migrate `cross_ts_go_test.go`**
- [ ] **Step 5: Migrate `cross_wasm_go_test.go`**
- [ ] **Step 6: Migrate `cross_ts_wasmmod_test.go`**
- [ ] **Step 7: Migrate `cross_plugin_go_test.go`**
- [ ] **Step 8: Migrate `cross_ts_plugin_test.go`**
- [ ] **Step 9: Migrate `cross_wasmmod_plugin_test.go`**
- [ ] **Step 10: Migrate `transport_compliance_test.go`**
- [ ] **Step 11: Build and run**

Run: `go test ./test/ -run "TestBackendMatrix|TestCrossKit|TestChain|TestCross|TestTransport" -timeout 300s`

- [ ] **Step 12: Commit**

```bash
git add test/backend_matrix_test.go test/crosskit_test.go test/chain_test.go test/cross_*.go test/transport_*.go
git commit -m "test: migrate cross-surface/cross-kit/backend-matrix/chains to pure async"
```

---

### Task 9: Migrate Test Files — Surface Tests

**Files:**
- `test/surface_ts_test.go`
- `test/surface_wasmmod_test.go`
- `test/surface_plugin_test.go`

- [ ] **Step 1: Migrate `surface_ts_test.go`** — ~30 PublishAwait calls
- [ ] **Step 2: Migrate `surface_wasmmod_test.go`** — ~5 PublishAwait calls (mostly for setup)
- [ ] **Step 3: Migrate `surface_plugin_test.go`** — ~40 PublishAwait calls
- [ ] **Step 4: Build and run ALL tests**

Run: `go test ./test/ -timeout 300s`
Expected: ALL 98 test functions pass.

- [ ] **Step 5: Commit**

```bash
git add test/surface_*.go
git commit -m "test: migrate surface tests to pure async — ALL 98 tests pass"
```

---

### Task 10: Cleanup and Documentation

**Files:**
- Modify: `sdk/client.go` — remove `Client` alias if still present
- Modify: `test/TEST_COVERAGE.md`
- Modify: `FEATURES.md`

- [ ] **Step 1: Clean up `sdk/client.go`**

Remove `type Client = Runtime` alias.

- [ ] **Step 2: Verify no PublishAwait references remain**

Run: `grep -rn "PublishAwait" . --include="*.go" | grep -v "_test.go"`
Expected: Zero results.

Run: `grep -rn "PublishAwait" . --include="*.go"`
Expected: Zero results.

- [ ] **Step 3: Run full test suite**

Run: `go test ./test/ -timeout 600s -count=1`
Expected: ALL pass.

Run: `go test ./kit/... ./internal/... ./sdk/... -timeout 120s`
Expected: ALL pass.

- [ ] **Step 4: Update `FEATURES.md`**

Update messaging model section. Mark streaming as DONE. Document the pure async pattern. Remove "PublishAwait" from any feature descriptions.

- [ ] **Step 5: Update `test/TEST_COVERAGE.md`**

Update to reflect new test patterns. Note that all tests use Publish + SubscribeTo.

- [ ] **Step 6: Commit**

```bash
git add sdk/client.go FEATURES.md test/TEST_COVERAGE.md
git commit -m "docs: update FEATURES.md and TEST_COVERAGE.md for pure async messaging"
```

---

### Task Summary

| Task | Description | Files | Estimated Steps |
|------|-------------|-------|----------------|
| 1 | New SDK types (Publish, Emit, SubscribeTo, PublishResult) | 4 files | 6 |
| 2 | Handler replyTo routing | 3 files | 5 |
| 3 | Remove BusTopic from response types | 12 files | 3 |
| 4 | Plugin SDK migration | 2 files | 4 |
| 5 | ai.stream handler | 2 files | 4 |
| 6 | Migrate Go Direct tests | 12 files | 13 |
| 7 | Migrate async/streaming/wasm/plugin/e2e tests | 10 files | 12 |
| 8 | Migrate cross-surface/cross-kit/matrix tests | 10 files | 12 |
| 9 | Migrate surface tests | 3 files | 5 |
| 10 | Cleanup and documentation | 3 files | 6 |
| **Total** | | **~60 files** | **70 steps** |
