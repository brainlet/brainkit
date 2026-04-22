// Test: register then unregister a tool.
// Exercises MCP-sourced tool metadata (MCPToolProperties,
// ToolAnnotations) and the ValidationError envelope — both
// part of the canonical @mastra/core/tools surface.
import { createTool, z } from "agent";
import type { MCPToolProperties, ToolAnnotations, ValidationError } from "agent";
import { kit, tools, output } from "kit";

// Compile-time probe: MCPToolProperties carries annotations +
// free-form metadata exactly matching canonical.
const annotations: ToolAnnotations = {
  title: "Temporary tool",
  readOnlyHint: false,
  destructiveHint: false,
  idempotentHint: true,
  openWorldHint: false,
};
const mcpProps: MCPToolProperties = {
  toolType: "mcp",
  annotations,
  _meta: { source: "fixture" },
};
const _validationError: ValidationError | undefined = undefined;
void mcpProps;
void _validationError;

const temp = createTool<"temp_tool", { x: string }, { echo: string }>({
  id: "temp_tool",
  description: "Temporary tool",
  inputSchema: z.object({ x: z.string() }),
  outputSchema: z.object({ echo: z.string() }),
  execute: async ({ x }) => ({ echo: x }),
});

kit.register("tool", "temp_tool", temp);
const before = tools.list();
const foundBefore = before.some((t: any) => t.shortName === "temp_tool");

kit.unregister("tool", "temp_tool");
const after = tools.list();
const foundAfter = after.some((t: any) => t.shortName === "temp_tool");

output({ foundBefore, foundAfter, removed: foundBefore && !foundAfter });
