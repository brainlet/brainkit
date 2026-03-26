// Test: streamText with onFinish callback
import { streamText } from "ai";
import { model, output } from "kit";

let finishEvent: any = null;

const result = streamText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Say exactly: callback test",
  onFinish: (event: any) => {
    finishEvent = {
      hasText: typeof event.text === "string" && event.text.length > 0,
      hasUsage: event.usage?.totalTokens > 0,
      finishReason: event.finishReason,
    };
  },
});

// Consume the stream to trigger onFinish
for await (const _ of result.textStream) {}
const text = await result.text;

// Small delay for onFinish to fire
await new Promise(r => setTimeout(r, 100));

output({
  text: text.substring(0, 100),
  callbackFired: finishEvent !== null,
  callbackHasText: finishEvent?.hasText || false,
  callbackHasUsage: finishEvent?.hasUsage || false,
});
