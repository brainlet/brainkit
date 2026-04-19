// Test: createHallucinationScorer — detects claims contradicting context.
import { createHallucinationScorer } from "agent";
import { model, output } from "kit";

const s = createHallucinationScorer({
  model: model("openai", "gpt-4o-mini"),
  options: { context: ["Go was created at Google in 2007."] },
});
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
