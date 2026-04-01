import { bus, output } from "kit";
import { Agent, createTool, z } from "agent";

// Create a tool
const greetTool = createTool({
  id: "greet",
  description: "greets someone",
  inputSchema: z.object({ name: z.string() }),
  execute: async ({ context }) => {
    return { greeting: "hello " + context.name };
  }
});

// Register via bus
bus.publish("incoming.agent-bus-test", { tool: "registered" });
output({ agentAvailable: typeof Agent === "function", toolCreated: true });
