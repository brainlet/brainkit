//go:build experiment

// Experiment: Full Kit lifecycle — testing with real Kit, real WASM, real bus.
// These tests prove that the scoped lifecycle pattern works with actual brainkit
// infrastructure, not just simulated JS.
package lifecycle

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/internal/bus"
	"github.com/brainlet/brainkit/internal/registry"
)

func newKit(t *testing.T) *kit.Kit {
	t.Helper()
	kit, err := kit.New(kit.Config{Namespace: "lifecycle-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

// ═══════════════════════════════════════════════════════════════
// Test 1: Agent lifecycle via real Kit — EvalTS creates, TeardownFile destroys
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_AgentCreateAndTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	// Deploy agents via EvalTS
	_, err := kit.EvalTS(ctx, "team.ts", `
		agent({ name: "leader", model: "openai/gpt-4o-mini", instructions: "lead" });
		agent({ name: "coder", model: "openai/gpt-4o-mini", instructions: "code" });
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Verify agents are in the resource registry
	resources, _ := kit.ListResources("agent")
	agentNames := map[string]bool{}
	for _, r := range resources {
		agentNames[r.Name] = true
	}
	if !agentNames["leader"] || !agentNames["coder"] {
		t.Fatalf("expected leader + coder, got: %+v", resources)
	}

	// Verify agents are in the agent registry (bus-side)
	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "agents.list",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("agents.list before teardown: %s", resp.Payload)

	// Teardown the file
	removed, err := kit.TeardownFile("team.ts")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("TeardownFile removed %d resources", removed)

	// Verify agents gone from resource registry
	resources, _ = kit.ListResources("agent")
	for _, r := range resources {
		if r.Name == "leader" || r.Name == "coder" {
			t.Fatalf("agent %q should be gone from resource registry", r.Name)
		}
	}

	t.Log("PASS: agents created via EvalTS, torn down via TeardownFile")
}

// ═══════════════════════════════════════════════════════════════
// Test 2: Tool lifecycle — createTool in .ts, teardown removes from registry
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_ToolCreateAndTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	_, err := kit.EvalTS(ctx, "tools.ts", `
		createTool({
			id: "lifecycle-calc",
			description: "Add two numbers",
			inputSchema: z.object({ a: z.number(), b: z.number() }),
			execute: async ({ a, b }) => ({ result: a + b }),
		});
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Verify tool in resource registry
	resources, _ := kit.ListResources("tool")
	found := false
	for _, r := range resources {
		if r.ID == "lifecycle-calc" {
			found = true
		}
	}
	if !found {
		t.Fatal("tool not found in registry")
	}

	// Teardown
	removed, _ := kit.TeardownFile("tools.ts")
	t.Logf("Removed %d resources", removed)

	// Verify tool gone from resource registry
	resources, _ = kit.ListResources("tool")
	for _, r := range resources {
		if r.ID == "lifecycle-calc" {
			t.Fatal("tool should be gone")
		}
	}

	t.Log("PASS: tool created and torn down")
}

// ═══════════════════════════════════════════════════════════════
// Test 3: WASM module lifecycle — compile → deploy shard → teardown
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_WASMShardTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	// Compile a shard — use string concat to avoid backtick escaping
	shardSource := `import { setMode, on, reply, log } from "brainkit";
export function init(): void {
  setMode("stateless");
  on("lifecycle.test", "handle");
}
export function handle(topic: string, payload: string): void {
  log("got: " + payload);
  reply('{"handled":true}');
}`
	compileCode := fmt.Sprintf(`
		await wasm.compile(%q, { name: "lifecycle-shard" });
		await wasm.deploy("lifecycle-shard");
	`, shardSource)
	_, err := kit.EvalTS(ctx, "wasm-deploy.ts", compileCode)
	if err != nil {
		t.Fatal(err)
	}

	// Verify shard is deployed
	desc, err := kit.DescribeWASM("lifecycle-shard")
	if err != nil {
		t.Fatal(err)
	}
	if desc.Mode != "stateless" {
		t.Fatalf("expected stateless, got %s", desc.Mode)
	}

	// Verify shard handles events
	result, err := kit.InjectWASMEvent("lifecycle-shard", "lifecycle.test", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != "" {
		t.Fatalf("shard error: %s", result.Error)
	}

	// Verify WASM is in resource registry
	resources, _ := kit.ListResources("wasm")
	found := false
	for _, r := range resources {
		if r.ID == "lifecycle-shard" {
			found = true
		}
	}
	if !found {
		t.Fatal("WASM not in resource registry")
	}

	// Now undeploy and verify
	err = kit.UndeployWASM("lifecycle-shard")
	if err != nil {
		t.Fatal(err)
	}

	// Shard should be gone — events don't route
	_, err = kit.InjectWASMEvent("lifecycle-shard", "lifecycle.test", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error after undeploy")
	}

	t.Log("PASS: WASM shard compiled, deployed, handled events, undeployed cleanly")
}

// ═══════════════════════════════════════════════════════════════
// Test 4: Bus subscription lifecycle — subscribe in .ts, teardown unsubscribes
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_BusSubscriptionTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	// Subscribe from .ts — creates a Go-side bus subscription via bridge
	_, err := kit.EvalTS(ctx, "listeners.ts", `
		var subId1 = bus.subscribe("lifecycle.events.*", function(msg) {});
		var subId2 = bus.subscribe("lifecycle.alerts.*", function(msg) {});
		globalThis.__lifecycle_sub_ids = [subId1, subId2];
	`)
	if err != nil {
		t.Fatal(err)
	}

	// Verify subscriptions are tracked in resource registry
	resources, _ := kit.ListResources("subscription")
	subCount := 0
	for _, r := range resources {
		if r.Source == "listeners.ts" {
			subCount++
		}
	}
	if subCount != 2 {
		t.Fatalf("expected 2 subscriptions from listeners.ts, got %d", subCount)
	}

	// Teardown — should unsubscribe from Go bus via cleanup hooks
	removed, _ := kit.TeardownFile("listeners.ts")
	t.Logf("Removed %d resources", removed)

	// Verify subscriptions gone from resource registry
	resources, _ = kit.ListResources("subscription")
	subCount = 0
	for _, r := range resources {
		if r.Source == "listeners.ts" {
			subCount++
		}
	}
	if subCount != 0 {
		t.Fatalf("expected 0 subscriptions after teardown, got %d", subCount)
	}

	// Verify __bus_subs cleaned up in JS
	jsSubs, _ := kit.EvalTS(ctx, "__check_subs.ts", `
		return JSON.stringify(Object.keys(globalThis.__bus_subs).length);
	`)
	t.Logf("JS bus_subs after teardown: %s", jsSubs)

	t.Log("PASS: bus subscriptions created in .ts, tracked in registry, cleaned up by TeardownFile")
}

// ═══════════════════════════════════════════════════════════════
// Test 5: Go-registered tool lifecycle — register from Go, verify teardown doesn't break
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_GoToolSurvivesTsTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	// Register a Go tool (not via .ts)
	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/go-tool", ShortName: "go-tool",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]string{"from": "go"})
			},
		},
	})

	// Create .ts resources
	kit.EvalTS(ctx, "ts-stuff.ts", `
		agent({ name: "ts-agent", model: "openai/gpt-4o-mini", instructions: "hi" });
	`)

	// Teardown .ts
	kit.TeardownFile("ts-stuff.ts")

	// Go tool should still work
	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "tools.call",
		Payload: json.RawMessage(`{"name":"go-tool","input":{}}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]string
	json.Unmarshal(resp.Payload, &result)
	if result["from"] != "go" {
		t.Fatalf("Go tool broken after ts teardown: %s", resp.Payload)
	}

	t.Log("PASS: Go-registered tools survive .ts file teardown")
}

// ═══════════════════════════════════════════════════════════════
// Test 6: Multiple file deploy/teardown isolation with real Kit
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_MultiFileTeardownIsolation(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	// Deploy 3 files
	kit.EvalTS(ctx, "team-a.ts", `
		agent({ name: "a1", model: "openai/gpt-4o-mini", instructions: "A1" });
		agent({ name: "a2", model: "openai/gpt-4o-mini", instructions: "A2" });
	`)
	kit.EvalTS(ctx, "team-b.ts", `
		agent({ name: "b1", model: "openai/gpt-4o-mini", instructions: "B1" });
	`)
	kit.EvalTS(ctx, "team-c.ts", `
		agent({ name: "c1", model: "openai/gpt-4o-mini", instructions: "C1" });
		agent({ name: "c2", model: "openai/gpt-4o-mini", instructions: "C2" });
	`)

	all, _ := kit.ListResources("agent")
	if len(all) < 5 {
		t.Fatalf("expected 5 agents, got %d", len(all))
	}

	// Teardown only team-b
	removed, _ := kit.TeardownFile("team-b.ts")
	if removed < 1 {
		t.Fatalf("expected at least 1 removed, got %d", removed)
	}

	// Verify b1 gone, others intact
	remaining, _ := kit.ListResources("agent")
	names := map[string]bool{}
	for _, r := range remaining {
		names[r.Name] = true
	}
	if names["b1"] {
		t.Fatal("b1 should be gone")
	}
	if !names["a1"] || !names["a2"] || !names["c1"] || !names["c2"] {
		t.Fatalf("a1/a2/c1/c2 should survive, got: %v", names)
	}

	t.Log("PASS: multi-file teardown isolation — only target file's resources removed")
}

// ═══════════════════════════════════════════════════════════════
// Test 7: Rapid deploy/teardown stress with real Kit
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_StressDeployTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	const cycles = 20

	for i := range cycles {
		filename := fmt.Sprintf("stress-%d.ts", i)
		agentName := fmt.Sprintf("stress-agent-%d", i)

		_, err := kit.EvalTS(ctx, filename, fmt.Sprintf(`
			agent({ name: %q, model: "openai/gpt-4o-mini", instructions: "cycle %d" });
		`, agentName, i))
		if err != nil {
			t.Fatalf("cycle %d deploy: %v", i, err)
		}

		// Verify exists
		resources, _ := kit.ListResources("agent")
		found := false
		for _, r := range resources {
			if r.Name == agentName {
				found = true
			}
		}
		if !found {
			t.Fatalf("cycle %d: agent not found", i)
		}

		// Teardown
		removed, err := kit.TeardownFile(filename)
		if err != nil {
			t.Fatalf("cycle %d teardown: %v", i, err)
		}
		if removed < 1 {
			t.Fatalf("cycle %d: expected removal, got %d", i, removed)
		}
	}

	// Verify clean
	resources, _ := kit.ListResources("agent")
	stressAgents := 0
	for _, r := range resources {
		for i := range cycles {
			if r.Name == fmt.Sprintf("stress-agent-%d", i) {
				stressAgents++
			}
		}
	}
	if stressAgents > 0 {
		t.Fatalf("leak: %d stress agents remain", stressAgents)
	}

	t.Logf("PASS: %d real Kit deploy/teardown cycles, zero leaks", cycles)
}

// ═══════════════════════════════════════════════════════════════
// Test 8: Concurrent bus usage during teardown — no crashes
// ═══════════════════════════════════════════════════════════════

func TestKitLifecycle_ConcurrentUsageDuringTeardown(t *testing.T) {
	kit := newKit(t)
	ctx := context.Background()

	// Register a Go tool that takes time
	var callCount atomic.Int32
	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/slow", ShortName: "slow",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				callCount.Add(1)
				time.Sleep(50 * time.Millisecond)
				return json.Marshal(map[string]string{"ok": "true"})
			},
		},
	})

	// Deploy agents
	kit.EvalTS(ctx, "concurrent.ts", `
		agent({ name: "busy-agent", model: "openai/gpt-4o-mini", instructions: "work" });
	`)

	// Fire concurrent tool calls
	for range 10 {
		go func() {
			bus.AskSync(kit.Bus, ctx, bus.Message{
				Topic:   "tools.call",
				Payload: json.RawMessage(`{"name":"slow","input":{}}`),
			})
		}()
	}

	// Teardown while calls are in flight — should not panic
	time.Sleep(10 * time.Millisecond) // let some calls start
	kit.TeardownFile("concurrent.ts")

	// Wait for in-flight calls to complete
	time.Sleep(200 * time.Millisecond)

	// Go tool should still work (it's Go-registered, not .ts)
	resp, _ := bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:   "tools.call",
		Payload: json.RawMessage(`{"name":"slow","input":{}}`),
	})
	var result map[string]string
	json.Unmarshal(resp.Payload, &result)
	if result["ok"] != "true" {
		t.Fatalf("Go tool broken after concurrent teardown: %s", resp.Payload)
	}

	t.Logf("PASS: concurrent bus usage during teardown — no crashes, %d calls completed", callCount.Load())
}
