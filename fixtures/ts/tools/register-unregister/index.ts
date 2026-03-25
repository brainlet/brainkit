// Test: register then unregister a tool
import { createTool, z } from "agent";
import { kit, tools, output } from "kit";

const temp = createTool({
  id: "temp_tool",
  description: "Temporary tool",
  inputSchema: z.object({ x: z.string() }),
  execute: async ({ x }) => ({ echo: x }),
});

kit.register("tool", "temp_tool", temp);
const before = tools.list();
const foundBefore = before.some((t: any) => t.shortName === "temp_tool");

kit.unregister("tool", "temp_tool");
const after = tools.list();
const foundAfter = after.some((t: any) => t.shortName === "temp_tool");

output({ foundBefore, foundAfter, removed: foundBefore && !foundAfter });
