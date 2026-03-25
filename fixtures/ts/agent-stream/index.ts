// Test: agent streams a response with real-time tokens
import { Agent } from "agent";
import { model, output } from "kit";

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
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
