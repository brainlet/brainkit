// Test: full composition — uses Agent, generateText, tools, z, createTool
// This is what a real developer .ts file looks like.
import { Agent, createTool, z } from "agent";
import { generateText } from "ai";
import { model, tools, output } from "kit";

// 1. Direct AI call (LOCAL)
const aiResult = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "What is 2+2? Reply with just the number.",
});

// 2. Call a Go tool through the bus (PLATFORM)
const reversed = await tools.call("reverse", { text: "brainlet" });

// 3. Create a local tool with Zod schema
const concatTool = createTool({
  id: "concat",
  description: "Concatenates two strings",
  inputSchema: z.object({
    a: z.string().describe("first string"),
    b: z.string().describe("second string"),
  }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

output({
  aiText: aiResult.text,
  reversed: (reversed as any).result,
  hasLocalTool: !!concatTool,
});
