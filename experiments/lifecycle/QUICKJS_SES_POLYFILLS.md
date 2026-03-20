# QuickJS + SES Polyfills

Two polyfills are required to run SES `lockdown()` in QuickJS. Without these, lockdown crashes with `"not an object"` at `tameFauxDataProperty`.

## Polyfill 1: Console Methods

QuickJS has basic `console.log`/`console.warn`/`console.error` but lacks many methods that SES's `tameConsole` expects.

```javascript
if (typeof console === "undefined") globalThis.console = {};
if (!console._times) console._times = {};
[
  "log", "warn", "error", "info", "debug",
  "time", "timeEnd", "timeLog",
  "group", "groupEnd", "groupCollapsed",
  "assert", "count", "countReset",
  "dir", "dirxml", "table", "trace", "clear",
  "profile", "profileEnd", "timeStamp"
].forEach(function(m) {
  if (!console[m]) console[m] = function() {};
});
```

The `console._times` property is specifically needed because SES checks for it to detect Node.js's internal `SafeMap` class.

## Polyfill 2: Iterator Prototype Faux Data Properties

QuickJS implements [Iterator Helpers](https://github.com/tc39/proposal-iterator-helpers) and exposes `%IteratorPrototype%.constructor` and `%IteratorPrototype%[Symbol.toStringTag]` as accessor properties (getter + setter). These are "faux data properties" — accessors that emulate data properties to work around the override mistake.

SES's `tameFauxDataProperty` function tests these by:
1. Calling the getter without `this` → works
2. Calling the getter with `this === obj` → works
3. Calling the setter with `this === { __proto__: null }` → **CRASHES in QuickJS**

The setter in QuickJS's Iterator prototype expects `this` to be a proper Iterator-related object. When called on `{ __proto__: null }`, it throws `"not an object"` from native C code.

**Fix**: Convert these accessor properties to actual data properties before SES loads:

```javascript
(function() {
  var ai = [][Symbol.iterator]();
  var ip = Object.getPrototypeOf(Object.getPrototypeOf(ai)); // %IteratorPrototype%
  if (ip) {
    // Fix constructor
    var cd = Object.getOwnPropertyDescriptor(ip, 'constructor');
    if (cd && cd.get && !('value' in cd)) {
      Object.defineProperty(ip, 'constructor', {
        value: cd.get.call(ip),
        writable: true,
        enumerable: false,
        configurable: true
      });
    }
    // Fix Symbol.toStringTag
    var td = Object.getOwnPropertyDescriptor(ip, Symbol.toStringTag);
    if (td && td.get && !('value' in td)) {
      Object.defineProperty(ip, Symbol.toStringTag, {
        value: td.get.call(ip),
        writable: false,
        enumerable: false,
        configurable: true
      });
    }
  }
})();
```

## Additional Stubs

```javascript
if (typeof performance === "undefined") {
  globalThis.performance = { now: function() { return Date.now(); } };
}
if (typeof process === "undefined") {
  globalThis.process = { env: {}, versions: {} };
}
if (typeof SharedArrayBuffer === "undefined") {
  globalThis.SharedArrayBuffer = ArrayBuffer;
}
```

## Lockdown Options

```javascript
lockdown({
  errorTaming: "unsafe",       // Keep QuickJS error stacks as-is
  overrideTaming: "moderate",  // Best compatibility with existing code
  consoleTaming: "unsafe",     // Keep our console (not SES's safe console)
  evalTaming: "unsafe-eval",   // REQUIRED — safe-eval creates a Function-based
                               // evaluator that doesn't work in QuickJS
});
```

### Why `evalTaming: "unsafe-eval"`?

SES's default `safe-eval` creates a custom evaluator using `new Function()` that wraps code in a scope proxy. In QuickJS, this custom evaluator fails with `"not a function"` because the generated function doesn't properly bind to the compartment's global.

With `unsafe-eval`, compartments use the native `eval` function. This means:
- Each `evaluate()` call is independent (variables don't persist between calls)
- Code runs in the compartment's global scope
- This is fine for our use case — a deployed `.ts` file runs as one `evaluate()` call

## Verification

After applying polyfills, all SES features work:

```
lockdown()           → OK
new Compartment()    → OK
evaluate()           → OK (returns completion value)
harden()             → OK (objects become immutable)
Isolation            → OK (outer globals invisible)
50 compartments      → OK (stress test passes)
GC collection        → OK (compartments collected after drop)
```

## Where to Apply

These polyfills should be added to the agent-embed bundle (`agent-embed/bundle/entry.mjs` or a dedicated polyfill file) BEFORE the SES UMD is loaded. The SES UMD itself (220KB) gets bundled alongside the Mastra bundle.
