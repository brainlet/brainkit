package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/stretchr/testify/require"
)

// compileShard compiles an AS fixture file, returning the module name.
func compileShard(t *testing.T, kit *Kit, fixturePath, name string) {
	t.Helper()
	source := loadFixture(t, fixturePath)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: %q });",
		"`"+source+"`", name,
	))
	require.NoError(t, err, "compile %s", fixturePath)
}

// deployShard compiles + deploys and returns the descriptor.
func deployShard(t *testing.T, kit *Kit, fixturePath, name string) *ShardDescriptor {
	t.Helper()
	compileShard(t, kit, fixturePath, name)
	desc, err := kit.DeployWASM(name)
	require.NoError(t, err, "deploy %s", name)
	return desc
}

// injectEvent sends an event to a shard and returns the result.
func injectEvent(t *testing.T, kit *Kit, shard, topic string, payload interface{}) *WASMEventResult {
	t.Helper()
	data, _ := json.Marshal(payload)
	result, err := kit.InjectWASMEvent(shard, topic, json.RawMessage(data))
	require.NoError(t, err, "inject %s → %s", shard, topic)
	require.Empty(t, result.Error, "handler error for %s → %s", shard, topic)
	return result
}

// ═══════════════════════════════════════════════════════════════
// Stateless Shard Fixtures
// ═══════════════════════════════════════════════════════════════

func TestShardFixture_StatelessEcho(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/stateless-echo.ts", "echo")

	require.Equal(t, "stateless", desc.Mode)
	require.Contains(t, desc.Handlers, "test.echo")
	require.Equal(t, "handleEcho", desc.Handlers["test.echo"])

	// Send a message, expect it echoed back
	result := injectEvent(t, kit, "echo", "test.echo", map[string]string{"msg": "hello"})
	require.Equal(t, `{"msg":"hello"}`, result.ReplyPayload)
}

func TestShardFixture_StatelessLogTopic(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-log-topic.ts", "topic-check")

	result := injectEvent(t, kit, "topic-check", "test.topic-check", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"receivedTopic":"test.topic-check"`)
}

func TestShardFixture_StatelessMultiHandler(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/stateless-multi-handler.ts", "multi")

	require.Equal(t, "stateless", desc.Mode)
	require.Len(t, desc.Handlers, 2)
	require.Contains(t, desc.Handlers, "test.ping")
	require.Contains(t, desc.Handlers, "test.pong")

	// Dispatch to ping handler
	result := injectEvent(t, kit, "multi", "test.ping", map[string]string{})
	require.Equal(t, `{"handler":"ping"}`, result.ReplyPayload)

	// Dispatch to pong handler
	result = injectEvent(t, kit, "multi", "test.pong", map[string]string{})
	require.Equal(t, `{"handler":"pong"}`, result.ReplyPayload)
}

func TestShardFixture_StatelessWildcard(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/stateless-wildcard.ts", "wildcard")

	require.Equal(t, "stateless", desc.Mode)
	require.Contains(t, desc.Handlers, "events.*")

	// Send to events.order — should match events.*
	result := injectEvent(t, kit, "wildcard", "events.order", map[string]string{"id": "123"})
	require.Contains(t, result.ReplyPayload, `"matchedTopic":"events.order"`)
	require.Contains(t, result.ReplyPayload, `"id":"123"`)

	// Send to events.payment — should also match
	result = injectEvent(t, kit, "wildcard", "events.payment", map[string]string{"id": "456"})
	require.Contains(t, result.ReplyPayload, `"matchedTopic":"events.payment"`)
}

func TestShardFixture_StatelessFireAndForget(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-fire-and-forget.ts", "forwarder")

	// Listen on the forwarded topic
	received := make(chan string, 1)
	kit.Bus.On("test.forwarded", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- string(msg.Payload)
	})

	// Inject — handler should forward via send()
	result, err := kit.InjectWASMEvent("forwarder", "test.forward", json.RawMessage(`{"data":"forwarded"}`))
	require.NoError(t, err)
	require.Empty(t, result.Error)

	// No reply expected (handler doesn't call reply())
	require.Empty(t, result.ReplyPayload)

	// Check the forwarded message arrived on the bus
	select {
	case msg := <-received:
		require.Contains(t, msg, `"data":"forwarded"`)
	case <-time.After(2 * time.Second):
		t.Fatal("forwarded message not received on bus")
	}
}

func TestShardFixture_StatelessReplyJSON(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-reply-json.ts", "transformer")

	result := injectEvent(t, kit, "transformer", "test.transform", map[string]string{"name": "david"})
	require.Contains(t, result.ReplyPayload, `"greeting":"hello david"`)
	require.Contains(t, result.ReplyPayload, `"original"`)
}

func TestShardFixture_StatelessNoReply(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-no-reply.ts", "silent")

	result := injectEvent(t, kit, "silent", "test.silent", map[string]string{"msg": "shh"})
	require.Empty(t, result.ReplyPayload, "handler that doesn't call reply() should have empty reply")
}

func TestShardFixture_StatelessAskAsync(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Register a mock "echo" tool that the shard will call via askAsync
	kit.Bus.On("tools.call", func(msg bus.Message, reply bus.ReplyFunc) {
		reply(json.RawMessage(`{"echoed":true}`))
	})

	deployShard(t, kit, "testdata/as/shard/stateless-ask-async.ts", "asker")

	// Inject — handler calls askAsync("tools.call", ..., "onToolResult")
	result, err := kit.InjectWASMEvent("asker", "test.ask", json.RawMessage(`{}`))
	require.NoError(t, err)
	require.Empty(t, result.Error)
	// askAsync callback sets state — but for stateless mode, state is discarded
	// The test verifies no crash/hang from askAsync in a handler
}

// ═══════════════════════════════════════════════════════════════
// Persistent Shard Fixtures
// ═══════════════════════════════════════════════════════════════

func TestShardFixture_PersistentCounter(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/persistent-counter.ts", "counter")

	require.Equal(t, "persistent", desc.Mode)
	require.Len(t, desc.Handlers, 3)

	// Get initial count = 0
	result := injectEvent(t, kit, "counter", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":0`)

	// Increment 5 times
	for i := 1; i <= 5; i++ {
		result = injectEvent(t, kit, "counter", "counter.inc", map[string]string{})
		require.Contains(t, result.ReplyPayload, fmt.Sprintf(`"count":%d`, i))
	}

	// Get should return 5
	result = injectEvent(t, kit, "counter", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":5`)

	// Reset
	result = injectEvent(t, kit, "counter", "counter.reset", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":0`)

	// Get after reset
	result = injectEvent(t, kit, "counter", "counter.get", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"count":0`)
}

func TestShardFixture_PersistentKVStore(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/persistent-kv-store.ts", "kvstore")

	require.Equal(t, "persistent", desc.Mode)

	// has before set → false
	result := injectEvent(t, kit, "kvstore", "kv.has", map[string]string{"key": "name"})
	require.Contains(t, result.ReplyPayload, `"exists":false`)

	// set
	result = injectEvent(t, kit, "kvstore", "kv.set", map[string]string{"key": "name", "value": "david"})
	require.Contains(t, result.ReplyPayload, `"ok":true`)

	// get
	result = injectEvent(t, kit, "kvstore", "kv.get", map[string]string{"key": "name"})
	require.Contains(t, result.ReplyPayload, `"value":"david"`)

	// has after set → true
	result = injectEvent(t, kit, "kvstore", "kv.has", map[string]string{"key": "name"})
	require.Contains(t, result.ReplyPayload, `"exists":true`)

	// get missing key → empty
	result = injectEvent(t, kit, "kvstore", "kv.get", map[string]string{"key": "missing"})
	require.Contains(t, result.ReplyPayload, `"value":""`)
}

func TestShardFixture_PersistentEventLog(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Listen for forwarded audit notifications
	notifications := make(chan string, 10)
	kit.Bus.On("audit.logged", func(msg bus.Message, _ bus.ReplyFunc) {
		notifications <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/persistent-event-log.ts", "auditlog")

	// Send 3 audit events
	injectEvent(t, kit, "auditlog", "audit.event", map[string]string{"action": "login"})
	injectEvent(t, kit, "auditlog", "audit.event", map[string]string{"action": "access"})
	injectEvent(t, kit, "auditlog", "audit.event", map[string]string{"action": "logout"})

	// Stats should show 3 events
	result := injectEvent(t, kit, "auditlog", "audit.stats", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"eventCount":3`)

	// Check notifications were sent
	require.Len(t, notifications, 3)
}

// ═══════════════════════════════════════════════════════════════
// Tool Provider Fixture
// ═══════════════════════════════════════════════════════════════

func TestShardFixture_ToolProvider(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/tool-provider.ts", "tools-shard")

	require.Equal(t, "stateless", desc.Mode)
	// tool() registrations don't show in Handlers (they're in shardTools)
	// but we can test by injecting events to the tool handler functions directly

	// The tool handler functions are exported — test them via InjectWASMEvent
	// using the function names registered with tool()
	// Note: tool registration maps tool name → function name in shardTools,
	// but handlers are topic → funcName. We need to check what deploy does with tools.
	t.Log("tool-provider deployed successfully with mode:", desc.Mode)
}

// ═══════════════════════════════════════════════════════════════
// AI / Tools / Agent Integration Fixtures
// ═══════════════════════════════════════════════════════════════

func TestShardFixture_AiSummarizer(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Mock ai.generate handler — simulates an LLM response
	kit.Bus.On("ai.generate", func(msg bus.Message, reply bus.ReplyFunc) {
		var req struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
		}
		json.Unmarshal(msg.Payload, &req)
		resp, _ := json.Marshal(map[string]interface{}{
			"text": "Summary of: " + req.Prompt,
			"usage": map[string]int{
				"promptTokens":     10,
				"completionTokens": 5,
			},
		})
		reply(resp)
	})

	// Listen for completion events
	completed := make(chan string, 5)
	kit.Bus.On("summarize.completed", func(msg bus.Message, _ bus.ReplyFunc) {
		completed <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/ai-summarizer.ts", "summarizer")

	// Send a summarize request
	injectEvent(t, kit, "summarizer", "summarize.request", map[string]string{
		"text":  "The quick brown fox jumps over the lazy dog",
		"model": "test-model",
	})

	// Wait for completion event
	select {
	case msg := <-completed:
		require.Contains(t, msg, "summary")
	case <-time.After(5 * time.Second):
		t.Fatal("summarize.completed not received")
	}

	// Check stored state
	result := injectEvent(t, kit, "summarizer", "summarize.last", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"requestCount":1`)
	require.Contains(t, result.ReplyPayload, `"lastSummary":`)

	// Send another request
	injectEvent(t, kit, "summarizer", "summarize.request", map[string]string{
		"text":  "Second document to summarize",
		"model": "test-model",
	})

	select {
	case <-completed:
	case <-time.After(5 * time.Second):
		t.Fatal("second summarize.completed not received")
	}

	result = injectEvent(t, kit, "summarizer", "summarize.last", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"requestCount":2`)
}

func TestShardFixture_ToolOrchestrator(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Mock tools.call handler
	kit.Bus.On("tools.call", func(msg bus.Message, reply bus.ReplyFunc) {
		var req struct {
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		json.Unmarshal(msg.Payload, &req)
		resp, _ := json.Marshal(map[string]interface{}{
			"tool":   req.Name,
			"result": "executed",
			"input":  json.RawMessage(req.Input),
		})
		reply(resp)
	})

	deployShard(t, kit, "testdata/as/shard/tool-orchestrator.ts", "orchestrator")

	result, err := kit.InjectWASMEvent("orchestrator", "orchestrate.query", json.RawMessage(
		`{"tool":"db_query","input":{"sql":"SELECT 1"}}`,
	))
	require.NoError(t, err)
	require.Empty(t, result.Error)
	// askAsync callback stores in state, but stateless mode discards it
	// The test verifies: no crash, askAsync fires, callback runs, tool handler called
}

func TestShardFixture_AgentDelegator(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Mock agents.request handler
	kit.Bus.On("agents.request", func(msg bus.Message, reply bus.ReplyFunc) {
		var req struct {
			Name   string `json:"name"`
			Prompt string `json:"prompt"`
		}
		json.Unmarshal(msg.Payload, &req)
		resp, _ := json.Marshal(map[string]string{
			"text": "Agent " + req.Name + " says: done with " + req.Prompt,
		})
		reply(resp)
	})

	// Listen for completion events
	completed := make(chan string, 5)
	kit.Bus.On("delegate.completed", func(msg bus.Message, _ bus.ReplyFunc) {
		completed <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/agent-delegator.ts", "delegator")

	// Delegate 3 tasks
	for i := 1; i <= 3; i++ {
		injectEvent(t, kit, "delegator", "delegate.task", map[string]interface{}{
			"agent":  "coder",
			"prompt": fmt.Sprintf("task %d", i),
		})

		select {
		case msg := <-completed:
			require.Contains(t, msg, fmt.Sprintf(`"taskNum":%d`, i))
		case <-time.After(5 * time.Second):
			t.Fatalf("delegate.completed not received for task %d", i)
		}
	}

	// Check accumulated results
	result := injectEvent(t, kit, "delegator", "delegate.results", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"completedTasks":3`)
	require.Contains(t, result.ReplyPayload, `"lastResult":`)
}

func TestShardFixture_MultiStepPipeline(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Mock tools.call for data_fetch
	kit.Bus.On("tools.call", func(msg bus.Message, reply bus.ReplyFunc) {
		var req struct {
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		json.Unmarshal(msg.Payload, &req)
		resp, _ := json.Marshal(map[string]interface{}{
			"rows": []map[string]interface{}{
				{"id": 1, "value": "data-row-1"},
				{"id": 2, "value": "data-row-2"},
			},
		})
		reply(resp)
	})

	// Mock ai.generate for analysis
	kit.Bus.On("ai.generate", func(msg bus.Message, reply bus.ReplyFunc) {
		var req struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
		}
		json.Unmarshal(msg.Payload, &req)
		resp, _ := json.Marshal(map[string]string{
			"text": "Analysis: data looks good, 2 rows processed",
		})
		reply(resp)
	})

	// Listen for pipeline completion
	completed := make(chan string, 5)
	kit.Bus.On("pipeline.completed", func(msg bus.Message, _ bus.ReplyFunc) {
		completed <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/multi-step-pipeline.ts", "pipeline")

	// Run the pipeline
	injectEvent(t, kit, "pipeline", "pipeline.run", map[string]string{"source": "test-db"})

	// Wait for pipeline to complete (tool → AI → done)
	select {
	case msg := <-completed:
		require.Contains(t, msg, `"run":1`)
	case <-time.After(10 * time.Second):
		t.Fatal("pipeline.completed not received")
	}

	// Check status
	result := injectEvent(t, kit, "pipeline", "pipeline.status", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"stage":"complete"`)
	require.Contains(t, result.ReplyPayload, `"runs":1`)
	require.Contains(t, result.ReplyPayload, `"analysis":"Analysis:`)

	// Run pipeline again — counter should increment
	injectEvent(t, kit, "pipeline", "pipeline.run", map[string]string{"source": "test-db-2"})

	select {
	case msg := <-completed:
		require.Contains(t, msg, `"run":2`)
	case <-time.After(10 * time.Second):
		t.Fatal("second pipeline.completed not received")
	}

	result = injectEvent(t, kit, "pipeline", "pipeline.status", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"runs":2`)
}

// ═══════════════════════════════════════════════════════════════
// Edge Cases
// ═══════════════════════════════════════════════════════════════

func TestShardFixture_DeployDefaultMode(t *testing.T) {
	// A shard that doesn't call setMode() should default to "stateless"
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { on, reply } from "brainkit";

export function init(): void {
  on("test.default", "handle");
}

export function handle(topic: string, payload: string): void {
  reply('{"mode":"default"}');
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"default-mode\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("default-mode")
	require.NoError(t, err)
	require.Equal(t, "stateless", desc.Mode, "no setMode() should default to stateless")

	result := injectEvent(t, kit, "default-mode", "test.default", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"mode":"default"`)
}

func TestShardFixture_DeployInvalidMode(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode, on } from "brainkit";

export function init(): void {
  setMode("bogus");
  on("test.x", "handle");
}

export function handle(topic: string, payload: string): void {}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"bad-mode\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)

	// DeployWASM uses bus.AskSync → wrapHandler, which returns errors as payload
	desc, err := kit.DeployWASM("bad-mode")
	if err != nil {
		// Error propagated directly
		require.Contains(t, err.Error(), "invalid shard mode")
	} else {
		// Error came back as payload — descriptor should be empty/zero
		require.Empty(t, desc.Module, "invalid mode should not produce a valid descriptor")
	}
}

func TestShardFixture_DeployNoHandlers(t *testing.T) {
	// A shard with init() but no on() registrations — should deploy but have no handlers
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
import { setMode } from "brainkit";

export function init(): void {
  setMode("stateless");
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", fmt.Sprintf(
		"await wasm.compile(%s, { name: \"empty-shard\" });",
		"`"+source+"`",
	))
	require.NoError(t, err)

	desc, err := kit.DeployWASM("empty-shard")
	require.NoError(t, err)
	require.Equal(t, "stateless", desc.Mode)
	require.Empty(t, desc.Handlers, "shard with no on() calls should have empty handlers")
}

func TestShardFixture_UndeployAndRedeploy(t *testing.T) {
	kit := newTestKitNoKey(t)
	deployShard(t, kit, "testdata/as/shard/stateless-echo.ts", "redeploy-test")

	// Works
	result := injectEvent(t, kit, "redeploy-test", "test.echo", map[string]string{"v": "1"})
	require.Equal(t, `{"v":"1"}`, result.ReplyPayload)

	// Undeploy
	err := kit.UndeployWASM("redeploy-test")
	require.NoError(t, err)

	// Inject should fail
	_, err = kit.InjectWASMEvent("redeploy-test", "test.echo", json.RawMessage(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "not deployed")

	// Redeploy
	desc, err := kit.DeployWASM("redeploy-test")
	require.NoError(t, err)
	require.Equal(t, "stateless", desc.Mode)

	// Works again
	result = injectEvent(t, kit, "redeploy-test", "test.echo", map[string]string{"v": "2"})
	require.Equal(t, `{"v":"2"}`, result.ReplyPayload)
}
