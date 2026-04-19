// Test: createPromptAlignmentScorerLLM — scores intent + requirements fit.
import { createPromptAlignmentScorerLLM } from "agent";
import { model, output } from "kit";

const s = createPromptAlignmentScorerLLM({ model: model("openai", "gpt-4o-mini") });
output({
  id: (s as any).id,
  hasRun: typeof (s as any).run === "function",
});
