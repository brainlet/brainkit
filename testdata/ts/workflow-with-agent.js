// Test: workflow with an agent step — combines workflows + AI
import { createWorkflow, createStep, agent, z, output } from "kit";

// Step 1: prepare prompt
const prepareStep = createStep({
  id: "prepare",
  inputSchema: z.object({ topic: z.string() }),
  outputSchema: z.object({ prompt: z.string() }),
  execute: async ({ inputData }) => {
    return { prompt: `In exactly one word, what color is a ${inputData.topic}?` };
  },
});

// Step 2: ask the AI agent
const aiStep = createStep({
  id: "ask-ai",
  inputSchema: z.object({ prompt: z.string() }),
  outputSchema: z.object({ answer: z.string() }),
  execute: async ({ inputData }) => {
    const a = agent({
      model: "openai/gpt-4o-mini",
      instructions: "Reply with exactly one word. No punctuation.",
    });
    const result = await a.generate(inputData.prompt);
    return { answer: result.text.trim().toLowerCase() };
  },
});

// Step 3: format result
const formatStep = createStep({
  id: "format",
  inputSchema: z.object({ answer: z.string() }),
  outputSchema: z.object({ formatted: z.string() }),
  execute: async ({ inputData }) => {
    return { formatted: `The answer is: ${inputData.answer}` };
  },
});

const workflow = createWorkflow({
  id: "agent-workflow",
  inputSchema: z.object({ topic: z.string() }),
  outputSchema: z.object({ formatted: z.string() }),
})
  .then(prepareStep)
  .then(aiStep)
  .then(formatStep)
  .commit();

const run = await workflow.createRun();
const result = await run.start({ inputData: { topic: "banana" } });

output({
  status: result.status,
  result: result.result,
  hasAnswer: result.status === "success" && result.result?.formatted?.includes("The answer is:"),
});
