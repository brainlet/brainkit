// Test: GraphRAG.query — build a tiny semantic graph, query with a fake
// embedding, assert the top result comes back with a score.
import { output } from "kit";

// threshold=0.3 so the edges survive cosine-similarity pruning between
// these overlapping synthetic vectors.
const g = new (GraphRAG as any)(3, 0.3);
const chunks = [
  { text: "Go is a compiled language", metadata: { id: "a" } },
  { text: "Go has goroutines and channels", metadata: { id: "b" } },
  { text: "Go tooling includes gofmt", metadata: { id: "c" } },
];
const embeddings = [
  { vector: [0.9, 0.3, 0.3] },
  { vector: [0.8, 0.4, 0.3] },
  { vector: [0.7, 0.5, 0.3] },
];
g.createGraph(chunks, embeddings);

const results = g.query({ query: [0.9, 0.3, 0.3], topK: 2, randomWalkSteps: 10, restartProb: 0.2 });
output({
  hasNodes: g.getNodes().length === 3,
  hasEdges: g.getEdges().length > 0,
  topCount: results.length,
  topHasScore: results.length > 0 && typeof results[0].score === "number",
});
