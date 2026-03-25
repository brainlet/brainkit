// Test: MDocument text chunking with recursive strategy
import { MDocument } from "agent";
import { output } from "kit";

const text = `
Chapter 1: Introduction

Artificial intelligence has transformed the way we build software.
Modern AI systems can generate code, understand natural language,
and make decisions that previously required human expertise.

Chapter 2: Architecture

The architecture of an AI system typically involves three layers:
the data layer, the model layer, and the application layer.
Each layer serves a distinct purpose and has its own challenges.

Chapter 3: Implementation

Building an AI system requires careful planning. You need to
choose the right models, prepare your data, and design your
interfaces thoughtfully.
`.trim();

const doc = MDocument.fromText(text);
const chunks = await doc.chunk({
  strategy: "recursive",
  maxSize: 200,
  overlap: 20,
});

output({
  chunkCount: chunks.length,
  hasMultiple: chunks.length > 1,
  firstChunkText: chunks[0]?.text?.substring(0, 50),
  allHaveText: chunks.every(c => c.text && c.text.length > 0),
});
