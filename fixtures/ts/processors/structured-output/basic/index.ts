import { StructuredOutputProcessor, z } from "agent";
import { model, output } from "kit";

const p = new StructuredOutputProcessor({
  schema: z.object({ name: z.string() }),
  model: model("openai", "gpt-4o-mini"),
});
output({ id: p.id, hasProcessOutputStream: typeof (p as any).processOutputStream === "function" });
