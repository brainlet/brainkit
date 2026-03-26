// Test: rerank function availability
import { output } from "kit";

try {
  const hasRerank = typeof rerank === "function";
  const hasRerankWithScorer = typeof rerankWithScorer === "function";
  output({
    hasRerank,
    hasRerankWithScorer,
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 100) });
}
