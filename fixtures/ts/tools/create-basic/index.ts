// Test: createTool + kit.register + tools.call roundtrip
import { createTool, z } from "agent";
import { kit, tools, output } from "kit";

const adder = createTool({
  id: "adder",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ sum: a + b }),
});

kit.register("tool", "adder", adder);
const result = await tools.call("adder", { a: 10, b: 32 });
output({ sum: (result as any).sum, registered: true });
