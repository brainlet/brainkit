package agentembed

import (
	_ "embed"
	"fmt"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

//go:generate go run ./cmd/compile-bundle

//go:embed agent_embed_bundle.js
var bundleSource string

//go:embed agent_embed_bundle.bc
var bundleBytecode []byte

//go:embed ses_polyfills.js
var sesPolyfillsSource string

//go:embed ses.umd.js
var sesSource string

// sesLockdownJS calls lockdown() with QuickJS-compatible options.
// Must run after ses.umd.js and before the Mastra bundle.
const sesLockdownJS = `lockdown({ errorTaming: "unsafe", overrideTaming: "moderate", consoleTaming: "unsafe", evalTaming: "unsafe-eval" });`

// LoadBundle evaluates the agent-embed bundle in the given bridge.
// Loading order: globals → SES polyfills → SES → lockdown → Mastra bundle.
// After loading, globalThis.__agent_embed is available with Agent, createTool, and Mastra.
func LoadBundle(b *jsbridge.Bridge) error {
	// 1. Node.js/browser global polyfills (process, Buffer, etc.)
	setup, err := b.Eval("agent-embed-setup.js", runtimeGlobalsJS)
	if err != nil {
		return fmt.Errorf("agent-embed: setup globals: %w", err)
	}
	setup.Free()

	// 2. SES polyfills (console stubs, Iterator prototype fix)
	sp, err := b.Eval("ses-polyfills.js", sesPolyfillsSource)
	if err != nil {
		return fmt.Errorf("agent-embed: SES polyfills: %w", err)
	}
	sp.Free()

	// 3. SES UMD (provides Compartment, harden, lockdown)
	sv, err := b.Eval("ses.umd.js", sesSource)
	if err != nil {
		return fmt.Errorf("agent-embed: SES load: %w", err)
	}
	sv.Free()

	// 4. Call lockdown() — freezes intrinsics, enables Compartment isolation
	lv, err := b.Eval("ses-lockdown.js", sesLockdownJS)
	if err != nil {
		return fmt.Errorf("agent-embed: SES lockdown: %w", err)
	}
	lv.Free()

	// 5. Mastra bundle (Agent, createTool, AI SDK, etc.)
	if len(bundleBytecode) > 0 {
		val, err := b.EvalBytecode(bundleBytecode)
		if err != nil {
			return fmt.Errorf("agent-embed: load bytecode: %w", err)
		}
		val.Free()
		return nil
	}

	val, err := b.EvalAsync("agent-embed-bundle.js", bundleSource)
	if err != nil {
		return fmt.Errorf("agent-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}


// BundleSource returns the raw JS bundle source (for benchmarking/compilation).
func BundleSource() string { return bundleSource }

// LoadPrelude loads everything except the main bundle: globals, SES polyfills, SES UMD, lockdown.
func LoadPrelude(b *jsbridge.Bridge) error {
	setup, err := b.Eval("agent-embed-setup.js", runtimeGlobalsJS)
	if err != nil {
		return fmt.Errorf("agent-embed: setup globals: %w", err)
	}
	setup.Free()
	sp, err := b.Eval("ses-polyfills.js", sesPolyfillsSource)
	if err != nil {
		return fmt.Errorf("agent-embed: SES polyfills: %w", err)
	}
	sp.Free()
	sv, err := b.Eval("ses.umd.js", sesSource)
	if err != nil {
		return fmt.Errorf("agent-embed: SES load: %w", err)
	}
	sv.Free()
	lv, err := b.Eval("ses-lockdown.js", sesLockdownJS)
	if err != nil {
		return fmt.Errorf("agent-embed: SES lockdown: %w", err)
	}
	lv.Free()
	return nil
}

// runtimeGlobalsJS contains ONLY bundle-specific setup that runs before SES lockdown.
// All Node.js API polyfills are now in jsbridge/*.go and loaded by sandbox.go.
// What remains here:
//   1. Pre-lockdown captures — SES tames Math.random/Date, we capture originals first
//   2. require() shim — bundle has dynamic require() calls for otel, zod, vscode-jsonrpc, execa
const runtimeGlobalsJS = `
// ─── Pre-lockdown captures ──────────────────────────────────────────────
// SES lockdown() tames Math.random, Date.now, Date() as "ambient authority".
// Capture the real implementations NOW, before lockdown freezes them.
// kit_runtime.js uses these to build Compartment endowments that restore access.
(function() {
  var _origMathRandom = Math.random.bind(Math);
  var _origDateNow = Date.now.bind(Date);
  var _origDate = Date;
  globalThis.__brainkit_pre_lockdown = {
    mathRandom: _origMathRandom,
    dateNow: _origDateNow,
    Date: _origDate,
  };
})();

// ─── require shim ───────────────────────────────────────────────────────
// Bundle-specific dynamic imports that can't be resolved at esbuild time.
// This require() is captured by esbuild's internal resolver at bundle start.
if (typeof require === "undefined") {
  var _noopSpan = {
    setAttribute: function() { return this; },
    setAttributes: function() { return this; },
    addEvent: function() { return this; },
    setStatus: function() { return this; },
    end: function() {},
    isRecording: function() { return false; },
    recordException: function() {},
    updateName: function() { return this; },
    spanContext: function() { return { traceId: "", spanId: "", traceFlags: 0 }; },
  };
  var _noopTracer = {
    startSpan: function() { return _noopSpan; },
    startActiveSpan: function(name, optionsOrFn, fnOrUndef) {
      var fn = typeof optionsOrFn === "function" ? optionsOrFn : fnOrUndef;
      if (typeof fn === "function") return fn(_noopSpan);
      return _noopSpan;
    },
  };
  var _otelStub = {
    trace: {
      getTracer: function() { return _noopTracer; },
      setSpan: function(ctx) { return ctx; },
      getSpan: function() { return _noopSpan; },
      getActiveSpan: function() { return undefined; },
    },
    context: {
      active: function() { return {}; },
      with: function(ctx, fn) { return fn(); },
      bind: function(ctx, fn) { return fn; },
    },
    SpanStatusCode: { UNSET: 0, OK: 1, ERROR: 2 },
    SpanKind: { INTERNAL: 0, SERVER: 1, CLIENT: 2 },
    diag: { debug: function() {}, info: function() {}, warn: function() {}, error: function() {} },
    propagation: {},
    metrics: { getMeter: function() { return {}; } },
  };
  var _zodV4Wrapper = {
    toJSONSchema: function() {
      var real = globalThis.__zod_v4_module;
      if (real && typeof real.toJSONSchema === "function") {
        return real.toJSONSchema.apply(real, arguments);
      }
      throw new Error("toJSONSchema not yet available");
    },
  };
  globalThis.require = function(mod) {
    if (mod === "@opentelemetry/api") return _otelStub;
    if (mod === "zod/v4" || mod === "zod") {
      return globalThis.__zod_v4_module || _zodV4Wrapper;
    }
    if (mod === "vscode-jsonrpc/node" || mod === "vscode-jsonrpc") {
      return globalThis.__vscode_jsonrpc_node || {};
    }
    if (mod === "vscode-languageserver-protocol") {
      return globalThis.__vscode_lsp_protocol || {};
    }
    if (mod === "execa") {
      return { execa: globalThis.__execa_polyfill || function() { throw new Error("execa not available"); } };
    }
    return {};
  };
}

// All other polyfills (Error.captureStackTrace, process extensions, Buffer,
// navigator, performance, Intl, EventTarget, scheduling, Headers, etc.)
// are now loaded by jsbridge polyfills in sandbox.go BEFORE this code runs.

"ok";
`
