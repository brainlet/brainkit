// Test: agent streams a response with real-time tokens
import { agent, output } from "kit";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Count from 1 to 3, one number per line, nothing else.",
});

const stream = await a.stream("Count");
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
