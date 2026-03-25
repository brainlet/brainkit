import { Agent, z } from "agent";
import { model, output } from "kit";
const agent = new Agent({ name: "structured", model: model("openai", "gpt-4o-mini"), instructions: "Extract structured data from user input." });
const result = await agent.generate("Bob is 30 years old and likes hiking and cooking.", { output: z.object({ name: z.string(), age: z.number(), hobbies: z.array(z.string()) }) });
output({ hasObject: result.object !== undefined, name: (result.object as any)?.name || "", age: (result.object as any)?.age || 0, hobbiesCount: Array.isArray((result.object as any)?.hobbies) ? (result.object as any).hobbies.length : 0 });
