// Test: dynamic tools resolver — tools computed per-request from RequestContext
import { agent, createTool, RequestContext, z, output } from "brainlet";

const addTool = createTool({
  id: "add",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

const multiplyTool = createTool({
  id: "multiply",
  description: "Multiplies two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a * b }),
});

// Agent with dynamic tools — which tools depend on requestContext
const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Use the available tool to compute the result. Return ONLY the number.",
  tools: ({ requestContext }) => {
    const mode = requestContext.get("mode");
    if (mode === "multiply") return { multiply: multiplyTool };
    return { add: addTool };
  },
});

// Call with "add" mode
const ctx1 = new RequestContext([["mode", "add"]]);
const r1 = await a.generate("What is 3 + 4?", { requestContext: ctx1 });

// Call with "multiply" mode
const ctx2 = new RequestContext([["mode", "multiply"]]);
const r2 = await a.generate("What is 3 * 4?", { requestContext: ctx2 });

output({
  addResult: r1.text,
  multiplyResult: r2.text,
  addCorrect: r1.text.includes("7"),
  multiplyCorrect: r2.text.includes("12"),
});
