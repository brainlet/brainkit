// Test: createToxicityScorer — flags toxic output.
import { createToxicityScorer } from "agent";
import { model, output } from "kit";

const s = createToxicityScorer({ model: model("openai", "gpt-4o-mini") });
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
