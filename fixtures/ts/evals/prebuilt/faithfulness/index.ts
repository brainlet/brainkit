// Test: createFaithfulnessScorer — verifies claims against provided context.
import { createFaithfulnessScorer } from "agent";
import { model, output } from "kit";

const s = createFaithfulnessScorer({ model: model("openai", "gpt-4o-mini") });
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
