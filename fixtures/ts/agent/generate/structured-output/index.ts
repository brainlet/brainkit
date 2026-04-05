// Test: agent with structured output schema
import { Agent, z } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "structured",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Extract structured data from user input. Return JSON.",
});

const result = await agent.generate("Bob is 30 years old and likes hiking and cooking.", {
  output: z.object({ name: z.string(), age: z.number(), hobbies: z.array(z.string()) }),
});

// Structured output may be in result.object or parsed from result.text
let parsed: any = result.object;
if (!parsed && result.text) {
  try { parsed = JSON.parse(result.text); } catch {}
}

output({
  hasData: parsed !== null && parsed !== undefined,
  name: parsed?.name || "",
  age: parsed?.age || 0,
  hobbiesCount: Array.isArray(parsed?.hobbies) ? parsed.hobbies.length : 0,
});
