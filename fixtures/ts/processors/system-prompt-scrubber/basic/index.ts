import { SystemPromptScrubber } from "agent";
import { model, output } from "kit";

const p = new SystemPromptScrubber({ model: model("openai", "gpt-4o-mini") });
output({ id: p.id, hasProcessOutputResult: typeof (p as any).processOutputResult === "function" });
