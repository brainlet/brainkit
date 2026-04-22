// Test: createTool + kit.register + tools.call roundtrip.
// Uses explicit generics on createTool to exercise the canonical
// 7-generic signature from @mastra/core/tools/tool.ts. The
// resulting Tool<TIn, TOut, ...> flows straight into ToolAction —
// no `as` coercion needed — proving the class structurally
// implements ToolAction.
import { createTool, z } from "agent";
import type { Tool, ToolAction } from "agent";
import { kit, tools, output } from "kit";

const adder = createTool<"adder", { a: number; b: number }, { sum: number }>({
  id: "adder",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ sum: z.number() }),
  execute: async ({ a, b }) => ({ sum: a + b }),
});

// Compile-time probe: the canonical `Tool` class carries its
// schemas in its generic slots, and every Tool structurally
// implements ToolAction.
const toolRef: Tool<{ a: number; b: number }, { sum: number }> = adder;
const actionRef: ToolAction<{ a: number; b: number }, { sum: number }> = adder;
void toolRef;
void actionRef;

kit.register("tool", "adder", adder);
const result = await tools.call("adder", { a: 10, b: 32 });
output({ sum: (result as any).sum, registered: true });
