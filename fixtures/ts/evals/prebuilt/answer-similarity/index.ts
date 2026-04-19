// Test: createAnswerSimilarityScorer — compares output against ground truth.
import { createAnswerSimilarityScorer } from "agent";
import { model, output } from "kit";

const s = createAnswerSimilarityScorer({ model: model("openai", "gpt-4o-mini") });
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
