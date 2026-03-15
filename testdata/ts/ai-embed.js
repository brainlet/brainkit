// Test: ai.embed() and ai.embedMany() — embedding generation
import { ai, output } from "brainlet";

// Single embedding
const single = await ai.embed({
  model: "openai/text-embedding-3-small",
  value: "Hello world",
});

// Multiple embeddings
const multi = await ai.embedMany({
  model: "openai/text-embedding-3-small",
  values: ["Hello", "World", "Brainlet"],
});

output({
  single: {
    dimensions: single.embedding.length,
    hasValues: single.embedding.length > 0 && typeof single.embedding[0] === "number",
  },
  multi: {
    count: multi.embeddings.length,
    dimensions: multi.embeddings[0]?.length,
    allVectors: multi.embeddings.every(e => e.length > 0),
  },
});
