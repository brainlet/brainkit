// Test: createBiasScorer — detects biased opinions in LLM output.
import { createBiasScorer } from "agent";
import { model, output } from "kit";

const s = createBiasScorer({ model: model("openai", "gpt-4o-mini") });
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
