// Test: MDocument.fromJSON — parses a JSON string document, chunks it
// with the json strategy.
import { MDocument } from "agent";
import { output } from "kit";

const doc = MDocument.fromJSON(JSON.stringify({
  title: "brainkit",
  description: "A Go runtime that embeds QuickJS and Watermill.",
  topics: ["quickjs", "watermill", "ses"],
}));

const chunks = await doc.chunk({ strategy: "json", maxSize: 1000 });
output({
  chunkCount: chunks.length,
  firstChunkHasText: chunks.length > 0 && typeof chunks[0].text === "string",
  hasMetadata: chunks.length > 0 && typeof chunks[0].metadata === "object",
});
