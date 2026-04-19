// Test: createContextRelevanceScorerLLM — judges retrieved context quality.
import { createContextRelevanceScorerLLM } from "agent";
import { model, output } from "kit";

const s = createContextRelevanceScorerLLM({
  model: model("openai", "gpt-4o-mini"),
  options: { context: ["retrieved-chunk-1", "retrieved-chunk-2"] },
});
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
