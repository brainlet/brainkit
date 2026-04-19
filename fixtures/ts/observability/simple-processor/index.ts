// Test: SimpleSpanProcessor — synchronous per-span export. Surface
// check: onStart/onEnd/forceFlush/shutdown present, exporter wired.
import {
  SimpleSpanProcessor,
  InMemorySpanExporter,
} from "agent";
import { output } from "kit";

const exporter = new (InMemorySpanExporter as any)();
const simple = new (SimpleSpanProcessor as any)(exporter);

let flushed = false;
let shutdown = false;
try {
  await simple.forceFlush();
  flushed = true;
  await simple.shutdown();
  shutdown = true;
} catch (_e) {
  // surface-only
}

output({
  hasOnStart: typeof (simple as any).onStart === "function",
  hasOnEnd: typeof (simple as any).onEnd === "function",
  hasForceFlush: typeof (simple as any).forceFlush === "function",
  hasShutdown: typeof (simple as any).shutdown === "function",
  exporterHasFinished: typeof (exporter as any).getFinishedSpans === "function",
});
