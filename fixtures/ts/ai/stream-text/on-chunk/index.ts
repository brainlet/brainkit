// Test: streamText with onChunk callback
import { streamText } from "ai";
import { model, output } from "kit";

const chunkTypes: string[] = [];

const result = streamText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Say hello in one word.",
  experimental_onChunk: (event: any) => {
    chunkTypes.push(event.type || "unknown");
  },
});

for await (const _ of result.textStream) {}
const text = await result.text;

output({
  hasText: text.length > 0,
  chunksFired: chunkTypes.length > 0,
  chunkCount: chunkTypes.length,
});
