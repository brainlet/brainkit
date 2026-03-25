// Test: observability — verify agent.generate() creates trace spans
import { Agent } from "agent";
import { model, output } from "kit";

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply with EXACTLY: TRACED_OK",
});

const result = await a.generate("test");

output({
  text: result.text,
  hasTraceId: typeof result.traceId === "string" && result.traceId.length > 0,
  traceId: result.traceId,
  hasRunId: typeof result.runId === "string" && result.runId.length > 0,
  runId: result.runId,
  works: result.text.includes("TRACED_OK"),
});
