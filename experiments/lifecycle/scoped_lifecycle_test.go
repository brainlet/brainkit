//go:build experiment

// Experiment: Scoped Lifecycle with Automatic Cleanup Hooks
//
// Simulates the REAL kit_runtime.js patterns:
// - agent() creates agents with bus subscriptions
// - createTool() registers tools
// - bus.subscribe() tracks subscriptions
// - createMemory() creates memory instances
// - scope() wraps a block with automatic cleanup tracking
// - teardown(source) runs ALL cleanup hooks for a source file
//
// This is the "cheated wrapper" approach: developers write normal code,
// the wrapper silently registers cleanup hooks, teardown is one call.
package lifecycle

import (
	"fmt"
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

// setupKitSimulator creates a JS environment that mirrors kit_runtime.js
// with automatic cleanup hooks on every creation function.
func setupKitSimulator(ctx *quickjs.Context) {
	eval(ctx, `
		// ══════════════════════════════════════════════
		// Resource Registry with Cleanup Hooks
		// (upgraded version of _resourceRegistry)
		// ══════════════════════════════════════════════
		globalThis.__registry = {
			entries: {},
			cleanups: {},

			register: function(type, id, ref, cleanupFn) {
				var key = type + ":" + id;
				// Idempotent: if re-registering same id, run old cleanup first
				if (this.cleanups[key]) {
					this.cleanups[key]();
				}
				this.entries[key] = {
					type: type,
					id: id,
					ref: ref,
					source: globalThis.__kit_current_source || "unknown",
				};
				if (cleanupFn) {
					this.cleanups[key] = cleanupFn;
				}
			},

			teardown: function(source) {
				var removed = 0;
				var keysToRemove = [];
				for (var key in this.entries) {
					if (this.entries[key].source === source) {
						keysToRemove.push(key);
					}
				}
				// Run cleanups in reverse order (LIFO — last created, first destroyed)
				for (var i = keysToRemove.length - 1; i >= 0; i--) {
					var key = keysToRemove[i];
					if (this.cleanups[key]) {
						try { this.cleanups[key](); } catch(e) { /* log but don't stop */ }
						delete this.cleanups[key];
					}
					delete this.entries[key];
					removed++;
				}
				return removed;
			},

			total: function() { return Object.keys(this.entries).length; },
			listBySource: function(source) {
				var result = [];
				for (var key in this.entries) {
					if (this.entries[key].source === source) result.push(this.entries[key]);
				}
				return result;
			},
		};

		// ══════════════════════════════════════════════
		// Simulated Bus (mirrors real bus behavior)
		// ══════════════════════════════════════════════
		globalThis.__bus = {
			_subs: {},
			_counter: 0,

			subscribe: function(topic, handler) {
				var id = "sub_" + (++this._counter);
				var source = globalThis.__kit_current_source || "unknown";
				this._subs[id] = { topic: topic, handler: handler, source: source };

				// AUTO-REGISTER cleanup — the "cheat"
				__registry.register("subscription", id, null, function() {
					delete __bus._subs[id];
				});

				return id;
			},

			unsubscribe: function(id) {
				delete this._subs[id];
			},

			publish: function(topic, data) {
				var delivered = 0;
				for (var id in this._subs) {
					var sub = this._subs[id];
					if (sub.topic === topic || (sub.topic.endsWith(".*") && topic.startsWith(sub.topic.slice(0, -1)))) {
						try { sub.handler(data); delivered++; } catch(e) {}
					}
				}
				return delivered;
			},

			subCount: function() { return Object.keys(this._subs).length; },
		};

		// ══════════════════════════════════════════════
		// Simulated Agent Registry
		// ══════════════════════════════════════════════
		globalThis.__agents = {};
		globalThis.__agent_registry = {};  // like the Go-side agent registry

		// ══════════════════════════════════════════════
		// Simulated Tool Registry
		// ══════════════════════════════════════════════
		globalThis.__tools = {};

		// ══════════════════════════════════════════════
		// Simulated Memory Store
		// ══════════════════════════════════════════════
		globalThis.__memories = {};
		globalThis.__memory_counter = 0;

		// ══════════════════════════════════════════════
		// Creation Functions (mirror kit_runtime.js)
		// Each one registers a cleanup hook automatically.
		// ══════════════════════════════════════════════

		globalThis.agent = function(config) {
			var name = config.name || ("agent_" + Date.now());
			var model = config.model || "default";

			// Create the agent object
			var agentObj = {
				name: name,
				model: model,
				generate: function(prompt) {
					return "response from " + name + " to: " + prompt;
				},
			};

			// Register in agent maps
			__agents[name] = agentObj;
			__agent_registry[name] = {
				name: name,
				model: model,
				status: "idle",
				capabilities: config.tools ? config.tools.map(function(t) { return t.name || t; }) : [],
			};

			// Subscribe to agent messages (like the real agent registry does)
			var msgSubId = __bus.subscribe("agents.message." + name, function(msg) {
				// handle incoming messages to this agent
			});

			// AUTO-REGISTER with full cleanup
			__registry.register("agent", name, agentObj, function cleanup() {
				delete __agents[name];
				delete __agent_registry[name];
				__bus.unsubscribe(msgSubId);
			});

			return agentObj;
		};

		globalThis.createTool = function(config) {
			var id = config.id || config.name;

			var toolObj = {
				id: id,
				description: config.description || "",
				execute: config.execute,
			};

			__tools[id] = toolObj;

			// AUTO-REGISTER with cleanup
			__registry.register("tool", id, toolObj, function cleanup() {
				delete __tools[id];
			});

			return toolObj;
		};

		globalThis.createMemory = function(config) {
			var id = "mem_" + (++__memory_counter);

			var threads = {};
			var threadCounter = 0;

			var memObj = {
				id: id,
				createThread: function(opts) {
					var tid = "thread_" + (++threadCounter);
					threads[tid] = { id: tid, messages: [], opts: opts || {} };
					return threads[tid];
				},
				save: function(threadId, messages) {
					if (threads[threadId]) {
						threads[threadId].messages = threads[threadId].messages.concat(messages);
					}
				},
				recall: function(threadId) {
					return threads[threadId] ? threads[threadId].messages : [];
				},
				close: function() {
					threads = {};
				},
			};

			__memories[id] = memObj;

			// AUTO-REGISTER with cleanup
			__registry.register("memory", id, memObj, function cleanup() {
				memObj.close();
				delete __memories[id];
			});

			return memObj;
		};

		// ══════════════════════════════════════════════
		// scope() — THE composition primitive
		// Wraps a block of code with automatic source tracking.
		// Returns an object with .teardown() to destroy everything.
		// ══════════════════════════════════════════════

		globalThis.scope = function(name, fn) {
			var prevSource = globalThis.__kit_current_source;
			globalThis.__kit_current_source = name;
			try {
				var result = fn();
				return {
					result: result,
					source: name,
					teardown: function() {
						return __registry.teardown(name);
					},
					resources: function() {
						return __registry.listBySource(name);
					},
				};
			} finally {
				globalThis.__kit_current_source = prevSource;
			}
		};
	`)
}

// ═══════════════════════════════════════════════════════════════
// Test 1: Agent creation auto-registers cleanup
// ═══════════════════════════════════════════════════════════════

func TestScoped_AgentAutoCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	// Create an agent — cleanup should be auto-registered
	eval(ctx, `
		globalThis.__kit_current_source = "my-agents.ts";
		var coder = agent({ name: "coder", model: "gpt-4o-mini" });
		globalThis.__kit_current_source = null;
	`)

	// Verify agent exists
	if evalStr(ctx, `__agents["coder"].name`) != "coder" {
		t.Fatal("agent not created")
	}
	if evalStr(ctx, `__agent_registry["coder"].status`) != "idle" {
		t.Fatal("agent not in registry")
	}
	if evalInt(ctx, `__bus.subCount()`) != 1 {
		t.Fatal("expected 1 bus subscription")
	}

	// Teardown the source file
	removed := evalInt(ctx, `__registry.teardown("my-agents.ts")`)
	if removed != 2 { // 1 agent + 1 subscription
		t.Fatalf("expected 2 removed (agent + sub), got %d", removed)
	}

	// Verify EVERYTHING is gone
	if evalStr(ctx, `typeof __agents["coder"]`) != "undefined" {
		t.Fatal("agent should be gone from __agents")
	}
	if evalStr(ctx, `typeof __agent_registry["coder"]`) != "undefined" {
		t.Fatal("agent should be gone from registry")
	}
	if evalInt(ctx, `__bus.subCount()`) != 0 {
		t.Fatal("bus subscription should be gone")
	}

	t.Log("PASS: agent auto-cleanup removes agent + registry entry + bus subscription")
}

// ═══════════════════════════════════════════════════════════════
// Test 2: Tool creation auto-registers cleanup
// ═══════════════════════════════════════════════════════════════

func TestScoped_ToolAutoCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	eval(ctx, `
		globalThis.__kit_current_source = "tools.ts";
		createTool({ id: "search", description: "Search the web", execute: async function() {} });
		createTool({ id: "calculator", description: "Math", execute: async function(a, b) { return a + b; } });
		globalThis.__kit_current_source = null;
	`)

	if evalInt(ctx, `Object.keys(__tools).length`) != 2 {
		t.Fatal("expected 2 tools")
	}

	evalInt(ctx, `__registry.teardown("tools.ts")`)

	if evalInt(ctx, `Object.keys(__tools).length`) != 0 {
		t.Fatal("tools should be gone")
	}

	t.Log("PASS: tool auto-cleanup removes from tool registry")
}

// ═══════════════════════════════════════════════════════════════
// Test 3: Memory creation auto-registers cleanup (with close)
// ═══════════════════════════════════════════════════════════════

func TestScoped_MemoryAutoCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	eval(ctx, `
		globalThis.__kit_current_source = "memory.ts";
		var mem = createMemory({});
		var thread = mem.createThread({ title: "test" });
		mem.save(thread.id, [{ role: "user", content: "hello" }]);
		globalThis.__test_mem_id = mem.id;
		globalThis.__kit_current_source = null;
	`)

	// Memory exists and has data
	memId := evalStr(ctx, `__test_mem_id`)
	messages := evalInt(ctx, fmt.Sprintf(`__memories[%q].recall(__memories[%q].createThread({}).id).length`, memId, memId))
	_ = messages // just verify it doesn't crash

	// Teardown
	evalInt(ctx, `__registry.teardown("memory.ts")`)

	if evalStr(ctx, fmt.Sprintf(`typeof __memories[%q]`, memId)) != "undefined" {
		t.Fatal("memory should be gone")
	}

	t.Log("PASS: memory auto-cleanup closes and removes memory instance")
}

// ═══════════════════════════════════════════════════════════════
// Test 4: Bus subscriptions auto-tracked and cleaned up
// ═══════════════════════════════════════════════════════════════

func TestScoped_BusSubscriptionAutoCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	eval(ctx, `
		globalThis.__kit_current_source = "listeners.ts";
		__bus.subscribe("events.*", function(data) { /* handle */ });
		__bus.subscribe("orders.new", function(data) { /* handle */ });
		__bus.subscribe("alerts.*", function(data) { /* handle */ });
		globalThis.__kit_current_source = null;
	`)

	if evalInt(ctx, `__bus.subCount()`) != 3 {
		t.Fatal("expected 3 subscriptions")
	}

	// Verify events deliver
	if evalInt(ctx, `__bus.publish("events.test", "hello")`) != 1 {
		t.Fatal("expected 1 delivery")
	}

	// Teardown
	evalInt(ctx, `__registry.teardown("listeners.ts")`)

	if evalInt(ctx, `__bus.subCount()`) != 0 {
		t.Fatal("all subscriptions should be gone")
	}

	// Verify nothing delivers
	if evalInt(ctx, `__bus.publish("events.test", "hello")`) != 0 {
		t.Fatal("should deliver to nobody after teardown")
	}

	t.Log("PASS: bus subscriptions auto-tracked, all cleaned on teardown")
}

// ═══════════════════════════════════════════════════════════════
// Test 5: scope() — THE composition primitive
// ═══════════════════════════════════════════════════════════════

func TestScoped_ScopeComposition(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	eval(ctx, `
		globalThis.team = scope("team-alpha", function() {
			var leader = agent({ name: "leader", model: "gpt-4o" });
			var coder = agent({ name: "coder", model: "gpt-4o-mini" });
			var reviewer = agent({ name: "reviewer", model: "gpt-4o-mini" });

			createTool({ id: "lint", description: "Run linter", execute: async function() {} });

			__bus.subscribe("team.events.*", function(data) { /* handle */ });

			return { leader: leader, coder: coder, reviewer: reviewer };
		});
	`)

	// Verify everything created
	if evalInt(ctx, `Object.keys(__agents).length`) != 3 {
		t.Fatal("expected 3 agents")
	}
	if evalInt(ctx, `Object.keys(__tools).length`) != 1 {
		t.Fatal("expected 1 tool")
	}
	// 3 agent message subs + 1 explicit sub = 4
	if evalInt(ctx, `__bus.subCount()`) != 4 {
		t.Fatalf("expected 4 subs, got %d", evalInt(ctx, `__bus.subCount()`))
	}

	// Use the agents
	gen := evalStr(ctx, `team.result.leader.generate("write code")`)
	if gen != "response from leader to: write code" {
		t.Fatalf("unexpected: %s", gen)
	}

	// Check resources tracked
	resourceCount := evalInt(ctx, `team.resources().length`)
	t.Logf("Resources in scope: %d", resourceCount)

	// TEARDOWN — one call
	removed := evalInt(ctx, `team.teardown()`)
	t.Logf("Removed: %d resources", removed)

	// Verify EVERYTHING is gone
	if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
		t.Fatal("agents should be gone")
	}
	if evalInt(ctx, `Object.keys(__tools).length`) != 0 {
		t.Fatal("tools should be gone")
	}
	if evalInt(ctx, `__bus.subCount()`) != 0 {
		t.Fatal("subscriptions should be gone")
	}
	if evalInt(ctx, `Object.keys(__agent_registry).length`) != 0 {
		t.Fatal("agent registry should be empty")
	}

	t.Log("PASS: scope() creates agents+tools+subs, teardown() destroys everything in one call")
}

// ═══════════════════════════════════════════════════════════════
// Test 6: Nested scopes — inner teardown doesn't affect outer
// ═══════════════════════════════════════════════════════════════

func TestScoped_NestedScopes(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	eval(ctx, `
		globalThis.outer = scope("outer-team", function() {
			var supervisor = agent({ name: "supervisor", model: "gpt-4o" });

			globalThis.inner = scope("inner-workers", function() {
				var w1 = agent({ name: "worker-1", model: "gpt-4o-mini" });
				var w2 = agent({ name: "worker-2", model: "gpt-4o-mini" });
				return { w1: w1, w2: w2 };
			});

			return { supervisor: supervisor };
		});
	`)

	if evalInt(ctx, `Object.keys(__agents).length`) != 3 {
		t.Fatal("expected 3 agents total")
	}

	// Teardown inner — supervisor survives
	evalInt(ctx, `inner.teardown()`)

	if evalInt(ctx, `Object.keys(__agents).length`) != 1 {
		t.Fatalf("expected 1 agent after inner teardown, got %d", evalInt(ctx, `Object.keys(__agents).length`))
	}
	if evalStr(ctx, `typeof __agents["supervisor"]`) == "undefined" {
		t.Fatal("supervisor should survive inner teardown")
	}
	if evalStr(ctx, `typeof __agents["worker-1"]`) != "undefined" {
		t.Fatal("worker-1 should be gone")
	}

	// Teardown outer — supervisor goes too
	evalInt(ctx, `outer.teardown()`)

	if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
		t.Fatal("all agents should be gone")
	}

	t.Log("PASS: nested scopes — inner teardown preserves outer resources")
}

// ═══════════════════════════════════════════════════════════════
// Test 7: Redeploy scope — teardown old, create new
// ═══════════════════════════════════════════════════════════════

func TestScoped_RedeployScope(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	// Deploy v1
	eval(ctx, `
		globalThis.deployment = scope("my-service", function() {
			return { a: agent({ name: "service-agent", model: "gpt-4o" }) };
		});
	`)

	gen := evalStr(ctx, `deployment.result.a.generate("v1")`)
	if gen != "response from service-agent to: v1" {
		t.Fatal("v1 should work")
	}

	// Redeploy: teardown + create new
	eval(ctx, `
		deployment.teardown();
		globalThis.deployment = scope("my-service", function() {
			return { a: agent({ name: "service-agent-v2", model: "gpt-4o-mini" }) };
		});
	`)

	// Only 1 agent (not 2)
	if evalInt(ctx, `Object.keys(__agents).length`) != 1 {
		t.Fatal("expected 1 agent after redeploy")
	}

	gen = evalStr(ctx, `deployment.result.a.generate("v2")`)
	if gen != "response from service-agent-v2 to: v2" {
		t.Fatal("v2 should work")
	}

	// Clean final teardown
	evalInt(ctx, `deployment.teardown()`)
	if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
		t.Fatal("should be clean after final teardown")
	}

	t.Log("PASS: redeploy — teardown old scope, create new, no leaks")
}

// ═══════════════════════════════════════════════════════════════
// Test 8: Scope with mixed resources (agent + tool + memory + sub)
// ═══════════════════════════════════════════════════════════════

func TestScoped_MixedResources(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	eval(ctx, `
		globalThis.app = scope("full-app", function() {
			var a = agent({ name: "assistant", model: "gpt-4o" });
			var search = createTool({ id: "search", description: "Search", execute: async function() {} });
			var calc = createTool({ id: "calc", description: "Calculate", execute: async function() {} });
			var mem = createMemory({});
			__bus.subscribe("notifications.*", function(d) {});
			__bus.subscribe("errors.*", function(d) {});
			return { agent: a, tools: [search, calc], memory: mem };
		});
	`)

	// Verify all created
	if evalInt(ctx, `Object.keys(__agents).length`) != 1 {
		t.Fatal("expected 1 agent")
	}
	if evalInt(ctx, `Object.keys(__tools).length`) != 2 {
		t.Fatal("expected 2 tools")
	}
	if evalInt(ctx, `Object.keys(__memories).length`) != 1 {
		t.Fatal("expected 1 memory")
	}
	// 1 agent msg sub + 2 explicit subs = 3
	if evalInt(ctx, `__bus.subCount()`) != 3 {
		t.Fatalf("expected 3 subs, got %d", evalInt(ctx, `__bus.subCount()`))
	}

	resources := evalInt(ctx, `app.resources().length`)
	t.Logf("Total tracked resources: %d", resources)

	// One call destroys everything
	removed := evalInt(ctx, `app.teardown()`)
	t.Logf("Removed: %d", removed)

	if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
		t.Fatal("agents leak")
	}
	if evalInt(ctx, `Object.keys(__tools).length`) != 0 {
		t.Fatal("tools leak")
	}
	if evalInt(ctx, `Object.keys(__memories).length`) != 0 {
		t.Fatal("memories leak")
	}
	if evalInt(ctx, `__bus.subCount()`) != 0 {
		t.Fatal("subs leak")
	}

	t.Log("PASS: mixed resources (agent + tools + memory + subs) — all cleaned in one teardown")
}

// ═══════════════════════════════════════════════════════════════
// Test 9: Stress — 50 scope create/teardown cycles
// ═══════════════════════════════════════════════════════════════

func TestScoped_StressCycles(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupKitSimulator(ctx)

	const cycles = 50

	for i := range cycles {
		scopeName := fmt.Sprintf("cycle-%d", i)
		eval(ctx, fmt.Sprintf(`
			globalThis.__scope = scope(%q, function() {
				agent({ name: "a1", model: "m" });
				agent({ name: "a2", model: "m" });
				createTool({ id: "t1", execute: async function() {} });
				createMemory({});
				__bus.subscribe("events.*", function() {});
			});
		`, scopeName))

		// Verify created
		agents := evalInt(ctx, `Object.keys(__agents).length`)
		if agents != 2 {
			t.Fatalf("cycle %d: expected 2 agents, got %d", i, agents)
		}

		// Teardown
		evalInt(ctx, `__scope.teardown()`)

		// Verify clean
		if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
			t.Fatalf("cycle %d: agent leak", i)
		}
		if evalInt(ctx, `Object.keys(__tools).length`) != 0 {
			t.Fatalf("cycle %d: tool leak", i)
		}
		if evalInt(ctx, `Object.keys(__memories).length`) != 0 {
			t.Fatalf("cycle %d: memory leak", i)
		}
		if evalInt(ctx, `__bus.subCount()`) != 0 {
			t.Fatalf("cycle %d: sub leak", i)
		}

		if i%10 == 0 {
			rt.RunGC()
		}
	}

	rt.RunGC()
	t.Logf("PASS: %d scope create/teardown cycles, zero leaks", cycles)
}
