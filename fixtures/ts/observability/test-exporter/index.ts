// `TestExporter` captures tracing events in-memory so tests can
// assert span shape after a run. It's the right tool for verifying
// "did the agent emit a tool_call span?" without a real OTLP
// collector. Observability holds a registry of named instances; each
// instance is what actually knows how to start spans. The exporter
// sits behind an instance — we pull the default instance out, start
// a span, end it, flush, then query the exporter's buffers.
import { Observability, TestExporter } from "agent";
import { output } from "kit";

const testExporter = new (TestExporter as any)();

const obs: any = new (Observability as any)({
  configs: {
    default: {
      serviceName: "test-exporter-fixture",
      exporters: [testExporter],
    },
  },
});

const instance = obs.getDefaultInstance();
const span = instance.startSpan({
  name: "brainkit.test.span",
  type: "GENERIC",
  attributes: { kind: "fixture" },
});
span.end({ attributes: { result: "done" } });

// Flush registered instances + their exporters.
if (typeof obs.shutdown === "function") {
  await obs.shutdown();
}

const allSpans: any[] = testExporter.getAllSpans?.() ?? [];
const completed: any[] = testExporter.getCompletedSpans?.() ?? [];
const events: any[] = testExporter.events ?? [];

const ourSpan = allSpans.find(
  (s: any) => (s?.name ?? s?.spanName) === "brainkit.test.span",
);

output({
  instanceResolved: !!instance,
  eventsCount: events.length > 0,
  allSpansCount: allSpans.length > 0,
  completedCount: completed.length > 0,
  foundOurSpan: !!ourSpan,
  spanName: (ourSpan?.name ?? ourSpan?.spanName) || "",
});
