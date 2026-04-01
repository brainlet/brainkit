import { kit, tools, output } from "kit";
import { createTool, z } from "agent";

// Register 5 tools
for (let i = 0; i < 5; i++) {
  const t = createTool({
    id: `multi-tool-${i}`,
    description: `tool ${i}`,
    inputSchema: z.object({ x: z.number() }),
    execute: async ({ x }) => ({ result: (x as number) + i })
  });
  kit.register("tool", `multi-tool-${i}`, t);
}

const list = tools.list();
const registered = list.filter((t: any) => t.shortName.startsWith("multi-tool-")).length;

output({ registered, expected: 5, allFound: registered === 5 });
