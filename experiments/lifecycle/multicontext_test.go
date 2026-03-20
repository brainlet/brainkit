// Experiment: Multiple QuickJS Contexts for file isolation.
//
// Each deployed .ts file gets its own Context within the same Runtime.
// Contexts share the heap/GC but have completely separate globals.
// Bridge functions (Go callbacks) are registered per context.
// Resources register in Go-side registries (shared across contexts).
//
// This gives us:
// - True isolation (no global collision, strict mode compatible)
// - Shared Go infrastructure (bus, tool registry, agent registry)
// - Clean teardown (ctx.Close() frees everything in that context)
// - No hacks (no with, no Proxy, no snapshot)
package lifecycle

import (
	"fmt"
	"sync"
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

// GoRegistry simulates the Go-side shared registries (tool, agent, bus).
// All contexts register into this shared state.
type GoRegistry struct {
	mu     sync.Mutex
	agents map[string]*GoAgent
	tools  map[string]*GoTool
	subs   map[string]*GoSub
	subSeq int
}

type GoAgent struct {
	Name   string
	Model  string
	Source string
}

type GoTool struct {
	ID     string
	Source string
}

type GoSub struct {
	ID     string
	Topic  string
	Source string
}

func NewGoRegistry() *GoRegistry {
	return &GoRegistry{
		agents: make(map[string]*GoAgent),
		tools:  make(map[string]*GoTool),
		subs:   make(map[string]*GoSub),
	}
}

func (r *GoRegistry) RegisterAgent(name, model, source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agents[name] = &GoAgent{Name: name, Model: model, Source: source}
}

func (r *GoRegistry) UnregisterAgent(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.agents, name)
}

func (r *GoRegistry) RegisterTool(id, source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[id] = &GoTool{ID: id, Source: source}
}

func (r *GoRegistry) Subscribe(topic, source string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.subSeq++
	id := fmt.Sprintf("sub_%d", r.subSeq)
	r.subs[id] = &GoSub{ID: id, Topic: topic, Source: source}
	return id
}

func (r *GoRegistry) Unsubscribe(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.subs, id)
}

func (r *GoRegistry) TeardownSource(source string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	removed := 0
	for k, v := range r.agents {
		if v.Source == source { delete(r.agents, k); removed++ }
	}
	for k, v := range r.tools {
		if v.Source == source { delete(r.tools, k); removed++ }
	}
	for k, v := range r.subs {
		if v.Source == source { delete(r.subs, k); removed++ }
	}
	return removed
}

func (r *GoRegistry) AgentCount() int { r.mu.Lock(); defer r.mu.Unlock(); return len(r.agents) }
func (r *GoRegistry) ToolCount() int  { r.mu.Lock(); defer r.mu.Unlock(); return len(r.tools) }
func (r *GoRegistry) SubCount() int   { r.mu.Lock(); defer r.mu.Unlock(); return len(r.subs) }

// registerBridges sets up Go bridge functions on a QuickJS context.
// Each context gets its own bridge registration pointing to the shared Go registry.
func registerBridges(ctx *quickjs.Context, reg *GoRegistry, source string) {
	// agent() bridge
	ctx.Globals().Set("__go_register_agent", ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		name := args[0].ToString()
		model := args[1].ToString()
		reg.RegisterAgent(name, model, source)
		return ctx.Bool(true)
	}))

	// createTool() bridge
	ctx.Globals().Set("__go_register_tool", ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		id := args[0].ToString()
		reg.RegisterTool(id, source)
		return ctx.Bool(true)
	}))

	// bus.subscribe() bridge
	ctx.Globals().Set("__go_subscribe", ctx.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		topic := args[0].ToString()
		subId := reg.Subscribe(topic, source)
		return ctx.String(subId)
	}))

	// Inject a mini kit API
	eval(ctx, `
		globalThis.agent = function(config) {
			__go_register_agent(config.name, config.model || "default");
			return { name: config.name, model: config.model };
		};
		globalThis.createTool = function(config) {
			__go_register_tool(config.id);
			return { id: config.id };
		};
		globalThis.subscribe = function(topic, handler) {
			return __go_subscribe(topic);
		};
	`)
}

// ═══════════════════════════════════════════════════════════════
// Test 1: Two contexts — complete global isolation
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_GlobalIsolation(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	reg := NewGoRegistry()

	ctxA := rt.NewContext()
	defer ctxA.Close()
	registerBridges(ctxA, reg, "file-a.ts")

	ctxB := rt.NewContext()
	defer ctxB.Close()
	registerBridges(ctxB, reg, "file-b.ts")

	// Both files use same variable names — zero collision
	eval(ctxA, `
		var config = { url: "https://a.com" };
		var API_KEY = "key-A";
		var myAgent = agent({ name: "agent-a", model: "gpt-4o" });
	`)

	eval(ctxB, `
		var config = { url: "https://b.com" };
		var API_KEY = "key-B";
		var myAgent = agent({ name: "agent-b", model: "gpt-4o-mini" });
	`)

	// Each context sees its own values
	if evalStr(ctxA, `config.url`) != "https://a.com" {
		t.Fatal("A should see its own config")
	}
	if evalStr(ctxB, `config.url`) != "https://b.com" {
		t.Fatal("B should see its own config")
	}
	if evalStr(ctxA, `API_KEY`) != "key-A" {
		t.Fatal("A should see its own API_KEY")
	}
	if evalStr(ctxB, `API_KEY`) != "key-B" {
		t.Fatal("B should see its own API_KEY")
	}

	// Both agents registered in shared Go registry
	if reg.AgentCount() != 2 {
		t.Fatalf("expected 2 agents in Go registry, got %d", reg.AgentCount())
	}

	t.Log("PASS: complete global isolation — same var names, different values, shared Go registry")
}

// ═══════════════════════════════════════════════════════════════
// Test 2: const/let work naturally (no with/proxy needed)
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_ConstLetWork(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	ctxA := rt.NewContext()
	defer ctxA.Close()
	registerBridges(ctxA, reg, "a.ts")

	// const and let just work — they're context-scoped
	eval(ctxA, `
		const SECRET = "immutable";
		let counter = 0;
		counter++;
	`)

	if evalStr(ctxA, `SECRET`) != "immutable" {
		t.Fatal("const should work")
	}
	if evalInt(ctxA, `counter`) != 1 {
		t.Fatal("let should work")
	}

	// Another context can't see them
	ctxB := rt.NewContext()
	defer ctxB.Close()

	if evalStr(ctxB, `typeof SECRET`) != "undefined" {
		t.Fatal("B should not see A's const")
	}

	t.Log("PASS: const/let work naturally in separate contexts")
}

// ═══════════════════════════════════════════════════════════════
// Test 3: Strict mode works (no with restriction)
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_StrictModeWorks(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctx := rt.NewContext()
	defer ctx.Close()

	result := evalStr(ctx, `
		"use strict";
		const x = 42;
		"" + x;
	`)

	if result != "42" {
		t.Fatalf("strict mode should work, got %s", result)
	}

	t.Log("PASS: strict mode works naturally in separate contexts (no with needed)")
}

// ═══════════════════════════════════════════════════════════════
// Test 4: Teardown = close context + cleanup Go registry
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_TeardownIsContextClose(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	// Deploy
	ctx := rt.NewContext()
	registerBridges(ctx, reg, "team.ts")
	eval(ctx, `
		agent({ name: "leader", model: "gpt-4o" });
		agent({ name: "coder", model: "gpt-4o-mini" });
		createTool({ id: "search" });
		subscribe("events.*");
	`)

	if reg.AgentCount() != 2 { t.Fatal("expected 2 agents") }
	if reg.ToolCount() != 1 { t.Fatal("expected 1 tool") }
	if reg.SubCount() != 1 { t.Fatal("expected 1 sub") }

	// Teardown = close context + cleanup Go registry
	ctx.Close() // JS objects freed
	removed := reg.TeardownSource("team.ts") // Go registry cleaned

	if removed != 4 {
		t.Fatalf("expected 4 removed, got %d", removed)
	}
	if reg.AgentCount() != 0 { t.Fatal("agents should be gone") }
	if reg.ToolCount() != 0 { t.Fatal("tools should be gone") }
	if reg.SubCount() != 0 { t.Fatal("subs should be gone") }

	// Runtime is still alive — can create new contexts
	ctx2 := rt.NewContext()
	defer ctx2.Close()
	val := evalStr(ctx2, `"still alive"`)
	if val != "still alive" {
		t.Fatal("runtime should survive context close")
	}

	t.Log("PASS: teardown = ctx.Close() + registry cleanup. Runtime survives.")
}

// ═══════════════════════════════════════════════════════════════
// Test 5: Multiple files — isolated teardown
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_IsolatedTeardown(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	// Deploy 3 files — track for manual close (defer closes in LIFO which can crash)
	ctxA := rt.NewContext()
	registerBridges(ctxA, reg, "team-a.ts")
	eval(ctxA, `agent({ name: "a1" }); agent({ name: "a2" });`)

	ctxB := rt.NewContext()
	registerBridges(ctxB, reg, "team-b.ts")
	eval(ctxB, `agent({ name: "b1" }); createTool({ id: "b-tool" });`)

	ctxC := rt.NewContext()
	registerBridges(ctxC, reg, "team-c.ts")
	eval(ctxC, `agent({ name: "c1" }); subscribe("events.*");`)

	if reg.AgentCount() != 4 { t.Fatalf("expected 4 agents, got %d", reg.AgentCount()) }

	// Teardown only B
	ctxB.Close()
	reg.TeardownSource("team-b.ts")

	if reg.AgentCount() != 3 { t.Fatalf("expected 3 agents after B teardown, got %d", reg.AgentCount()) }
	if reg.ToolCount() != 0 { t.Fatal("B's tool should be gone") }

	// A and C still work — use simple eval (no bridge calls to avoid potential GC issues)
	valA := evalStr(ctxA, `"a-alive"`)
	valC := evalStr(ctxC, `"c-alive"`)
	if valA != "a-alive" { t.Fatal("A should still work") }
	if valC != "c-alive" { t.Fatal("C should still work") }

	// Cleanup remaining (close in reverse creation order)
	ctxC.Close()
	reg.TeardownSource("team-c.ts")
	ctxA.Close()
	reg.TeardownSource("team-a.ts")

	if reg.AgentCount() != 0 { t.Fatal("all agents should be gone") }

	t.Log("PASS: isolated teardown — close one context, others unaffected")
}

// ═══════════════════════════════════════════════════════════════
// Test 6: Context can access shared Go tools via bridge
// (simulates cross-file tool calls going through Go)
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_CrossFileViaBridge(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	// File A registers a tool
	ctxA := rt.NewContext()
	defer ctxA.Close()
	registerBridges(ctxA, reg, "provider.ts")
	eval(ctxA, `createTool({ id: "calculator" });`)

	// File B calls the tool via Go bridge (simulated)
	ctxB := rt.NewContext()
	defer ctxB.Close()
	registerBridges(ctxB, reg, "consumer.ts")

	// Register a bridge that checks Go registry
	ctxB.Globals().Set("__go_call_tool", ctxB.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		toolId := args[0].ToString()
		reg.mu.Lock()
		_, exists := reg.tools[toolId]
		reg.mu.Unlock()
		if exists {
			return ctx.String("result from " + toolId)
		}
		return ctx.String("tool not found: " + toolId)
	}))

	result := evalStr(ctxB, `__go_call_tool("calculator")`)
	if result != "result from calculator" {
		t.Fatalf("expected tool result, got %s", result)
	}

	// B can't directly access A's JS objects (complete isolation)
	// But it CAN call A's tools through Go — that's the bus pattern
	t.Log("PASS: cross-file communication works through Go bridges (not shared JS objects)")
}

// ═══════════════════════════════════════════════════════════════
// Test 7: Bridge registration cost — is it fast enough?
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_BridgeRegistrationPerformance(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	const N = 50

	for i := range N {
		ctx := rt.NewContext()
		registerBridges(ctx, reg, fmt.Sprintf("file-%d.ts", i))
		eval(ctx, fmt.Sprintf(`agent({ name: "agent-%d" });`, i))
		ctx.Close()
		reg.TeardownSource(fmt.Sprintf("file-%d.ts", i))
	}

	if reg.AgentCount() != 0 {
		t.Fatal("leak detected")
	}

	t.Logf("PASS: %d context create/register/eval/close cycles completed", N)
}

// ═══════════════════════════════════════════════════════════════
// Test 8: Stress — many contexts alive simultaneously
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_ManySimultaneousContexts(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	const N = 20
	contexts := make([]*quickjs.Context, N)

	// Create N contexts
	for i := range N {
		contexts[i] = rt.NewContext()
		registerBridges(contexts[i], reg, fmt.Sprintf("ctx-%d.ts", i))
		eval(contexts[i], fmt.Sprintf(`
			agent({ name: "agent-%d" });
			createTool({ id: "tool-%d" });
		`, i, i))
	}

	if reg.AgentCount() != N { t.Fatalf("expected %d agents, got %d", N, reg.AgentCount()) }
	if reg.ToolCount() != N { t.Fatalf("expected %d tools, got %d", N, reg.ToolCount()) }

	// Each context still works independently
	for i := range N {
		val := evalStr(contexts[i], fmt.Sprintf(`"ctx-%d-alive"`, i))
		if val != fmt.Sprintf("ctx-%d-alive", i) {
			t.Fatalf("context %d not responding", i)
		}
	}

	// Teardown all in reverse
	for i := N - 1; i >= 0; i-- {
		contexts[i].Close()
		reg.TeardownSource(fmt.Sprintf("ctx-%d.ts", i))
	}

	if reg.AgentCount() != 0 || reg.ToolCount() != 0 {
		t.Fatal("leak after teardown all")
	}

	t.Logf("PASS: %d simultaneous contexts, all isolated, all torn down cleanly", N)
}

// ═══════════════════════════════════════════════════════════════
// Test 9: Context with closures and callbacks
// Closures in one context can't capture state from another
// ═══════════════════════════════════════════════════════════════

func TestMultiCtx_ClosureIsolation(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	defer ctxA.Close()
	ctxB := rt.NewContext()
	defer ctxB.Close()

	eval(ctxA, `
		var secret = "A-secret";
		globalThis.getSecret = function() { return secret; };
	`)

	eval(ctxB, `
		var secret = "B-secret";
		globalThis.getSecret = function() { return secret; };
	`)

	aSecret := evalStr(ctxA, `getSecret()`)
	bSecret := evalStr(ctxB, `getSecret()`)

	if aSecret != "A-secret" { t.Fatalf("A's closure wrong: %s", aSecret) }
	if bSecret != "B-secret" { t.Fatalf("B's closure wrong: %s", bSecret) }

	t.Log("PASS: closures are isolated between contexts — each retains its own scope")
}
