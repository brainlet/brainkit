// Test: recursive chunking — splits on paragraph / sentence boundaries.
import { MDocument } from "agent";
import { output } from "kit";

const text = [
  "Paragraph one lays out the problem.",
  "Paragraph two continues with details and examples.",
  "Paragraph three explores edge cases and tradeoffs.",
  "Paragraph four draws conclusions and lists references.",
].join("\n\n");

const doc = MDocument.fromText(text);
const chunks = await doc.chunk({ strategy: "recursive", maxSize: 80, overlap: 10 });
output({
  chunkCount: chunks.length,
  allHaveText: chunks.every((c) => typeof c.text === "string" && c.text.length > 0),
  multipleChunks: chunks.length > 1,
});
