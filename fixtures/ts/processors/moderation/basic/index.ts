// Test: ModerationProcessor construct. model is LLM-gated.
import { ModerationProcessor } from "agent";
import { model, output } from "kit";

const p = new ModerationProcessor({ model: model("openai", "gpt-4o-mini") });
output({ id: p.id, hasProcessInput: typeof (p as any).processInput === "function" });
