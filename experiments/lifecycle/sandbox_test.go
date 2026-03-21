//go:build experiment

// Experiment: Can we sandbox scripts within the same QuickJS runtime?
//
// Three approaches:
// 1. Separate contexts (built into QuickJS)
// 2. ES Module scoping (EvalModule instead of Eval)
// 3. Proxy-based virtual globals per source file
package lifecycle

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

// ═══════════════════════════════════════════════════════════════
// Test 1: Do const/let in IIFE already scope properly?
// (This is what EvalTS does today)
// ═══════════════════════════════════════════════════════════════

func TestSandbox_IIFEScoping(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// File A defines a const
	eval(ctx, `(function() {
		const client = "from-A";
		globalThis.__resultA = typeof client;
	})()`)

	// File B defines the same const name
	eval(ctx, `(function() {
		const client = "from-B";
		globalThis.__resultB = typeof client;
	})()`)

	// Neither leaks to global
	typ := evalStr(ctx, `typeof client`)
	if typ != "undefined" {
		t.Fatalf("const should not leak to global, got %s", typ)
	}

	t.Log("PASS: const/let in IIFE are already scoped — no collision between files")
}

// ═══════════════════════════════════════════════════════════════
// Test 2: Only globalThis.x = ... causes collision
// ═══════════════════════════════════════════════════════════════

func TestSandbox_OnlyExplicitGlobalsCollide(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// File A sets a global
	eval(ctx, `(function() {
		globalThis.API_URL = "https://a.com";
	})()`)

	// File B overwrites it
	eval(ctx, `(function() {
		globalThis.API_URL = "https://b.com";
	})()`)

	url := evalStr(ctx, `globalThis.API_URL`)
	if url != "https://b.com" {
		t.Fatalf("expected B's value, got %s", url)
	}

	t.Log("PASS (demonstrating the problem): explicit globalThis writes DO collide")
}

// ═══════════════════════════════════════════════════════════════
// Test 3: Separate QuickJS Contexts — full isolation
// ═══════════════════════════════════════════════════════════════

func TestSandbox_SeparateContexts(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	defer ctxA.Close()
	ctxB := rt.NewContext()
	defer ctxB.Close()

	// A sets a global
	eval(ctxA, `globalThis.secret = "A-only";`)

	// B can't see it
	typ := evalStr(ctxB, `typeof globalThis.secret`)
	if typ != "undefined" {
		t.Fatalf("context B should not see A's global, got %s", typ)
	}

	// B sets same name — no collision
	eval(ctxB, `globalThis.secret = "B-only";`)

	// A still has its own value
	val := evalStr(ctxA, `globalThis.secret`)
	if val != "A-only" {
		t.Fatalf("A's value should be untouched, got %s", val)
	}

	t.Log("PASS: separate contexts = complete isolation, no global collision")
}

// ═══════════════════════════════════════════════════════════════
// Test 4: Does QuickJS support Proxy?
// ═══════════════════════════════════════════════════════════════

func TestSandbox_ProxySupport(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	result := evalStr(ctx, `
		var target = { x: 1 };
		var handler = {
			get: function(t, prop) {
				if (prop === "x") return 42;
				return t[prop];
			}
		};
		var p = new Proxy(target, handler);
		"" + p.x;
	`)

	if result != "42" {
		t.Fatalf("Proxy not working, got %s", result)
	}

	t.Log("PASS: QuickJS supports Proxy")
}

// ═══════════════════════════════════════════════════════════════
// Test 5: Proxy-based virtual global scope per file
// Each file gets its own "layer" over globalThis.
// Writes go to the layer. Reads fall through to real global.
// ═══════════════════════════════════════════════════════════════

func TestSandbox_ProxyVirtualGlobals(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// Set up the sandbox infrastructure
	eval(ctx, `
		globalThis.__sandboxes = {};

		globalThis.__createSandbox = function(source) {
			var localScope = {};
			var proxy = new Proxy(globalThis, {
				get: function(target, prop) {
					if (prop in localScope) return localScope[prop];
					return target[prop];
				},
				set: function(target, prop, value) {
					// Intercept writes — store locally, don't pollute globalThis
					localScope[prop] = value;
					return true;
				},
				has: function(target, prop) {
					return (prop in localScope) || (prop in target);
				},
				deleteProperty: function(target, prop) {
					if (prop in localScope) { delete localScope[prop]; return true; }
					return delete target[prop];
				},
			});
			__sandboxes[source] = { proxy: proxy, scope: localScope };
			return proxy;
		};

		globalThis.__destroySandbox = function(source) {
			delete __sandboxes[source];
		};

		globalThis.__getSandboxKeys = function(source) {
			var sb = __sandboxes[source];
			return sb ? Object.keys(sb.scope) : [];
		};
	`)

	// Simulate two files writing to "their" globalThis
	// We use Function constructor to run code with a custom "this" / scope
	eval(ctx, `
		// File A's sandbox
		var sandboxA = __createSandbox("file-a.ts");

		// Simulate: file A sets API_URL
		// We can't redirect 'globalThis' keyword, but we can use the proxy directly
		sandboxA.API_URL = "https://a.com";
		sandboxA.myCache = { data: [] };

		// File B's sandbox
		var sandboxB = __createSandbox("file-b.ts");
		sandboxB.API_URL = "https://b.com";
		sandboxB.myCache = { data: [1, 2, 3] };
	`)

	// Real globalThis is clean
	typ := evalStr(ctx, `typeof globalThis.API_URL`)
	if typ != "undefined" {
		t.Fatalf("globalThis should not have API_URL, got %s", typ)
	}

	// Each sandbox has its own value
	aUrl := evalStr(ctx, `__sandboxes["file-a.ts"].scope.API_URL`)
	bUrl := evalStr(ctx, `__sandboxes["file-b.ts"].scope.API_URL`)

	if aUrl != "https://a.com" {
		t.Fatalf("A's URL wrong: %s", aUrl)
	}
	if bUrl != "https://b.com" {
		t.Fatalf("B's URL wrong: %s", bUrl)
	}

	// Reading through proxy falls through to globalThis for shared stuff
	eval(ctx, `globalThis.__shared = "visible-to-all";`)
	shared := evalStr(ctx, `__sandboxes["file-a.ts"].proxy.__shared`)
	if shared != "visible-to-all" {
		t.Fatalf("proxy should fall through to globalThis for reads: %s", shared)
	}

	// Destroy sandbox A — only A's scope is gone
	eval(ctx, `__destroySandbox("file-a.ts")`)

	bKeys := evalStr(ctx, `JSON.stringify(__getSandboxKeys("file-b.ts"))`)
	t.Logf("B's keys after A destroyed: %s", bKeys)

	// B still has its values
	bUrl = evalStr(ctx, `__sandboxes["file-b.ts"].scope.API_URL`)
	if bUrl != "https://b.com" {
		t.Fatalf("B should be untouched: %s", bUrl)
	}

	t.Log("PASS: Proxy-based sandboxes — writes captured locally, reads fall through, full isolation")
}

// ═══════════════════════════════════════════════════════════════
// Test 6: Can we use `with` statement to redirect variable access
// through a Proxy? (This would make user code transparent)
// ═══════════════════════════════════════════════════════════════

func TestSandbox_WithProxyTransparent(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// Set up shared globals
	eval(ctx, `
		globalThis.__kit = { agent: function(cfg) { return { name: cfg.name }; } };
	`)

	result := evalStr(ctx, `
		var localScope = {};
		var sandbox = new Proxy(localScope, {
			get: function(target, prop) {
				if (prop in target) return target[prop];
				return globalThis[prop]; // fall through
			},
			set: function(target, prop, value) {
				target[prop] = value; // always local
				return true;
			},
			has: function(target, prop) {
				// CRITICAL: return true for everything so 'with' intercepts all lookups
				return true;
			},
		});

		var result;
		with (sandbox) {
			// This code thinks it's writing to globals, but everything is captured in localScope
			var myVar = 42;
			API_URL = "https://sandboxed.com";

			// Can still access real globals via fall-through
			var kitRef = __kit;

			result = API_URL + " | " + myVar + " | " + (kitRef ? "kit-found" : "no-kit");
		}
		result;
	`)

	t.Logf("Result from sandboxed code: %s", result)

	// Verify globalThis was NOT polluted
	typ := evalStr(ctx, `typeof globalThis.API_URL`)
	if typ != "undefined" {
		t.Fatalf("globalThis polluted! API_URL = %s", evalStr(ctx, `globalThis.API_URL`))
	}
	typ = evalStr(ctx, `typeof globalThis.myVar`)
	if typ != "undefined" {
		t.Fatalf("globalThis polluted! myVar exists")
	}

	t.Log("PASS: with(proxy) makes sandboxing TRANSPARENT — user code works normally, globals captured locally")
}

// ═══════════════════════════════════════════════════════════════
// Test 7: Full transparent sandbox — deploy function that wraps
// user code in with(proxy) automatically
// ═══════════════════════════════════════════════════════════════

func TestSandbox_TransparentDeploy(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	eval(ctx, `
		globalThis.__kit = {
			agent: function(cfg) { return { name: cfg.name, type: "agent" }; },
			createTool: function(cfg) { return { id: cfg.id, type: "tool" }; },
		};

		globalThis.__file_scopes = {};

		globalThis.sandboxedDeploy = function(source, code) {
			var localScope = {};
			var sandbox = new Proxy(localScope, {
				get: function(target, prop) {
					if (prop in target) return target[prop];
					return globalThis[prop];
				},
				set: function(target, prop, value) {
					target[prop] = value;
					return true;
				},
				has: function() { return true; },
			});

			// Run user code inside with(sandbox)
			var fn = new Function("sandbox", "with(sandbox) { " + code + " }");
			fn(sandbox);

			__file_scopes[source] = localScope;
			return {
				scope: localScope,
				teardown: function() {
					delete __file_scopes[source];
				},
			};
		};
	`)

	// Deploy two files with same variable names
	eval(ctx, `
		globalThis.fileA = sandboxedDeploy("plugin-a.ts",
			'var config = { url: "https://a.com" };' +
			'var myAgent = __kit.agent({ name: "agent-a" });'
		);
	`)

	eval(ctx, `
		globalThis.fileB = sandboxedDeploy("plugin-b.ts",
			'var config = { url: "https://b.com" };' +
			'var myAgent = __kit.agent({ name: "agent-b" });'
		);
	`)

	// No collision — each file has its own "config" and "myAgent"
	aUrl := evalStr(ctx, `fileA.scope.config.url`)
	bUrl := evalStr(ctx, `fileB.scope.config.url`)
	aAgent := evalStr(ctx, `fileA.scope.myAgent.name`)
	bAgent := evalStr(ctx, `fileB.scope.myAgent.name`)

	if aUrl != "https://a.com" { t.Fatalf("A url: %s", aUrl) }
	if bUrl != "https://b.com" { t.Fatalf("B url: %s", bUrl) }
	if aAgent != "agent-a" { t.Fatalf("A agent: %s", aAgent) }
	if bAgent != "agent-b" { t.Fatalf("B agent: %s", bAgent) }

	// globalThis is clean
	if evalStr(ctx, `typeof globalThis.config`) != "undefined" {
		t.Fatal("globalThis polluted by config")
	}
	if evalStr(ctx, `typeof globalThis.myAgent`) != "undefined" {
		t.Fatal("globalThis polluted by myAgent")
	}

	// Teardown A, B still works
	eval(ctx, `fileA.teardown()`)
	bUrl = evalStr(ctx, `fileB.scope.config.url`)
	if bUrl != "https://b.com" {
		t.Fatal("B should survive A's teardown")
	}

	t.Log("PASS: transparent sandboxed deploy — same variable names, zero collision, zero globalThis pollution")
}
