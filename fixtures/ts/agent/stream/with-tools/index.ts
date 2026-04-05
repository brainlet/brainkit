// Test: agent.stream() with tool calls mid-stream
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const addTool = createTool({
  id: "add",
  description: "Add two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }: any) => ({ result: a + b }),
});

const agent = new Agent({
  name: "stream-tool-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Use the add tool when asked math questions. Return only the number.",
  tools: { add: addTool },
  maxSteps: 3,
});

const stream = await agent.stream("What is 17 + 25? Use the add tool.");
const chunks: string[] = [];
for await (const chunk of stream.textStream) {
  chunks.push(chunk);
}
const text = await stream.text;

output({
  hasText: text.length > 0,
  hasChunks: chunks.length > 0,
  containsAnswer: text.includes("42"),
});
