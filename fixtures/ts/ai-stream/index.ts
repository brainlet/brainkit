// Test: direct AI streaming without an agent (LOCAL)
import { streamText } from "ai";
import { model, output } from "kit";

const stream = await streamText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "Count from 1 to 3, one number per line, nothing else.",
});

const chunks = [];
for await (const chunk of stream.textStream) {
  chunks.push(chunk);
}
const text = await stream.text;

output({
  text: text,
  chunks: chunks.length,
  hasRealTimeTokens: chunks.length > 0,
});
