import { streamObject, z } from "ai";
import { model, output } from "kit";
const result = streamObject({
  model: model("openai", "gpt-4o-mini"),
  schema: z.object({ name: z.string(), age: z.number(), hobbies: z.array(z.string()) }),
  prompt: "Generate a profile for a fictional person named Bob who is 30.",
});
let partials = 0;
for await (const partial of result.partialObjectStream) { partials++; }
const final = await result.object;
output({ partials, hasName: typeof final.name === "string", hasAge: typeof final.age === "number", hasHobbies: Array.isArray(final.hobbies) });
