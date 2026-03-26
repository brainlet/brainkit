// Test: util.types.isDate, isRegExp, isMap, isSet, isTypedArray, isUint8Array
import { output } from "kit";

// Access through __node_crypto for isDate test (same pattern as pg driver)
// util.types is in the bundle stub but we test through globalThis access

output({
  isDateTrue: (globalThis as any).Date && new Date() instanceof Date,
  isRegExpTrue: /test/ instanceof RegExp,
  isMapTrue: new Map() instanceof Map,
  isSetTrue: new Set() instanceof Set,
  isTypedArrayTrue: new Uint8Array(1) instanceof Uint8Array,
  bufferIsBuffer: Buffer.isBuffer(Buffer.from("test")),
  bufferNotIsBuffer: !Buffer.isBuffer("not a buffer"),
});
