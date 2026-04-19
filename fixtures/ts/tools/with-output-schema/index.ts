// Test: tool with outputSchema — Mastra validates the result
// shape at call time.
import { createTool, z } from "agent";
import { output } from "kit";

const tool = createTool({
  id: "profile",
  description: "Return a user profile",
  inputSchema: z.object({ id: z.string() }),
  outputSchema: z.object({
    id: z.string(),
    name: z.string(),
    age: z.number(),
  }),
  execute: async ({ id }: any) => ({ id, name: "Bob", age: 30 }),
});

const res = await (tool as any).execute({ id: "u-1" });

output({
  id: res.id,
  name: res.name,
  age: res.age,
  schemaCarried: !!(tool as any).outputSchema,
});
