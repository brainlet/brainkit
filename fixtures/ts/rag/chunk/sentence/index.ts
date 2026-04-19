// Test: sentence chunking — splits on sentence boundaries.
import { MDocument } from "agent";
import { output } from "kit";

const text = [
  "brainkit embeds QuickJS.",
  "It runs SES compartments.",
  "Each .ts file becomes a sandboxed module.",
  "Watermill provides the message bus.",
  "Topics are sanitized per transport.",
].join(" ");

const doc = MDocument.fromText(text);
const chunks = await doc.chunk({ strategy: "sentence", maxSize: 60, overlap: 0 });
output({
  chunkCount: chunks.length,
  multipleChunks: chunks.length > 1,
  allHaveText: chunks.every((c) => typeof c.text === "string" && c.text.length > 0),
});
