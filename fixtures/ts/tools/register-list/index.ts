// Test: kit.register("tool", ...) and tools.list() from JS
import { createTool, z } from "agent";
import { kit, tools, output } from "kit";

// Register a tool using the new pattern
const myCalc = createTool({
  id: "my_calculator",
  description: "Performs basic math",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

kit.register("tool", "my_calculator", myCalc);

// List tools
const allTools = await tools.list();

output({
  registered: true,
  toolCount: allTools.length,
  found: allTools.some(t => t.shortName === "my_calculator"),
  names: allTools.map(t => t.shortName),
});
