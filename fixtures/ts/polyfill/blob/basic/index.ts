// Test: globalThis.Blob polyfill — size, type, arrayBuffer, text.
import { output } from "kit";

const blob = new Blob(["hello, ", "world"], { type: "text/plain" });
const text = await blob.text();
const ab = await blob.arrayBuffer();
const bytes = new Uint8Array(ab);

output({
  hasBlob: typeof Blob === "function",
  type: blob.type,
  text,
  byteCount: bytes.byteLength,
  firstByte: bytes[0], // 'h'
});
