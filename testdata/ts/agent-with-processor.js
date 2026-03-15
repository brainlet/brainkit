// Test: agent with input processor — adds context before LLM sees the messages
import { agent, z, output } from "brainlet";

// Track whether the processor was called
let processorCalled = false;

// A simple input processor that tracks invocation and passes messages through.
// We verify the processor was called and the agent still works correctly.
const trackingProcessor = {
  id: "tracker",
  processInput: async ({ messages }) => {
    processorCalled = true;
    return messages;
  },
};

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Reply with EXACTLY: PROCESSOR_TEST_OK",
  inputProcessors: [trackingProcessor],
});

const result = await a.generate("test");

output({
  text: result.text,
  processorCalled: processorCalled,
  works: result.text.includes("PROCESSOR_TEST_OK") && processorCalled,
});
