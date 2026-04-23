import { bus, output } from "kit";
import { Agent, createTool, z } from "agent";

// Create a tool. Destructures the validated input payload
// directly — the second argument to `execute` is the
// `ToolExecutionContext`, not a nested `context` field.
const greetTool = createTool({
  id: "greet",
  description: "greets someone",
  inputSchema: z.object({ name: z.string() }),
  execute: async ({ name }) => {
    return { greeting: "hello " + name };
  }
});

// Register via bus
bus.publish("incoming.agent-bus-test", { tool: "registered" });
output({ agentAvailable: typeof Agent === "function", toolCreated: true });
