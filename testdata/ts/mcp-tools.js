// Test: MCP client — list tools + call a tool from an MCP server
import { mcp, output } from "brainlet";

// List tools from all connected MCP servers
const tools = await mcp.listTools();

if (tools.length === 0) {
  output({ error: "No MCP tools found" });
} else {
  // Call the echo tool directly
  const echoResult = await mcp.callTool("test", "echo", { message: "hello from brainkit" });

  output({
    toolCount: tools.length,
    toolNames: tools.map(t => t.name).slice(0, 10),
    echoResult: echoResult,
    hasTools: tools.length > 0,
  });
}
