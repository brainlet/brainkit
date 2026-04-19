// Test: custom span via Observability — construct a default-enabled
// Observability registry, get its selected instance, start a span,
// record an event + end. Asserts the span has an id and traceId.
import { Observability } from "agent";
import { output } from "kit";

const obs = new Observability({ default: { enabled: true } } as any);

const instance: any = obs.getDefaultInstance();
let spanShape = false;
let idShape = false;
let ended = false;
let errorMsg = "";
try {
  const span: any = instance.startSpan({
    type: "generic",
    name: "fixture.custom.span",
    input: { hello: "world" },
    attributes: {},
  } as any);
  spanShape = span != null;
  idShape = typeof span?.id === "string" && span.id.length > 0
    && typeof span?.traceId === "string" && span.traceId.length > 0;
  if (span?.update) span.update({ attributes: { step: "probe" } });
  if (span?.end) {
    span.end({ output: { ok: true } });
    ended = true;
  }
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({
  hasInstance: instance != null,
  spanShape,
  idShape,
  ended,
  errorMsg,
});
