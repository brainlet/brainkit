// Test: createAnswerRelevancyScorer — LLM-judge scorer factory from Mastra's
// prebuilt set. Surface-only: construct with a model, assert id + run() exist.
// Calling .run() would make an LLM request; we skip it to keep fixtures cheap.
import { createAnswerRelevancyScorer } from "agent";
import { model, output } from "kit";

const s = createAnswerRelevancyScorer({ model: model("openai", "gpt-4o-mini") });
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
