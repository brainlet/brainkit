package jsbridge

import quickjs "github.com/buke/quickjs-go"

// StructuredClonePolyfill provides globalThis.structuredClone.
//
// The previous JSON-round-trip shim lost binary data
// (ArrayBuffer / TypedArrays come back as `{}`), which broke any
// library that ships bytes across a LoopbackPort — most visibly pdfjs,
// whose main-thread fake worker serializes the PDF bytes via
// structuredClone before the worker side reads them.
//
// This version handles the common cases: primitives, Date, RegExp,
// Map, Set, Array, ArrayBuffer, all TypedArray variants, and plain
// objects. Circular references are resolved via a WeakMap cache. The
// second `options` arg (`{ transfer }`) is accepted but ignored —
// transfers behave as copies in a single-threaded runtime, which is
// still more correct than silently dropping bytes.
type StructuredClonePolyfill struct{}

// StructuredClone creates a structuredClone polyfill.
func StructuredClone() *StructuredClonePolyfill { return &StructuredClonePolyfill{} }

func (p *StructuredClonePolyfill) Name() string { return "structuredClone" }

func (p *StructuredClonePolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
(function() {
  function clone(value, seen) {
    if (value === null || value === undefined) return value;
    const t = typeof value;
    if (t === 'boolean' || t === 'number' || t === 'string' || t === 'bigint' || t === 'symbol') return value;
    if (t === 'function') return value;
    if (seen.has(value)) return seen.get(value);
    if (value instanceof Date) {
      const copy = new Date(value.getTime());
      seen.set(value, copy);
      return copy;
    }
    if (value instanceof RegExp) {
      const copy = new RegExp(value.source, value.flags);
      seen.set(value, copy);
      return copy;
    }
    if (value instanceof ArrayBuffer) {
      const copy = value.slice(0);
      seen.set(value, copy);
      return copy;
    }
    if (ArrayBuffer.isView(value)) {
      // TypedArray or DataView — clone the underlying buffer slice.
      const Ctor = value.constructor;
      if (Ctor === DataView) {
        const copy = new DataView(value.buffer.slice(value.byteOffset, value.byteOffset + value.byteLength));
        seen.set(value, copy);
        return copy;
      }
      const copy = new Ctor(value.length);
      copy.set(value);
      seen.set(value, copy);
      return copy;
    }
    if (value instanceof Map) {
      const copy = new Map();
      seen.set(value, copy);
      for (const [k, v] of value) copy.set(clone(k, seen), clone(v, seen));
      return copy;
    }
    if (value instanceof Set) {
      const copy = new Set();
      seen.set(value, copy);
      for (const v of value) copy.add(clone(v, seen));
      return copy;
    }
    if (Array.isArray(value)) {
      const copy = new Array(value.length);
      seen.set(value, copy);
      for (let i = 0; i < value.length; i++) copy[i] = clone(value[i], seen);
      return copy;
    }
    if (t === 'object') {
      const copy = {};
      seen.set(value, copy);
      for (const k of Object.keys(value)) copy[k] = clone(value[k], seen);
      return copy;
    }
    return value;
  }
  globalThis.structuredClone = function(value, _options) {
    return clone(value, new WeakMap());
  };
})();
`)
}
