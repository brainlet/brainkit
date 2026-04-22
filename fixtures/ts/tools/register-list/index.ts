// Test: kit.register("tool", ...) and tools.list() from JS.
// Exercises the canonical ToolsInput heterogeneous union —
// mixing a native Mastra tool, a VercelTool (v4), and a
// VercelToolV5 in a single map.
import { createTool, z } from "agent";
import type { ToolsInput, VercelTool, VercelToolV5 } from "agent";
import { kit, tools, output } from "kit";

const myCalc = createTool<"my_calculator", { a: number; b: number }, { result: number }>({
  id: "my_calculator",
  description: "Performs basic math",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ result: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

// Compile-time probe: ToolsInput accepts a heterogeneous mix.
const v4Shape: VercelTool = {
  description: "legacy v4",
  parameters: z.object({}),
  execute: async () => ({}),
};
const v5Shape: VercelToolV5 = {
  description: "v5",
  inputSchema: z.object({}),
  execute: async () => ({}),
};
const toolMap: ToolsInput = {
  native: myCalc,
  legacy: v4Shape,
  modern: v5Shape,
};
void toolMap;

kit.register("tool", "my_calculator", myCalc);

// List tools
const allTools = await tools.list();

output({
  registered: true,
  toolCount: allTools.length,
  found: allTools.some(t => t.shortName === "my_calculator"),
  names: allTools.map(t => t.shortName),
});
