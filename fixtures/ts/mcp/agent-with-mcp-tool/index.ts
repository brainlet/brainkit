import { mcp, output } from "kit";
const mcpTools = mcp.listTools();
const hasEcho = mcpTools.some((t: any) => t.name === "echo");
const result = await mcp.callTool("test", "echo", { message: "from mcp fixture" });
output({ mcpToolCount: mcpTools.length, hasEcho, echoResult: result });
