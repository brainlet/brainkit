// Test: direct AI streaming without an agent (LOCAL)
import { ai, output } from "brainlet";

const stream = await ai.stream({
  model: "openai/gpt-4o-mini",
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
