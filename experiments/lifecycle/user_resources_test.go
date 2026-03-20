// Experiment: User-created resources
//
// Three approaches to handle resources the developer creates
// outside our API surface:
//
// 1. Global snapshot — diff globals before/after eval, auto-delete new ones
// 2. onTeardown() hook — user registers their own cleanup (like React useEffect)
// 3. Timer wrapping — automatically track setTimeout/setInterval by source
package lifecycle

import (
	"fmt"
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

func setupUserResourceInfra(ctx *quickjs.Context) {
	eval(ctx, `
		// ══════════════════════════════════════════
		// Resource registry (from previous experiments)
		// ══════════════════════════════════════════
		globalThis.__registry = {
			entries: {}, cleanups: {},
			register: function(type, id, name, ref, cleanupFn) {
				var key = type + ":" + id;
				if (this.cleanups[key]) { try { this.cleanups[key](); } catch(e) {} }
				this.entries[key] = { type: type, id: id, name: name || id, source: globalThis.__kit_current_source || "unknown", ref: ref };
				if (typeof cleanupFn === "function") this.cleanups[key] = cleanupFn;
			},
			unregister: function(type, id) {
				var key = type + ":" + id;
				if (this.cleanups[key]) { try { this.cleanups[key](); } catch(e) {} delete this.cleanups[key]; }
				delete this.entries[key];
			},
			teardown: function(source) {
				var removed = 0;
				var keys = [];
				for (var k in this.entries) { if (this.entries[k].source === source) keys.push(k); }
				for (var i = keys.length - 1; i >= 0; i--) {
					var k = keys[i];
					if (this.cleanups[k]) { try { this.cleanups[k](); } catch(e) {} delete this.cleanups[k]; }
					delete this.entries[k]; removed++;
				}
				return removed;
			},
			total: function() { return Object.keys(this.entries).length; },
		};

		// ══════════════════════════════════════════
		// Approach 1: Global Snapshot
		// Snapshot globalThis keys before eval, diff after,
		// auto-delete new globals on teardown.
		// ══════════════════════════════════════════
		globalThis.__global_snapshots = {};

		globalThis.__snapshotGlobalsBefore = function(source) {
			var keys = {};
			for (var k in globalThis) keys[k] = true;
			// Also capture Object.keys for non-enumerable awareness
			Object.keys(globalThis).forEach(function(k) { keys[k] = true; });
			__global_snapshots[source] = keys;
		};

		globalThis.__snapshotGlobalsAfter = function(source) {
			var before = __global_snapshots[source] || {};
			var newKeys = [];
			for (var k in globalThis) {
				if (!before[k]) newKeys.push(k);
			}
			Object.keys(globalThis).forEach(function(k) {
				if (!before[k] && newKeys.indexOf(k) < 0) newKeys.push(k);
			});
			// Register cleanup that deletes these globals
			if (newKeys.length > 0) {
				__registry.register("globals", source, source, null, function() {
					for (var i = 0; i < newKeys.length; i++) {
						try { delete globalThis[newKeys[i]]; } catch(e) {}
					}
				});
			}
			delete __global_snapshots[source];
			return newKeys;
		};

		// ══════════════════════════════════════════
		// Approach 2: onTeardown() hook
		// User registers their own cleanup functions.
		// ══════════════════════════════════════════
		globalThis.__teardown_hooks = {};
		globalThis.__teardown_counter = 0;

		globalThis.onTeardown = function(fn) {
			var source = globalThis.__kit_current_source || "unknown";
			var id = "teardown_" + (++__teardown_counter);
			__registry.register("teardown-hook", id, id, null, fn);
		};

		// ══════════════════════════════════════════
		// Approach 3: Timer Wrapping
		// Replace setTimeout/setInterval with tracked versions.
		// ══════════════════════════════════════════
		globalThis.__timers = {};
		globalThis.__timerCounter = 0;

		globalThis.setTimeout = function(fn, delay) {
			var id = ++__timerCounter;
			var source = globalThis.__kit_current_source || "unknown";
			__timers[id] = { fn: fn, source: source, type: "timeout" };
			__registry.register("timer", "t" + id, "t" + id, null, function() {
				delete __timers[id];
			});
			return id;
		};

		globalThis.setInterval = function(fn, interval) {
			var id = ++__timerCounter;
			var source = globalThis.__kit_current_source || "unknown";
			__timers[id] = { fn: fn, source: source, type: "interval" };
			__registry.register("timer", "i" + id, "i" + id, null, function() {
				delete __timers[id];
			});
			return id;
		};

		globalThis.clearTimeout = function(id) {
			delete __timers[id];
			__registry.unregister("timer", "t" + id);
		};

		globalThis.clearInterval = function(id) {
			delete __timers[id];
			__registry.unregister("timer", "i" + id);
		};

		// Helper: simulate firing a timer
		globalThis.__fireTimer = function(id) {
			if (__timers[id]) { __timers[id].fn(); if (__timers[id] && __timers[id].type === "timeout") delete __timers[id]; }
		};

		// ══════════════════════════════════════════
		// Combined deploy/teardown that uses ALL approaches
		// ══════════════════════════════════════════
		globalThis.deploy = function(source, fn) {
			__snapshotGlobalsBefore(source);
			var prevSource = globalThis.__kit_current_source;
			globalThis.__kit_current_source = source;
			try {
				var result = fn();
				var newGlobals = __snapshotGlobalsAfter(source);
				return {
					result: result,
					source: source,
					newGlobals: newGlobals,
					teardown: function() { return __registry.teardown(source); },
				};
			} finally {
				globalThis.__kit_current_source = prevSource;
			}
		};
	`)
}

// ═══════════════════════════════════════════════════════════════
// Test 1: Global Snapshot — auto-detects and cleans user globals
// ═══════════════════════════════════════════════════════════════

func TestUserRes_GlobalSnapshotCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupUserResourceInfra(ctx)

	eval(ctx, `
		globalThis.app = deploy("user-code.ts", function() {
			// User creates arbitrary globals
			globalThis.myCache = { data: {} };
			globalThis.myCounter = 0;
			globalThis.myHelper = function(x) { return x * 2; };
			return "deployed";
		});
	`)

	// Verify user globals exist
	if evalStr(ctx, `typeof globalThis.myCache`) != "object" {
		t.Fatal("myCache should exist")
	}
	if evalStr(ctx, `typeof globalThis.myHelper`) != "function" {
		t.Fatal("myHelper should exist")
	}

	// Check what was detected
	newGlobals := evalStr(ctx, `JSON.stringify(app.newGlobals)`)
	t.Logf("Detected new globals: %s", newGlobals)

	// Teardown — should auto-delete user globals
	removed := evalInt(ctx, `app.teardown()`)
	t.Logf("Removed %d resources", removed)

	// Verify user globals are gone
	if evalStr(ctx, `typeof globalThis.myCache`) != "undefined" {
		t.Fatal("myCache should be cleaned up")
	}
	if evalStr(ctx, `typeof globalThis.myCounter`) != "undefined" {
		t.Fatal("myCounter should be cleaned up")
	}
	if evalStr(ctx, `typeof globalThis.myHelper`) != "undefined" {
		t.Fatal("myHelper should be cleaned up")
	}

	t.Log("PASS: global snapshot detects and cleans user-created globals")
}

// ═══════════════════════════════════════════════════════════════
// Test 2: onTeardown() hook — user registers custom cleanup
// ═══════════════════════════════════════════════════════════════

func TestUserRes_OnTeardownHook(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupUserResourceInfra(ctx)

	eval(ctx, `
		globalThis.__cleanup_log = [];

		globalThis.app = deploy("user-with-hooks.ts", function() {
			// User creates a cache and registers cleanup
			var cache = {};
			cache["key1"] = "value1";
			cache["key2"] = "value2";

			onTeardown(function() {
				__cleanup_log.push("cache cleared");
				for (var k in cache) delete cache[k];
			});

			// User creates a connection and registers cleanup
			var connection = { active: true, socket: "ws://..." };
			onTeardown(function() {
				__cleanup_log.push("connection closed");
				connection.active = false;
				connection.socket = null;
			});

			// Store refs to check later
			globalThis.__test_cache = cache;
			globalThis.__test_conn = connection;

			return "deployed";
		});
	`)

	// Verify resources exist
	if evalStr(ctx, `__test_conn.active`) != "true" {
		t.Fatal("connection should be active")
	}

	// Teardown
	evalInt(ctx, `app.teardown()`)

	// Verify user cleanup hooks ran (check the log, not the objects — globals got cleaned by snapshot)
	log := evalStr(ctx, `JSON.stringify(__cleanup_log)`)
	t.Logf("Cleanup log: %s", log)

	// Both hooks should have fired
	if evalInt(ctx, `__cleanup_log.length`) != 2 {
		t.Fatalf("expected 2 cleanup hooks, got: %s", log)
	}

	// Globals should be cleaned by snapshot
	if evalStr(ctx, `typeof globalThis.__test_conn`) != "undefined" {
		t.Fatal("__test_conn should be deleted by global snapshot")
	}
	if evalStr(ctx, `typeof globalThis.__test_cache`) != "undefined" {
		t.Fatal("__test_cache should be deleted by global snapshot")
	}

	t.Log("PASS: onTeardown() hooks run during teardown + globals auto-cleaned by snapshot")
}

// ═══════════════════════════════════════════════════════════════
// Test 3: Timer wrapping — auto-tracks setTimeout/setInterval
// ═══════════════════════════════════════════════════════════════

func TestUserRes_TimerAutoCleanup(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupUserResourceInfra(ctx)

	eval(ctx, `
		globalThis.__timer_log = [];

		globalThis.app = deploy("timers.ts", function() {
			setTimeout(function() { __timer_log.push("timeout-1"); }, 1000);
			setInterval(function() { __timer_log.push("interval-1"); }, 500);
			setTimeout(function() { __timer_log.push("timeout-2"); }, 2000);
			return "deployed";
		});
	`)

	if evalInt(ctx, `Object.keys(__timers).length`) != 3 {
		t.Fatalf("expected 3 timers, got %d", evalInt(ctx, `Object.keys(__timers).length`))
	}

	// Fire one timer
	eval(ctx, `__fireTimer(1)`)
	if evalInt(ctx, `__timer_log.length`) != 1 {
		t.Fatal("expected 1 fired")
	}

	// Teardown — cancels remaining timers
	evalInt(ctx, `app.teardown()`)

	if evalInt(ctx, `Object.keys(__timers).length`) != 0 {
		t.Fatal("all timers should be cancelled")
	}

	// Try to fire cancelled timer — noop
	eval(ctx, `__fireTimer(2)`)
	if evalInt(ctx, `__timer_log.length`) != 1 {
		t.Fatal("cancelled timer should not fire")
	}

	t.Log("PASS: setTimeout/setInterval auto-tracked and cancelled on teardown")
}

// ═══════════════════════════════════════════════════════════════
// Test 4: All three combined — globals + hooks + timers
// ═══════════════════════════════════════════════════════════════

func TestUserRes_CombinedApproaches(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupUserResourceInfra(ctx)

	eval(ctx, `
		globalThis.__combined_log = [];

		globalThis.app = deploy("full-app.ts", function() {
			// User global (auto-detected by snapshot)
			globalThis.appState = { initialized: true, users: [] };

			// User cleanup hook
			onTeardown(function() {
				__combined_log.push("db closed");
			});

			// User timer (auto-tracked)
			var timerId = setInterval(function() {
				appState.users.push("tick");
			}, 1000);

			// Another cleanup hook
			onTeardown(function() {
				__combined_log.push("cleanup done");
			});

			return { timerId: timerId };
		});
	`)

	// Everything exists
	if evalStr(ctx, `typeof globalThis.appState`) != "object" {
		t.Fatal("appState should exist")
	}
	if evalStr(ctx, `appState.initialized`) != "true" {
		t.Fatal("appState.initialized should be true")
	}
	if evalInt(ctx, `Object.keys(__timers).length`) < 1 {
		t.Fatal("expected timer")
	}

	// One teardown destroys everything
	removed := evalInt(ctx, `app.teardown()`)
	t.Logf("Removed %d resources (globals + hooks + timers)", removed)

	// Globals cleaned
	if evalStr(ctx, `typeof globalThis.appState`) != "undefined" {
		t.Fatal("appState should be gone (global snapshot cleanup)")
	}

	// Hooks ran
	log := evalStr(ctx, `JSON.stringify(__combined_log)`)
	t.Logf("Cleanup log: %s", log)

	// Timers cancelled
	if evalInt(ctx, `Object.keys(__timers).length`) != 0 {
		t.Fatal("timers should be cancelled")
	}

	t.Log("PASS: combined approach — globals + hooks + timers all cleaned in one teardown")
}

// ═══════════════════════════════════════════════════════════════
// Test 5: Two files — one teardown doesn't affect the other's globals
// ═══════════════════════════════════════════════════════════════

func TestUserRes_IsolatedGlobalSnapshots(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupUserResourceInfra(ctx)

	eval(ctx, `
		globalThis.fileA = deploy("file-a.ts", function() {
			globalThis.stateA = { from: "A" };
			return "A";
		});
	`)
	eval(ctx, `
		globalThis.fileB = deploy("file-b.ts", function() {
			globalThis.stateB = { from: "B" };
			return "B";
		});
	`)

	// Both exist
	if evalStr(ctx, `stateA.from`) != "A" { t.Fatal("stateA missing") }
	if evalStr(ctx, `stateB.from`) != "B" { t.Fatal("stateB missing") }

	// Teardown only A
	evalInt(ctx, `fileA.teardown()`)

	// A gone, B intact
	if evalStr(ctx, `typeof globalThis.stateA`) != "undefined" {
		t.Fatal("stateA should be gone")
	}
	if evalStr(ctx, `stateB.from`) != "B" {
		t.Fatal("stateB should survive")
	}

	// Teardown B
	evalInt(ctx, `fileB.teardown()`)
	if evalStr(ctx, `typeof globalThis.stateB`) != "undefined" {
		t.Fatal("stateB should be gone")
	}

	t.Log("PASS: global snapshots are per-file — isolated teardown works")
}

// ═══════════════════════════════════════════════════════════════
// Test 6: Stress — 30 deploy/teardown with user resources
// ═══════════════════════════════════════════════════════════════

func TestUserRes_StressCycles(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	setupUserResourceInfra(ctx)

	const cycles = 30

	for i := range cycles {
		source := fmt.Sprintf("user-cycle-%d", i)
		eval(ctx, fmt.Sprintf(`
			globalThis.__cycle = deploy(%q, function() {
				globalThis["state_%d"] = { cycle: %d, data: new Array(50).fill("x") };
				onTeardown(function() {});
				setTimeout(function() {}, 1000);
				return %d;
			});
		`, source, i, i, i))

		// Verify created
		state := evalStr(ctx, fmt.Sprintf(`typeof globalThis["state_%d"]`, i))
		if state != "object" {
			t.Fatalf("cycle %d: state not created", i)
		}

		// Teardown
		evalInt(ctx, `__cycle.teardown()`)

		// Verify clean
		state = evalStr(ctx, fmt.Sprintf(`typeof globalThis["state_%d"]`, i))
		if state != "undefined" {
			t.Fatalf("cycle %d: state leaked", i)
		}

		if i%10 == 0 {
			rt.RunGC()
		}
	}

	// Final check
	if evalInt(ctx, `__registry.total()`) != 0 {
		t.Fatal("registry should be empty")
	}
	if evalInt(ctx, `Object.keys(__timers).length`) != 0 {
		t.Fatal("timers should be empty")
	}

	t.Logf("PASS: %d cycles with user resources — globals, hooks, timers all cleaned", cycles)
}
