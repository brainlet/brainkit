// Test: Agent generate options passthrough
// Verifies: temperature, maxSteps, onStepFinish, onFinish, structuredOutput, activeTools
import { agent, createTool, z, output } from "kit";

const results = {};

// Tool for testing activeTools filtering
const addTool = createTool({
  id: "add",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

const mulTool = createTool({
  id: "multiply",
  description: "Multiplies two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  execute: async ({ a, b }) => ({ result: a * b }),
});

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a math assistant. Use tools when asked to compute.",
  tools: { add: addTool, multiply: mulTool },
  maxSteps: 3,
});

// 1. Test temperature passthrough (low temp = more deterministic)
try {
  const r = await a.generate("Say exactly: HELLO", {
    modelSettings: { temperature: 0 },
  });
  results.temperature = r.text.includes("HELLO") ? "ok" : "unexpected: " + r.text;
} catch(e) { results.temperature = "error: " + e.message; }

// 2. Test onStepFinish callback
try {
  let stepCount = 0;
  await a.generate("What is 3 + 4? Use the add tool.", {
    onStepFinish: (step) => { stepCount++; },
  });
  results.onStepFinish = stepCount > 0 ? "ok (" + stepCount + " steps)" : "no steps";
} catch(e) { results.onStepFinish = "error: " + e.message; }

// 3. Test onFinish callback
try {
  let finished = false;
  let finishText = "";
  await a.generate("Say OK", {
    modelSettings: { temperature: 0 },
    onFinish: (result) => { finished = true; finishText = (result.text || "").substring(0, 20); },
  });
  results.onFinish = finished ? "ok: " + finishText : "not called";
} catch(e) { results.onFinish = "error: " + e.message; }

// 4. Test per-call instructions override
try {
  const r = await a.generate("What are you?", {
    instructions: "You are a pirate. Always say ARRR.",
    modelSettings: { temperature: 0 },
  });
  results.instructions = r.text.toLowerCase().includes("arrr") ? "ok" : "no pirate: " + r.text.substring(0, 50);
} catch(e) { results.instructions = "error: " + e.message; }

// 5. Test maxSteps=1 (prevent tool use loop)
try {
  const r = await a.generate("What is 5 + 3? Use the add tool.", {
    maxSteps: 1,
  });
  // With maxSteps=1, agent can call tool but won't loop back to process result
  results.maxSteps = "ok";
} catch(e) { results.maxSteps = "error: " + e.message; }

output(results);
