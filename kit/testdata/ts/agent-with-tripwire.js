// Test: agent with tripwire — output processor always aborts
// NOTE: outputProcessors is a removed API. This fixture needs rewriting
// once the new processor pattern is defined in the Mastra Agent API.
import { Agent } from "agent";
import { model, output } from "kit";

// Output processor that ALWAYS blocks the response
const blockAllProcessor = {
  id: "block-all",
  processOutputResult: ({ messages, abort }) => {
    abort("All responses blocked for testing", { retry: false, metadata: { reason: "test" } });
    return messages;
  },
};

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Say hello",
  // TODO: outputProcessors API removed — needs new Mastra pattern
  // outputProcessors: [blockAllProcessor],
});

try {
  const result = await a.generate("Hi");
  // Tripwire fires — Mastra catches it and returns result with non-"stop" finishReason.
  // Depending on Mastra version, finishReason may be "tripwire" or "other".
  output({
    text: result.text,
    finishReason: result.finishReason,
    tripped: result.finishReason !== "stop",
  });
} catch(e) {
  // Tripwire may also throw in some configurations
  output({
    error: e.message,
    tripped: true,
  });
}
