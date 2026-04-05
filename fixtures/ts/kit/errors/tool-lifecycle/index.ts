import { kit, tools, output } from "kit";
import { createTool, z } from "agent";

const myTool = createTool({
  id: "lifecycle-tool",
  description: "test lifecycle",
  inputSchema: z.object({ x: z.number() }),
  execute: async ({ x }) => ({ doubled: (x as number) * 2 })
});
kit.register("tool", "lifecycle-tool", myTool);

const list = tools.list();
const found = list.some((t: any) => t.shortName === "lifecycle-tool");

const result = await tools.call("lifecycle-tool", { x: 21 });

output({ registered: true, found, result });
