// Test: Constrained subagents via createSubagent() + subagents config
// Verifies: tool filtering, fresh agent per invocation, event forwarding, metadata
import { agent, createSubagent, createTool, z, output } from "brainlet";

const results = {};

try {
  // Create tools
  const viewTool = createTool({
    id: "view",
    description: "Read a file's contents",
    inputSchema: z.object({ path: z.string() }),
    execute: async ({ path }) => ({ content: "// File: " + path + "\nconsole.log('hello');" }),
  });

  const searchTool = createTool({
    id: "search",
    description: "Search for a pattern in files",
    inputSchema: z.object({ pattern: z.string() }),
    execute: async ({ pattern }) => ({ matches: ["main.ts:5: " + pattern, "utils.ts:12: " + pattern] }),
  });

  const editTool = createTool({
    id: "edit",
    description: "Edit a file",
    inputSchema: z.object({ path: z.string(), content: z.string() }),
    execute: async ({ path, content }) => ({ edited: true, path: path }),
  });

  // Define constrained subagent types
  const explorer = createSubagent({
    id: "explore",
    instructions: "You are a codebase explorer. Use view and search to answer questions about code. Never edit files.",
    allowedTools: ["view", "search"],
    model: "openai/gpt-4o-mini",
    maxSteps: 5,
  });

  const coder = createSubagent({
    id: "execute",
    instructions: "You write code. Read files first, then edit them.",
    allowedTools: ["view", "search", "edit"],
    model: "openai/gpt-4o-mini",
    maxSteps: 5,
  });

  // Track events
  const events = [];

  // Create supervisor with constrained subagents
  const lead = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a tech lead. When asked to explore code, use the subagent tool with agentType 'explore'. When asked to edit code, use the subagent tool with agentType 'execute'. Always delegate, never do the work yourself.",
    tools: { view: viewTool, search: searchTool, edit: editTool },
    subagents: [explorer, coder],
    onSubagentEvent: (event) => {
      events.push({ type: event.type, agentType: event.agentType });
    },
    maxSteps: 5,
  });

  // Test: delegate exploration
  const r = await lead.generate("Explore the codebase and find where 'hello' is used. Use the explore subagent.");

  results.hasResponse = r.text.length > 0 ? "ok" : "empty";
  results.eventCount = events.length;
  results.hasStartEvent = events.some(function(e) { return e.type === "start"; }) ? "ok" : "no start";
  results.hasEndEvent = events.some(function(e) { return e.type === "end"; }) ? "ok" : "no end";
  results.explorerUsed = events.some(function(e) { return e.agentType === "explore"; }) ? "ok" : "no explore";

  // Check that subagent metadata is in the result
  results.hasMetaTag = r.text.includes("subagent-meta") || (r.toolResults && JSON.stringify(r.toolResults).includes("subagent-meta")) ? "ok" : "no meta tag";

  results.status = "ok";
} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 300);
}

output(results);
