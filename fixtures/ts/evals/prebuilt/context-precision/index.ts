// Test: createContextPrecisionScorer — needs context array or extractor.
import { createContextPrecisionScorer } from "agent";
import { model, output } from "kit";

const s = createContextPrecisionScorer({
  model: model("openai", "gpt-4o-mini"),
  options: { context: ["doc-a", "doc-b"] },
});
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
