// `simulateStreamingMiddleware()` upgrades a non-streaming model
// into a streaming one: the generate call runs, the full text is
// captured, then replayed through the stream interface. Useful when
// a downstream consumer expects a stream but the provider doesn't
// support one (or you want deterministic test output).
import {
  streamText,
  wrapLanguageModel,
  simulateStreamingMiddleware,
} from "ai";
import { model, output } from "kit";

const base = model("openai", "gpt-4o-mini");
const streaming = wrapLanguageModel({
  model: base,
  middleware: simulateStreamingMiddleware(),
});

const result = streamText({
  model: streaming,
  prompt: "Return exactly the single word: hello",
});

let chunks = 0;
let full = "";
for await (const delta of result.textStream) {
  chunks++;
  full += delta;
  if (chunks > 200) break;
}

const finishReason = await result.finishReason;

output({
  chunkCount: chunks > 0,
  hasText: full.length > 0,
  finishReason: String(finishReason),
  containsHello: full.toLowerCase().includes("hello"),
});
