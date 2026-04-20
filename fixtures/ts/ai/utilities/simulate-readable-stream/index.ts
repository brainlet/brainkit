// `simulateReadableStream({chunks, initialDelayInMs, chunkDelayInMs})`
// builds a ReadableStream that emits each chunk in order. Useful for
// unit-testing stream consumers without a real model.
import { simulateReadableStream } from "ai";
import { output } from "kit";

const stream = simulateReadableStream({
  chunks: ["Hello", " ", "brainkit", "!"],
  initialDelayInMs: null,
  chunkDelayInMs: null,
});

const reader = stream.getReader();
const collected: string[] = [];
while (true) {
  const { value, done } = await reader.read();
  if (done) break;
  collected.push(value);
}

output({
  chunkCount: collected.length,
  joined: collected.join(""),
  firstChunk: collected[0],
  lastChunk: collected[collected.length - 1],
});
