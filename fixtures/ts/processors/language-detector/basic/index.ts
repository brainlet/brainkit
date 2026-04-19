import { LanguageDetector } from "agent";
import { model, output } from "kit";

const p = new LanguageDetector({ model: model("openai", "gpt-4o-mini"), allowedLanguages: ["en"] });
output({ id: p.id, hasProcessInput: typeof (p as any).processInput === "function" });
