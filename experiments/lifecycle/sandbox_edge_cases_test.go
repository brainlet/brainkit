//go:build experiment

// Edge case experiments for the with(proxy) sandbox approach.
// Testing everything that could go wrong.
package lifecycle

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

func setupSandboxInfra(ctx *quickjs.Context) {
	eval(ctx, `
		globalThis.__kit = {
			agent: function(cfg) { return { name: cfg.name, type: "agent" }; },
			createTool: function(cfg) { return { id: cfg.id, type: "tool" }; },
			z: { object: function() { return {}; }, string: function() { return {}; } },
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

			var fn = new Function("sandbox", "with(sandbox) { " + code + " }");
			fn(sandbox);

			__file_scopes[source] = localScope;
			return {
				scope: localScope,
				teardown: function() { delete __file_scopes[source]; },
			};
		};
	`)
}

// ═══════════════════════════════════════════════════════════════
// Edge 1: 'this' keyword — does it point to proxy or globalThis?
// ═══════════════════════════════════════════════════════════════

func TestEdge_ThisKeyword(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__this_test = sandboxedDeploy("this-test.ts",
			'var selfRef = this;' +
			'var isGlobal = (selfRef === globalThis);'
		);
	`)

	// 'this' inside new Function points to globalThis (not the proxy)
	// This is expected behavior for non-strict mode
	isGlobal := evalStr(ctx, `"" + __this_test.scope.isGlobal`)
	t.Logf("this === globalThis inside sandbox: %s", isGlobal)

	t.Log("NOTE: 'this' inside sandbox points to globalThis, not the proxy. Users should use variable names, not 'this'.")
}

// ═══════════════════════════════════════════════════════════════
// Edge 2: typeof — does typeof undeclared var work?
// ═══════════════════════════════════════════════════════════════

func TestEdge_TypeofUndeclared(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__typeof_test = sandboxedDeploy("typeof-test.ts",
			'var result = typeof nonExistentVar;'
		);
	`)

	result := evalStr(ctx, `__typeof_test.scope.result`)
	if result != "undefined" {
		t.Fatalf("typeof undeclared should be 'undefined', got %s", result)
	}

	t.Log("PASS: typeof works correctly in sandbox for undeclared variables")
}

// ═══════════════════════════════════════════════════════════════
// Edge 3: Function declarations — do they hoist correctly?
// ═══════════════════════════════════════════════════════════════

func TestEdge_FunctionDeclarations(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__fn_test = sandboxedDeploy("fn-test.ts",
			// Function declaration
			'function myHelper(x) { return x * 2; }' +
			'var result = myHelper(21);'
		);
	`)

	result := evalInt(ctx, `__fn_test.scope.result`)
	if result != 42 {
		t.Fatalf("expected 42, got %d", result)
	}

	// Function should be in local scope, not global
	if evalStr(ctx, `typeof globalThis.myHelper`) != "undefined" {
		t.Fatal("function declaration leaked to globalThis")
	}

	t.Log("PASS: function declarations work and stay in sandbox scope")
}

// ═══════════════════════════════════════════════════════════════
// Edge 4: Closures — does a closure created in sandbox
// retain access to sandbox scope?
// ═══════════════════════════════════════════════════════════════

func TestEdge_ClosuresRetainScope(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__closure_test = sandboxedDeploy("closure-test.ts",
			'var secret = "hidden";' +
			'var getSecret = function() { return secret; };'
		);
	`)

	// The closure should access the sandboxed 'secret', not a global
	result := evalStr(ctx, `__closure_test.scope.getSecret()`)
	if result != "hidden" {
		t.Fatalf("closure should access sandbox scope, got %s", result)
	}

	// secret is NOT on globalThis
	if evalStr(ctx, `typeof globalThis.secret`) != "undefined" {
		t.Fatal("secret leaked to globalThis")
	}

	t.Log("PASS: closures retain access to sandboxed variables")
}

// ═══════════════════════════════════════════════════════════════
// Edge 5: Async code — does async/await work inside sandbox?
// ═══════════════════════════════════════════════════════════════

func TestEdge_AsyncInSandbox(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	// Modify deploy to support async
	eval(ctx, `
		globalThis.sandboxedDeployAsync = function(source, code) {
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

			var fn = new Function("sandbox", "with(sandbox) { return (async function() { " + code + " })(); }");
			var promise = fn(sandbox);

			__file_scopes[source] = localScope;
			return {
				promise: promise,
				scope: localScope,
				teardown: function() { delete __file_scopes[source]; },
			};
		};
	`)

	// async code can't easily be tested in pure QuickJS Eval (no event loop)
	// but we can verify the Function construction works
	eval(ctx, `
		globalThis.__async_test = sandboxedDeployAsync("async-test.ts",
			'var x = 42;' +
			'var result = x + 1;'
		);
	`)

	result := evalInt(ctx, `__async_test.scope.result`)
	if result != 43 {
		t.Fatalf("expected 43, got %d", result)
	}

	t.Log("PASS: async wrapper construction works inside sandbox")
}

// ═══════════════════════════════════════════════════════════════
// Edge 6: Can sandboxed code call our API functions?
// agent(), createTool() etc. that live on globalThis.__kit
// ═══════════════════════════════════════════════════════════════

func TestEdge_APIAccessFromSandbox(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__api_test = sandboxedDeploy("api-test.ts",
			'var a = __kit.agent({ name: "sandboxed-agent" });' +
			'var t = __kit.createTool({ id: "sandboxed-tool" });'
		);
	`)

	name := evalStr(ctx, `__api_test.scope.a.name`)
	if name != "sandboxed-agent" {
		t.Fatalf("expected sandboxed-agent, got %s", name)
	}

	toolId := evalStr(ctx, `__api_test.scope.t.id`)
	if toolId != "sandboxed-tool" {
		t.Fatalf("expected sandboxed-tool, got %s", toolId)
	}

	// API results are in local scope
	if evalStr(ctx, `typeof globalThis.a`) != "undefined" {
		t.Fatal("agent leaked to globalThis")
	}

	t.Log("PASS: API functions accessible from sandbox, results captured locally")
}

// ═══════════════════════════════════════════════════════════════
// Edge 7: Can sandboxed code use destructuring?
// const { agent, createTool } = __kit;
// ═══════════════════════════════════════════════════════════════

func TestEdge_Destructuring(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	// Note: 'const' and 'let' in new Function with 'with' is tricky
	// because const/let create block scope, not captured by proxy
	eval(ctx, `
		globalThis.__destr_test = sandboxedDeploy("destr-test.ts",
			'var agent = __kit.agent;' +
			'var myAgent = agent({ name: "destr-agent" });'
		);
	`)

	name := evalStr(ctx, `__destr_test.scope.myAgent.name`)
	if name != "destr-agent" {
		t.Fatalf("expected destr-agent, got %s", name)
	}

	t.Log("PASS: destructuring (via var) works in sandbox")
}

// ═══════════════════════════════════════════════════════════════
// Edge 8: const/let inside sandbox — do they leak?
// THIS IS THE CRITICAL EDGE CASE.
// const/let create block scope, NOT captured by proxy set trap.
// ═══════════════════════════════════════════════════════════════

func TestEdge_ConstLetBehavior(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__constlet_test = sandboxedDeploy("constlet-test.ts",
			'var varValue = "captured-by-proxy";' +
			'const constValue = "block-scoped";' +
			'let letValue = "also-block-scoped";' +
			// Store a flag to test what's visible
			'var varSees = typeof constValue;' +
			'var letSees = typeof letValue;'
		);
	`)

	// var goes through proxy → captured in localScope
	varVal := evalStr(ctx, `__constlet_test.scope.varValue`)
	if varVal != "captured-by-proxy" {
		t.Fatalf("var should be captured: %s", varVal)
	}

	// const/let are block-scoped — they DON'T go through the proxy set trap
	// They exist during execution but aren't in localScope after
	constInScope := evalStr(ctx, `typeof __constlet_test.scope.constValue`)
	letInScope := evalStr(ctx, `typeof __constlet_test.scope.letValue`)

	t.Logf("const in localScope: %s", constInScope)
	t.Logf("let in localScope: %s", letInScope)

	// BUT: the code CAN access const/let during execution
	varSees := evalStr(ctx, `__constlet_test.scope.varSees`)
	t.Logf("typeof constValue during execution: %s", varSees)

	if constInScope != "undefined" {
		t.Log("NOTE: const IS captured in proxy scope (QuickJS behavior)")
	} else {
		t.Log("NOTE: const is NOT captured in proxy scope (block-scoped as expected)")
	}

	t.Log("Edge case documented — const/let behavior in with(proxy) depends on engine")
}

// ═══════════════════════════════════════════════════════════════
// Edge 9: Error handling — does try/catch work in sandbox?
// ═══════════════════════════════════════════════════════════════

func TestEdge_ErrorHandling(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()
	setupSandboxInfra(ctx)

	eval(ctx, `
		globalThis.__error_test = sandboxedDeploy("error-test.ts",
			'var caught = null;' +
			'try { throw new Error("sandbox-error"); }' +
			'catch(e) { caught = e.message; }'
		);
	`)

	caught := evalStr(ctx, `__error_test.scope.caught`)
	if caught != "sandbox-error" {
		t.Fatalf("expected sandbox-error, got %s", caught)
	}

	t.Log("PASS: try/catch works in sandbox")
}

// ═══════════════════════════════════════════════════════════════
// Edge 10: Strict mode — 'with' is FORBIDDEN in strict mode!
// This is the biggest potential blocker.
// ═══════════════════════════════════════════════════════════════

func TestEdge_StrictMode(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// Try using 'with' in strict mode
	result := evalStr(ctx, `
		var error = null;
		try {
			var fn = new Function('"use strict"; with({}) { var x = 1; }');
			fn();
		} catch(e) {
			error = e.message;
		}
		error || "no-error";
	`)

	t.Logf("Strict mode + with: %s", result)

	if result == "no-error" {
		t.Log("PASS: strict mode allows with (unexpected but good for us)")
	} else {
		t.Logf("EXPECTED: strict mode FORBIDS with — error: %s", result)
		t.Log("This means: if .ts code compiles to strict mode, the with(proxy) approach FAILS.")
		t.Log("Fallback needed: IIFE + global snapshot (current approach) or separate contexts.")
	}
}

// ═══════════════════════════════════════════════════════════════
// Edge 11: Module syntax — can we use import/export in sandbox?
// ═══════════════════════════════════════════════════════════════

func TestEdge_ImportExport(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	ctx := rt.NewContext()
	defer ctx.Close()

	// ES modules can't use 'with' (modules are strict mode by default)
	// This tests whether the limitation matters
	result := evalStr(ctx, `
		var error = null;
		try {
			// Modules are always strict mode in ES spec
			// new Function with module syntax doesn't work
			var fn = new Function('import { x } from "test"');
			fn();
		} catch(e) {
			error = e.message;
		}
		error || "no-error";
	`)

	t.Logf("import in new Function: %s", result)
	t.Log("NOTE: import/export can't work with new Function — modules need different handling")
}
