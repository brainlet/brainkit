// `cosineSimilarity(a, b)` returns cosine of the angle between two
// embedding vectors. Values near 1 = very similar, near 0 = unrelated,
// near -1 = opposite. Used for ranking query results against stored
// embeddings without a dedicated vector DB.
import { cosineSimilarity } from "ai";
import { output } from "kit";

// Identical vectors → 1.
const identical = cosineSimilarity([1, 0, 0], [1, 0, 0]);
// Orthogonal vectors → 0.
const orthogonal = cosineSimilarity([1, 0, 0], [0, 1, 0]);
// Opposite vectors → -1.
const opposite = cosineSimilarity([1, 0, 0], [-1, 0, 0]);
// Mostly aligned → > 0.9.
const aligned = cosineSimilarity([1, 1, 0], [0.9, 1.1, 0.05]);

function approx(a: number, b: number, eps = 1e-6) {
  return Math.abs(a - b) < eps;
}

output({
  identicalIsOne: approx(identical, 1),
  orthogonalIsZero: approx(orthogonal, 0),
  oppositeIsMinusOne: approx(opposite, -1),
  alignedIsHigh: aligned > 0.9 && aligned <= 1,
});
