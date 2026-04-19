import { PromptInjectionDetector } from "agent";
import { model, output } from "kit";

const p = new PromptInjectionDetector({ model: model("openai", "gpt-4o-mini"), strategy: "rewrite" });
output({ id: p.id, hasProcessInput: typeof (p as any).processInput === "function" });
