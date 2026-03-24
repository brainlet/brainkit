// Test: agent with input processor — adds context before LLM sees the messages
// NOTE: inputProcessors is a removed API. This fixture needs rewriting
// once the new processor pattern is defined in the Mastra Agent API.
import { Agent } from "agent";
import { model, output } from "kit";

// A simple input processor that tracks invocation and passes messages through.
// We verify the processor was called and the agent still works correctly.
let processorCalled = false;

const trackingProcessor = {
  id: "tracker",
  processInput: async ({ messages }) => {
    processorCalled = true;
    return messages;
  },
};

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply with EXACTLY: PROCESSOR_TEST_OK",
  // TODO: inputProcessors API removed — needs new Mastra pattern
  // inputProcessors: [trackingProcessor],
});

const result = await a.generate("test");

output({
  text: result.text,
  processorCalled: processorCalled,
  works: result.text.includes("PROCESSOR_TEST_OK"),
});
