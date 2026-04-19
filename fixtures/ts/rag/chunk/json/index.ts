// Test: json chunking — splits a structured JSON document into chunks
// keyed by top-level fields.
import { MDocument } from "agent";
import { output } from "kit";

const payload = JSON.stringify({
  name: "brainkit",
  description: "A Go runtime that embeds QuickJS and Watermill.",
  modules: ["engine", "bus", "runtime", "rag"],
  metadata: { version: "0.x", license: "MIT" },
});

const doc = MDocument.fromJSON(payload);
const chunks = await doc.chunk({ strategy: "json", maxSize: 512 });
output({
  chunkCount: chunks.length,
  atLeastOne: chunks.length >= 1,
  allHaveText: chunks.every((c) => typeof c.text === "string" && c.text.length > 0),
});
