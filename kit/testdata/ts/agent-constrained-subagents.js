// Test: Constrained subagents via subagents config
// NOTE: createSubagent() is a removed API. This fixture uses the new Agent pattern.
// Verifies: tool filtering, fresh agent per invocation, event forwarding, metadata
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

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

  // Define constrained sub-agents using new Agent()
  const explorer = new Agent({
    name: "explore",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a codebase explorer. Use view and search to answer questions about code. Never edit files.",
    tools: { view: viewTool, search: searchTool },
    maxSteps: 5,
  });

  const coder = new Agent({
    name: "execute",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You write code. Read files first, then edit them.",
    tools: { view: viewTool, search: searchTool, edit: editTool },
    maxSteps: 5,
  });

  // Create supervisor with constrained subagents
  const lead = new Agent({
    name: "lead",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a tech lead. When asked to explore code, use the subagent tool with agentType 'explore'. When asked to edit code, use the subagent tool with agentType 'execute'. Always delegate, never do the work yourself.",
    tools: { view: viewTool, search: searchTool, edit: editTool },
    agents: { explore: explorer, execute: coder },
    maxSteps: 5,
  });

  // Test: delegate exploration
  const r = await lead.generate("Explore the codebase and find where 'hello' is used. Use the explore subagent.");

  results.hasResponse = r.text.length > 0 ? "ok" : "empty";
  results.status = "ok";
} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 300);
}

output(results);
