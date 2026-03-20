# Findings: What We Learned

## QuickJS Object Lifecycle (10 experiments)

1. **`Value.Delete(name)` works** — removes globals from Go, JS sees `undefined`
2. **GC reclaims deleted objects** — `RunGC()` frees memory after all refs removed
3. **Cross-references are correct** — A→B survives deleting A if B holds ref; both collectible after all refs removed
4. **Closures keep objects alive** — this is the critical rule. A bus callback closing over an agent keeps the agent alive even after `delete globalThis.agent`. Must unsubscribe to fully clean up.
5. **Deploy/teardown cycles work** — 100 cycles × 5 agents = 500 operations, zero leaks

## Cleanup Hook Pattern (9 experiments)

6. **Auto-registration works** — every `agent()`, `createTool()`, `bus.subscribe()` can silently register a cleanup function
7. **`scope(name, fn)`** — wraps code execution with automatic source tracking, returns handle with `.teardown()`
8. **Nested scopes** — inner teardown preserves outer. Proven with supervisor + workers pattern.
9. **Redeploy** — teardown old + deploy new = atomic swap, zero duplicates
10. **Mixed resources** — agent + tools + memory + subscriptions all cleaned in one `teardown()` call

## User Resource Tracking (6 experiments)

11. **Global snapshot** — diff `globalThis` before/after eval catches user-created globals
12. **`onTeardown()` hook** — user registers custom cleanup (like React useEffect)
13. **Timer wrapping** — replaced `setTimeout`/`setInterval` with source-tracked versions
14. **Combined approach** — all three layers work together, 30-cycle stress passes

## Sandboxing (18 experiments)

15. **IIFE scoping** — `const`/`let` don't leak (already how EvalTS works)
16. **`with(proxy)`** — transparent sandboxing works in non-strict mode but **FAILS in strict mode**. Dead end for TypeScript/modules.
17. **Separate QuickJS contexts** — full isolation, works with Go bridges, 20 simultaneous contexts proven. But requires Mastra bundle per context (16.5MB × N).
18. **Context close is safe** — closing one context while others are open works fine (earlier "crash" was a test assertion bug, not a QuickJS bug)

## SES Compartments (10 experiments)

19. **SES loads in QuickJS** — 220KB UMD bundle, `lockdown` and `Compartment` both available
20. **`lockdown()` works** with two polyfills:
    - Console stubs (groupCollapsed, etc.)
    - Iterator prototype faux data properties → real data properties
21. **Compartments evaluate correctly** with `evalTaming: "unsafe-eval"`
22. **Isolation is real** — outer globals invisible, vars don't leak between compartments
23. **Shared API via endowments** — `agent()`, `createTool()` passed as hardened endowments
24. **`harden()` works** — frozen objects can't be mutated by compartment code
25. **50 compartments in a loop** — stress test passes, correct results
26. **GC collects compartments** — drop reference, RunGC, memory freed

## Dead Ends

- **vm2** — deprecated, critical security vulnerabilities, Node.js only
- **ShadowRealm** — TC39 Stage 2.7, not in QuickJS yet
- **`with(proxy)`** — clever trick but strict mode forbids `with`. Not viable for TypeScript.
- **SES without polyfills** — crashes on Iterator Helpers faux data properties in QuickJS

## Key Insight

The "not an object" error from SES lockdown was caused by QuickJS's Iterator Helpers exposing `constructor` and `Symbol.toStringTag` as accessor properties with setters that crash when called on `{ __proto__: null }` objects. SES's `tameFauxDataProperty` calls these setters as part of intrinsic hardening. The fix: convert these to real data properties before SES loads. Two lines of polyfill code.

## Production Readiness

| Concern | Status |
|---------|--------|
| Isolation between files | Proven (SES Compartments) |
| Shared API without duplication | Proven (hardened endowments) |
| Global collision prevention | Proven (separate compartment globals) |
| Teardown / cleanup | Proven (cleanup hooks + drop compartment ref) |
| Memory management | Proven (GC collects unreferenced compartments) |
| Strict mode compatibility | Proven (SES runs in strict mode) |
| Stress testing | Proven (50 compartments, 100 deploy/teardown cycles) |
| Real Kit integration | Proven (8 tests with actual brainkit) |
| Bundle size impact | 220KB for SES (acceptable) |
| QuickJS compatibility | Proven (two small polyfills) |
