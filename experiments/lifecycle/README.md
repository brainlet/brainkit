# Lifecycle Management Experiments

**Date**: 2026-03-19
**Status**: Research complete — SES Compartments proven as the production solution
**Tests**: 80 tagged tests across 10 test files

## Running The Experiments

These tests are exploratory and stay out of the default repository verification path. Run them explicitly with the `experiment` build tag:

```bash
go test -tags experiment ./experiments/lifecycle
```

## The Question

Can we reliably create, track, and DESTROY JavaScript objects from Go — and guarantee they're actually gone? If yes, `.ts` files become the composition API with full lifecycle management.

## Why This Matters

Brainkit's current API exposes 50+ typed bus messages for every platform operation. Most of these are configuration operations (create agent, register tool, create memory) that `.ts` code already does natively via `agent()`, `createTool()`, etc. The typed messages are redundant.

If we can guarantee lifecycle management — deploy a `.ts` file, track everything it creates, tear it down completely — then:
- `.ts` files become the configuration API (deploy/teardown)
- Typed messages shrink to runtime operations only (ai.generate, tools.call, etc.)
- Plugins deploy `.ts` files instead of sending registration messages
- The SDK gets dramatically simpler

But this only works if teardown is 100% reliable. No leaks, no dangling references, no crashes.

## Research Path

We explored four approaches, each building on the previous:

### Approach 1: Single Context + Cleanup Hooks

**Files**: `lifecycle_test.go`, `scoped_lifecycle_test.go`, `kit_lifecycle_test.go`

Every creation function (`agent()`, `createTool()`, `bus.subscribe()`) silently registers a cleanup hook in the resource registry. `TeardownFile(source)` runs all hooks in LIFO order.

**What works**:
- `Value.Delete(name)` removes globals, GC reclaims them
- Closures keep objects alive (must unsubscribe to fully clean up)
- Cleanup hooks handle: agents, tools, memory, bus subscriptions
- `scope(name, fn)` wraps code with automatic source tracking
- Nested scopes, redeploy (atomic swap), 100-cycle stress — all pass
- Real Kit integration: 8 tests with actual brainkit infrastructure

**Limitation**: No isolation between files sharing the same QuickJS context. Two plugins writing `globalThis.x` collide.

### Approach 2: User Resource Tracking

**File**: `user_resources_test.go`

Three layers to catch resources the developer creates outside our API:
1. **Global snapshot**: diff `globalThis` keys before/after eval, auto-delete new ones
2. **`onTeardown(fn)`**: user registers custom cleanup (closures, connections, caches)
3. **Timer wrapping**: `setTimeout`/`setInterval` tracked by source, cancelled on teardown

**What works**: All three layers combined — globals, hooks, timers all cleaned in one teardown. 30-cycle stress passes.

**Limitation**: Still no isolation. Snapshot can't disambiguate two files setting the same global key.

### Approach 3: Sandboxing Within Same Runtime

**Files**: `sandbox_test.go`, `sandbox_edge_cases_test.go`

Explored three JS sandboxing techniques:

1. **IIFE scoping** (current approach): `const`/`let` are scoped, only explicit `globalThis.x` leaks. Works but no true isolation.

2. **`with(proxy)`**: Transparent sandboxing — all variable writes captured locally, reads fall through to real globals. Works in non-strict mode but **fails in strict mode** (`with` is forbidden). Since TypeScript and ES modules are always strict, this is a dead end for production.

3. **Separate QuickJS Contexts**: Full isolation, separate globals. Works but requires Mastra bundle (16.5MB) loaded per context — memory cost is prohibitive.

### Approach 4: SES Compartments (THE ANSWER)

**File**: `ses_test.go`

[SES (Secure ECMAScript)](https://hardenedjs.org/) from Agoric provides `Compartment` — an isolated evaluation environment used by MetaMask for plugin sandboxing.

**Key properties**:
- Separate globals per compartment (true isolation)
- Shared intrinsics — `Array`, `Object`, `Map`, etc. are shared (no memory duplication)
- Controlled API access via endowments (pass specific functions to each compartment)
- Works in strict mode, compatible with ES modules
- 220KB bundle size
- `harden()` freezes objects to prevent mutation

**QuickJS compatibility**: SES loads and `lockdown()` works with two polyfills:

1. **Console methods**: QuickJS lacks `console.groupCollapsed` and other methods that SES's `tameConsole` expects. Fixed with stub functions.

2. **Iterator prototype faux data properties**: QuickJS's Iterator Helpers expose `constructor` and `Symbol.toStringTag` as accessor properties with setters that crash when called on null-prototype objects. SES's `tameFauxDataProperty` triggers this. Fixed by converting these to real data properties before SES loads.

**Polyfill code** (add before loading SES):
```javascript
// Console stubs
["log","warn","error","info","debug","time","timeEnd","timeLog",
 "group","groupEnd","groupCollapsed","assert","count","countReset",
 "dir","dirxml","table","trace","clear","profile","profileEnd",
 "timeStamp"].forEach(m => { if (!console[m]) console[m] = () => {}; });

// Iterator prototype fix
(function() {
  var ai = [][Symbol.iterator]();
  var ip = Object.getPrototypeOf(Object.getPrototypeOf(ai));
  if (ip) {
    var cd = Object.getOwnPropertyDescriptor(ip, 'constructor');
    if (cd && cd.get && !('value' in cd)) {
      Object.defineProperty(ip, 'constructor', {
        value: cd.get.call(ip), writable: true,
        enumerable: false, configurable: true
      });
    }
    var td = Object.getOwnPropertyDescriptor(ip, Symbol.toStringTag);
    if (td && td.get && !('value' in td)) {
      Object.defineProperty(ip, Symbol.toStringTag, {
        value: td.get.call(ip), writable: false,
        enumerable: false, configurable: true
      });
    }
  }
})();
```

**Lockdown options** that work:
```javascript
lockdown({
  errorTaming: "unsafe",       // QuickJS error stacks
  overrideTaming: "moderate",  // compatibility
  consoleTaming: "unsafe",     // keep our console
  evalTaming: "unsafe-eval",   // required — safe-eval doesn't work in QuickJS
});
```

**Note on `evalTaming: "unsafe-eval"`**: Each `evaluate()` call is independent — variables don't persist between calls on the same compartment. This is fine for our use case: a deployed `.ts` file runs as ONE evaluate call, creates agents/tools via endowed API functions, and they're registered in the Go-side registries.

## Proven Results

| Test | Result |
|------|--------|
| Lockdown works in QuickJS | PASS |
| Compartment evaluates with endowed globals | PASS |
| Compartments have separate globals | PASS |
| Outer globals NOT accessible from compartment | PASS |
| Endowed globals isolated between compartments | PASS |
| Shared API functions work through endowments | PASS |
| Same variable names in different compartments — zero collision | PASS |
| 50 compartments created and evaluated | PASS |
| Compartments GC'd after dropping reference | PASS |
| `harden()` freezes endowed API — mutation blocked | PASS |

## Architecture Implications

### How it works for Brainkit

```
1. Kit loads Mastra bundle (once, in main context)
2. Kit loads SES + calls lockdown() (once)
3. Kit creates shared API object with harden():
   const kitAPI = harden({ agent, createTool, createMemory, ... });

4. Deploy "team.ts":
   const c = new Compartment({ globals: kitAPI });
   c.evaluate(fileContent);
   // Resources tracked in Go-side registries

5. Teardown "team.ts":
   // Drop compartment reference
   // Clean Go-side registry entries for this source
   // GC collects the compartment and everything in it
```

### What the developer writes

```typescript
// team.ts — runs inside its own Compartment
const leader = agent({ name: "leader", model: "gpt-4o" });
const coder = agent({ name: "coder", model: "gpt-4o-mini" });
const searchTool = createTool({ id: "search", ... });

// These register in Go-side registries via endowed API
// No global pollution — everything scoped to this compartment
```

### What plugins do

```go
// Plugin deploys .ts files via SDK
client.Deploy(ctx, "agents/reviewer.ts", tsCode)
// Kit creates a Compartment, evaluates the code

// Later — teardown
client.Teardown(ctx, "agents/reviewer.ts")
// Kit drops the Compartment, cleans Go registries
```

### What WASM shards do

WASM shards can deploy .ts configurations too:
```assemblyscript
bus.askAsync("kit.deploy", '{"source":"agents/helper.ts","code":"..."}', "onDeployed")
```

This enables WASM automation modules to dynamically build agent teams — the infrastructure builder pattern.

## Files

| File | Purpose | Tests |
|------|---------|-------|
| `lifecycle_test.go` | QuickJS primitives: delete, GC, closures, deploy/teardown | 10 |
| `scoped_lifecycle_test.go` | Cleanup hooks, scope(), nested scopes, redeploy | 9 |
| `user_resources_test.go` | Global snapshot, onTeardown(), timer tracking | 6 |
| `sandbox_test.go` | Sandboxing: IIFE, Proxy, with(proxy), separate contexts | 7 |
| `sandbox_edge_cases_test.go` | Edge cases: this, typeof, const/let, strict mode, async | 11 |
| `multicontext_test.go` | Multiple QuickJS contexts, Go bridges, performance | 8 |
| `multicontext_bug_test.go` | Bug reproductions for context close behavior | 6 |
| `multicontext_bug2_test.go` | Exact pattern reproduction, confirmed no bug | 4 |
| `ses_test.go` | **SES Compartments: lockdown, isolation, shared API, stress** | **10** |
| `kit_lifecycle_test.go` | Real Kit integration: agents, tools, WASM, bus subs | 8 |
| **Total** | | **80** |

## Decision

**SES Compartments** is the production solution for file-level isolation in Brainkit.

Combined with cleanup hooks on our API functions (for Go-side registry management), this gives us:
- True JS isolation between deployed files (no global collision)
- Shared API via hardened endowments (agent, createTool, etc.)
- Lightweight teardown (drop compartment reference + clean Go registries)
- Strict mode compatible
- Production-grade (used by MetaMask)
- 220KB additional bundle size
- Two small polyfills for QuickJS compatibility
