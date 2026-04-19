// Test: character chunking — fixed-size character windows with overlap.
import { MDocument } from "agent";
import { output } from "kit";

const text = "abcdefghijklmnopqrstuvwxyz0123456789".repeat(10);
const doc = MDocument.fromText(text);
const chunks = await doc.chunk({ strategy: "character", maxSize: 40, overlap: 5 });
output({
  chunkCount: chunks.length,
  multipleChunks: chunks.length > 1,
  allHaveText: chunks.every((c) => typeof c.text === "string" && c.text.length > 0),
});
