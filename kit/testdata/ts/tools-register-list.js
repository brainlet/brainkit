// Test: tools.register() and tools.list() from JS
import { tools, output } from "kit";

// Register a tool
await tools.register("my_calculator", {
  description: "Performs basic math",
  inputSchema: { type: "object", properties: { a: { type: "number" }, b: { type: "number" } } },
});

// List tools
const allTools = await tools.list();

output({
  registered: true,
  toolCount: allTools.length,
  found: allTools.some(t => t.shortName === "my_calculator"),
  names: allTools.map(t => t.shortName),
});
