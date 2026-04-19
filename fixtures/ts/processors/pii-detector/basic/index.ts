import { PIIDetector } from "agent";
import { model, output } from "kit";

const p = new PIIDetector({ model: model("openai", "gpt-4o-mini"), strategy: "redact", detectionTypes: ["email", "phone"] });
output({ id: p.id, hasProcessInput: typeof (p as any).processInput === "function" });
