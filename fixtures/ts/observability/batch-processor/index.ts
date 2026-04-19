// Real BatchSpanProcessor pipeline:
// BasicTracerProvider → BatchSpanProcessor → InMemorySpanExporter.
// Start/end a span, forceFlush, verify exporter captured it.
import {
  BasicTracerProvider,
  BatchSpanProcessor,
  InMemorySpanExporter,
} from "agent";
import { output } from "kit";

const exporter = new (InMemorySpanExporter as any)();
const processor = new (BatchSpanProcessor as any)(exporter, {
  maxExportBatchSize: 1,
  scheduledDelayMillis: 10,
});
const provider = new (BasicTracerProvider as any)({
  spanProcessors: [processor],
});

const tracer = provider.getTracer("brainkit-fixture");
const span = tracer.startSpan("bsp-span");
span.setAttribute("kind", "batch");
span.end();

await processor.forceFlush();
const finished = exporter.getFinishedSpans();

await processor.shutdown();
await provider.shutdown();

output({
  hasProcessor: typeof processor.forceFlush === "function",
  spanCount: finished.length,
  firstName: finished.length > 0 ? finished[0].name : "",
  firstKind: finished.length > 0
    ? (finished[0].attributes && finished[0].attributes.kind)
    : "",
});
