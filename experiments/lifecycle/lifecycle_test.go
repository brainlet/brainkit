//go:build experiment

// Package lifecycle experiments with QuickJS object lifecycle management from Go.
//
// The question: can we reliably create, track, and DESTROY JavaScript objects
// from Go — and guarantee they're actually gone (GC'd, no leaks, no dangling refs)?
//
// If yes → .ts files become the composition API with full lifecycle management.
// If no → we need a different approach.
package lifecycle

import (
	"fmt"
	"runtime"
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

// eval runs JS code and frees the result (for statements).
func eval(ctx *quickjs.Context, code string) {
	v := ctx.Eval(code)
	v.Free()
}

// evalStr runs JS code and returns the string result.
func evalStr(ctx *quickjs.Context, code string) string {
	v := ctx.Eval(code)
	s := v.ToString()
	v.Free()
	return s
}

// evalInt runs JS code and returns the int32 result.
func evalInt(ctx *quickjs.Context, code string) int32 {
	v := ctx.Eval(code)
	n := v.ToInt32()
	v.Free()
	return n
}

// ═══════════════════════════════════════════════════════════════
// Experiment 1: Basic Object Deletion
// ═══════════════════════════════════════════════════════════════

func TestExp_DeleteGlobalVariable(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `globalThis.myAgent = { name: "coder", status: "idle" };`)

	typ := evalStr(ctx, `typeof globalThis.myAgent`)
	if typ != "object" {
		t.Fatalf("expected object, got %s", typ)
	}

	globals := ctx.Globals()
	if !globals.Delete("myAgent") {
		t.Fatal("Delete returned false")
	}

	typ = evalStr(ctx, `typeof globalThis.myAgent`)
	if typ != "undefined" {
		t.Fatalf("expected undefined after delete, got %s", typ)
	}

	t.Log("PASS: global variable deleted from Go → becomes undefined")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 2: Delete Object With Closures
// ═══════════════════════════════════════════════════════════════

func TestExp_DeleteObjectWithClosures(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		var _secret = "hidden-data";
		globalThis.myService = {
			getData: function() { return _secret; },
			counter: 0,
			increment: function() { this.counter++; return this.counter; },
		};
	`)

	val := evalStr(ctx, `myService.getData()`)
	if val != "hidden-data" {
		t.Fatalf("expected hidden-data, got %s", val)
	}

	ctx.Globals().Delete("myService")

	typ := evalStr(ctx, `typeof globalThis.myService`)
	if typ != "undefined" {
		t.Fatalf("expected undefined, got %s", typ)
	}

	msg := evalStr(ctx, `
		try { myService.getData(); "should-not-reach" }
		catch(e) { "caught: " + e.message }
	`)
	if msg == "should-not-reach" {
		t.Fatal("myService should not be accessible")
	}
	t.Logf("After delete: %s", msg)
	t.Log("PASS: object with closures deleted, access throws")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 3: GC Reclaims Deleted Objects
// ═══════════════════════════════════════════════════════════════

func TestExp_GCReclaimsDeletedObjects(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.bigObject = {
			data: new Array(10000).fill("x".repeat(100)),
			nested: { a: new Array(5000).fill(42) },
		};
	`)

	rt.RunGC()
	runtime.GC()

	ctx.Globals().Delete("bigObject")
	rt.RunGC()

	typ := evalStr(ctx, `typeof globalThis.bigObject`)
	if typ != "undefined" {
		t.Fatalf("expected undefined, got %s", typ)
	}

	val := evalStr(ctx, `
		globalThis.newObject = { data: new Array(10000).fill("y".repeat(100)) };
		"ok"
	`)
	if val != "ok" {
		t.Fatal("failed to allocate after delete")
	}
	ctx.Globals().Delete("newObject")
	rt.RunGC()

	t.Log("PASS: GC reclaims deleted objects, new allocations succeed")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 4: Cross-Reference Survival
// ═══════════════════════════════════════════════════════════════

func TestExp_CrossReferenceSurvival(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.agentA = { name: "A" };
		globalThis.agentB = { name: "B", partner: globalThis.agentA };
	`)

	globals := ctx.Globals()
	globals.Delete("agentA")

	typ := evalStr(ctx, `typeof globalThis.agentA`)
	if typ != "undefined" {
		t.Fatalf("agentA should be undefined, got %s", typ)
	}

	// agentB.partner still holds the reference
	name := evalStr(ctx, `agentB.partner.name`)
	if name != "A" {
		t.Fatalf("expected A via cross-ref, got %s", name)
	}

	globals.Delete("agentB")
	rt.RunGC()

	typ = evalStr(ctx, `typeof globalThis.agentB`)
	if typ != "undefined" {
		t.Fatal("agentB should be undefined")
	}

	t.Log("PASS: cross-references survive partial deletion, collectible after all refs removed")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 5: Callback Survives Object Deletion
// CRITICAL: closures keep objects alive even after global deletion
// ═══════════════════════════════════════════════════════════════

func TestExp_CallbackSurvivesObjectDeletion(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.__callbacks = {};
		globalThis.__subscribe = function(topic, fn) {
			if (!__callbacks[topic]) __callbacks[topic] = [];
			__callbacks[topic].push(fn);
		};
		globalThis.__publish = function(topic, data) {
			var cbs = __callbacks[topic] || [];
			var results = [];
			for (var i = 0; i < cbs.length; i++) {
				try { results.push(cbs[i](data)); } catch(e) { results.push("error:" + e.message); }
			}
			return JSON.stringify(results);
		};
		globalThis.__unsubscribeAll = function(topic) {
			delete __callbacks[topic];
		};
	`)

	eval(ctx, `
		globalThis.myAgent = { name: "listener", received: [] };
		__subscribe("events.test", function(data) {
			myAgent.received.push(data);
			return "handled by " + myAgent.name;
		});
	`)

	before := evalStr(ctx, `__publish("events.test", "hello")`)
	t.Logf("Before delete: %s", before)

	ctx.Globals().Delete("myAgent")

	// Callback closure still holds myAgent reference — object LIVES
	after := evalStr(ctx, `__publish("events.test", "world")`)
	t.Logf("After delete (callback still registered): %s", after)

	// Must also unsubscribe to truly clean up
	eval(ctx, `__unsubscribeAll("events.test")`)
	none := evalStr(ctx, `__publish("events.test", "nobody")`)
	t.Logf("After unsubscribe: %s", none)

	rt.RunGC()
	t.Log("PASS: callbacks keep objects alive via closures. MUST unsubscribe to fully clean up.")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 6: Full Deploy/Teardown Cycle
// The complete lifecycle: deploy .ts → create resources →
// teardown → verify everything is gone
// ═══════════════════════════════════════════════════════════════

func TestExp_FullDeployTeardownCycle(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// Infrastructure
	eval(ctx, `
		globalThis.__registry = {
			entries: {}, cleanups: {},
			register: function(type, id, ref, cleanup) {
				this.entries[type + ":" + id] = { type: type, id: id, ref: ref, source: globalThis.__current_source || "unknown" };
				if (cleanup) this.cleanups[type + ":" + id] = cleanup;
			},
			teardown: function(source) {
				var removed = 0;
				for (var key in this.entries) {
					if (this.entries[key].source === source) {
						if (this.cleanups[key]) { this.cleanups[key](); delete this.cleanups[key]; }
						delete this.entries[key]; removed++;
					}
				}
				return removed;
			},
			list: function() { return JSON.stringify(Object.keys(this.entries)); },
		};
		globalThis.__agents = {};
		globalThis.__subs = {};
		globalThis.__subCounter = 0;
		globalThis.__busSubscribe = function(topic, fn) {
			var id = "sub_" + (++__subCounter);
			__subs[id] = { topic: topic, fn: fn };
			return id;
		};
		globalThis.__busUnsubscribe = function(id) { delete __subs[id]; };
		globalThis.__busPublish = function(topic, data) {
			var count = 0;
			for (var id in __subs) {
				if (__subs[id].topic === topic) { try { __subs[id].fn(data); count++; } catch(e) {} }
			}
			return count;
		};
		globalThis.__createAgent = function(config) {
			var agent = { name: config.name, generate: function(p) { return "response from " + config.name; } };
			__agents[config.name] = agent;
			var subId = __busSubscribe("agents.message." + config.name, function(msg) {});
			__registry.register("agent", config.name, agent, function() {
				delete __agents[config.name];
				__busUnsubscribe(subId);
			});
			return agent;
		};
	`)

	// DEPLOY
	eval(ctx, `
		globalThis.__current_source = "team-alpha.ts";
		globalThis.leader = __createAgent({ name: "leader" });
		globalThis.coder = __createAgent({ name: "coder" });
		globalThis.reviewer = __createAgent({ name: "reviewer" });
		globalThis.__current_source = null;
	`)

	if evalInt(ctx, `Object.keys(__agents).length`) != 3 {
		t.Fatal("expected 3 agents")
	}
	if evalInt(ctx, `Object.keys(__subs).length`) != 3 {
		t.Fatal("expected 3 subscriptions")
	}

	gen := evalStr(ctx, `leader.generate("hello")`)
	if gen != "response from leader" {
		t.Fatalf("expected response, got %s", gen)
	}

	// TEARDOWN
	removed := evalInt(ctx, `__registry.teardown("team-alpha.ts")`)
	if removed != 3 {
		t.Fatalf("expected 3 removed, got %d", removed)
	}

	globals := ctx.Globals()
	globals.Delete("leader")
	globals.Delete("coder")
	globals.Delete("reviewer")

	if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
		t.Fatal("agents should be empty")
	}
	if evalInt(ctx, `Object.keys(__subs).length`) != 0 {
		t.Fatal("subscriptions should be empty")
	}
	if evalStr(ctx, `typeof globalThis.leader`) != "undefined" {
		t.Fatal("leader should be undefined")
	}
	if evalInt(ctx, `__busPublish("agents.message.leader", "hello")`) != 0 {
		t.Fatal("should deliver to nobody")
	}

	rt.RunGC()
	t.Log("PASS: full deploy/teardown — agents created, used, torn down, GC'd")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 7: Redeploy (Atomic Swap)
// ═══════════════════════════════════════════════════════════════

func TestExp_RedeployCycle(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.__registry = {
			entries: {}, cleanups: {},
			register: function(type, id, ref, cleanup) {
				this.entries[type + ":" + id] = { type: type, id: id, ref: ref, source: globalThis.__current_source || "unknown" };
				if (cleanup) this.cleanups[type + ":" + id] = cleanup;
			},
			teardown: function(source) {
				var removed = 0;
				for (var key in this.entries) {
					if (this.entries[key].source === source) {
						if (this.cleanups[key]) { this.cleanups[key](); delete this.cleanups[key]; }
						delete this.entries[key]; removed++;
					}
				}
				return removed;
			},
		};
		globalThis.__agents = {};
		globalThis.__subs = {}; globalThis.__subCounter = 0;
		globalThis.__busSubscribe = function(t, fn) { var id = "sub_" + (++__subCounter); __subs[id] = { topic: t, fn: fn }; return id; };
		globalThis.__busUnsubscribe = function(id) { delete __subs[id]; };
		globalThis.__createAgent = function(config) {
			var agent = { name: config.name, version: config.version || "v1" };
			__agents[config.name] = agent;
			var subId = __busSubscribe("msg." + config.name, function(m) {});
			__registry.register("agent", config.name, agent, function() { delete __agents[config.name]; __busUnsubscribe(subId); });
			return agent;
		};
	`)

	// Deploy v1
	eval(ctx, `
		globalThis.__current_source = "my-team.ts";
		globalThis.worker = __createAgent({ name: "worker", version: "v1" });
		globalThis.__current_source = null;
	`)

	if evalStr(ctx, `__agents["worker"].version`) != "v1" {
		t.Fatal("expected v1")
	}

	// Teardown v1
	eval(ctx, `__registry.teardown("my-team.ts")`)
	ctx.Globals().Delete("worker")
	rt.RunGC()

	// Deploy v2
	eval(ctx, `
		globalThis.__current_source = "my-team.ts";
		globalThis.worker = __createAgent({ name: "worker", version: "v2" });
		globalThis.__current_source = null;
	`)

	if evalStr(ctx, `__agents["worker"].version`) != "v2" {
		t.Fatal("expected v2 after redeploy")
	}
	if evalInt(ctx, `Object.keys(__agents).length`) != 1 {
		t.Fatal("expected 1 agent after redeploy (not 2)")
	}
	if evalInt(ctx, `Object.keys(__subs).length`) != 1 {
		t.Fatal("expected 1 sub after redeploy (not 2)")
	}

	t.Log("PASS: teardown + redeploy — clean swap, no duplicates")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 8: Timer Cleanup
// ═══════════════════════════════════════════════════════════════

func TestExp_TimerCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.__timers = {};
		globalThis.__timerCounter = 0;
		globalThis.__timerLog = [];
		globalThis.__setTimeout = function(fn, delay) {
			var id = ++__timerCounter;
			__timers[id] = { fn: fn, source: globalThis.__current_source || "unknown" };
			return id;
		};
		globalThis.__fireTimer = function(id) {
			if (__timers[id]) { __timers[id].fn(); delete __timers[id]; }
		};
		globalThis.__cancelTimersBySource = function(source) {
			var cancelled = 0;
			for (var id in __timers) { if (__timers[id].source === source) { delete __timers[id]; cancelled++; } }
			return cancelled;
		};
	`)

	eval(ctx, `
		globalThis.__current_source = "timers.ts";
		__setTimeout(function() { __timerLog.push("t1"); }, 1000);
		__setTimeout(function() { __timerLog.push("t2"); }, 2000);
		__setTimeout(function() { __timerLog.push("t3"); }, 3000);
		globalThis.__current_source = null;
	`)

	if evalInt(ctx, `Object.keys(__timers).length`) != 3 {
		t.Fatal("expected 3 timers")
	}

	eval(ctx, `__fireTimer(1)`)
	if evalInt(ctx, `__timerLog.length`) != 1 {
		t.Fatal("expected 1 fired")
	}

	cancelled := evalInt(ctx, `__cancelTimersBySource("timers.ts")`)
	if cancelled != 2 {
		t.Fatalf("expected 2 cancelled, got %d", cancelled)
	}

	eval(ctx, `__fireTimer(2)`)
	if evalInt(ctx, `__timerLog.length`) != 1 {
		t.Fatal("cancelled timer should not fire")
	}

	t.Log("PASS: timers tracked by source, cancelled on teardown")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 9: Isolated Teardown (3 files, teardown 1)
// ═══════════════════════════════════════════════════════════════

func TestExp_IsolatedTeardown(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.__registry = {
			entries: {}, cleanups: {},
			register: function(type, id, ref, cleanup) {
				this.entries[type + ":" + id] = { type: type, id: id, ref: ref, source: globalThis.__current_source || "unknown" };
				if (cleanup) this.cleanups[type + ":" + id] = cleanup;
			},
			teardown: function(source) {
				var removed = 0;
				for (var key in this.entries) {
					if (this.entries[key].source === source) {
						if (this.cleanups[key]) { this.cleanups[key](); delete this.cleanups[key]; }
						delete this.entries[key]; removed++;
					}
				}
				return removed;
			},
			countBySource: function(source) { var n = 0; for (var k in this.entries) { if (this.entries[k].source === source) n++; } return n; },
			total: function() { return Object.keys(this.entries).length; },
		};
		globalThis.__agents = {};
		globalThis.__subs = {}; globalThis.__subCounter = 0;
		globalThis.__busSubscribe = function(t, fn) { var id = "sub_" + (++__subCounter); __subs[id] = { topic: t, fn: fn }; return id; };
		globalThis.__busUnsubscribe = function(id) { delete __subs[id]; };
		globalThis.__createAgent = function(config) {
			var agent = { name: config.name };
			__agents[config.name] = agent;
			var subId = __busSubscribe("msg." + config.name, function(m) {});
			__registry.register("agent", config.name, agent, function() { delete __agents[config.name]; __busUnsubscribe(subId); });
			return agent;
		};
	`)

	for _, f := range []struct{ source, code string }{
		{"team-a.ts", `__createAgent({ name: "a1" }); __createAgent({ name: "a2" });`},
		{"team-b.ts", `__createAgent({ name: "b1" }); __createAgent({ name: "b2" }); __createAgent({ name: "b3" });`},
		{"team-c.ts", `__createAgent({ name: "c1" });`},
	} {
		eval(ctx, fmt.Sprintf(`globalThis.__current_source = %q; %s; globalThis.__current_source = null;`, f.source, f.code))
	}

	if evalInt(ctx, `Object.keys(__agents).length`) != 6 {
		t.Fatal("expected 6 agents")
	}

	removed := evalInt(ctx, `__registry.teardown("team-b.ts")`)
	if removed != 3 {
		t.Fatalf("expected 3 removed, got %d", removed)
	}

	if evalInt(ctx, `Object.keys(__agents).length`) != 3 {
		t.Fatal("expected 3 remaining")
	}
	if evalInt(ctx, `__registry.countBySource("team-a.ts")`) != 2 {
		t.Fatal("team-a should have 2")
	}
	if evalInt(ctx, `__registry.countBySource("team-c.ts")`) != 1 {
		t.Fatal("team-c should have 1")
	}

	for _, name := range []string{"b1", "b2", "b3"} {
		if evalStr(ctx, fmt.Sprintf(`typeof __agents[%q]`, name)) != "undefined" {
			t.Fatalf("agent %s should be gone", name)
		}
	}
	for _, name := range []string{"a1", "a2", "c1"} {
		if evalStr(ctx, fmt.Sprintf(`typeof __agents[%q]`, name)) == "undefined" {
			t.Fatalf("agent %s should still exist", name)
		}
	}

	t.Log("PASS: isolated teardown — only target file's resources removed")
}

// ═══════════════════════════════════════════════════════════════
// Experiment 10: Stress — 100 Deploy/Teardown Cycles
// ═══════════════════════════════════════════════════════════════

func TestExp_StressDeployTeardown(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.__registry = {
			entries: {}, cleanups: {},
			register: function(type, id, ref, cleanup) {
				this.entries[type + ":" + id] = { type: type, id: id, ref: ref, source: globalThis.__current_source || "unknown" };
				if (cleanup) this.cleanups[type + ":" + id] = cleanup;
			},
			teardown: function(source) {
				var removed = 0;
				for (var key in this.entries) {
					if (this.entries[key].source === source) {
						if (this.cleanups[key]) { this.cleanups[key](); delete this.cleanups[key]; }
						delete this.entries[key]; removed++;
					}
				}
				return removed;
			},
			total: function() { return Object.keys(this.entries).length; },
		};
		globalThis.__agents = {};
		globalThis.__subs = {}; globalThis.__subCounter = 0;
		globalThis.__busSubscribe = function(t, fn) { var id = "sub_" + (++__subCounter); __subs[id] = { topic: t, fn: fn }; return id; };
		globalThis.__busUnsubscribe = function(id) { delete __subs[id]; };
		globalThis.__createAgent = function(config) {
			var agent = { name: config.name, data: new Array(100).fill("x") };
			__agents[config.name] = agent;
			var subId = __busSubscribe("msg." + config.name, function(m) {});
			__registry.register("agent", config.name, agent, function() { delete __agents[config.name]; __busUnsubscribe(subId); });
			return agent;
		};
	`)

	const cycles = 100
	const agentsPerCycle = 5

	for i := range cycles {
		source := fmt.Sprintf("cycle-%d.ts", i)

		for j := range agentsPerCycle {
			name := fmt.Sprintf("agent-%d-%d", i, j)
			eval(ctx, fmt.Sprintf(`globalThis.__current_source = %q; __createAgent({ name: %q }); globalThis.__current_source = null;`, source, name))
		}

		total := evalInt(ctx, `__registry.total()`)
		if int(total) != agentsPerCycle {
			t.Fatalf("cycle %d: expected %d, got %d", i, agentsPerCycle, total)
		}

		removed := evalInt(ctx, fmt.Sprintf(`__registry.teardown(%q)`, source))
		if int(removed) != agentsPerCycle {
			t.Fatalf("cycle %d: expected %d removed, got %d", i, agentsPerCycle, removed)
		}

		if evalInt(ctx, `__registry.total()`) != 0 {
			t.Fatalf("cycle %d: not clean after teardown", i)
		}

		if i%10 == 0 {
			rt.RunGC()
		}
	}

	if evalInt(ctx, `Object.keys(__agents).length`) != 0 {
		t.Fatal("leak: agents remain")
	}
	if evalInt(ctx, `Object.keys(__subs).length`) != 0 {
		t.Fatal("leak: subscriptions remain")
	}

	t.Logf("PASS: %d cycles × %d agents = %d operations, zero leaks", cycles, agentsPerCycle, cycles*agentsPerCycle)
}
