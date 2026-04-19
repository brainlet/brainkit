// Test: rerankWithScorer() — custom RelevanceScoreProvider re-ranks
// synthetic vector search results. No model, no network; scorer is a
// plain text-overlap function.
import { rerankWithScorer } from "agent";
import { output } from "kit";

const scorer = {
  getRelevanceScore: async (query: string, text: string) => {
    const q = query.toLowerCase();
    const t = text.toLowerCase();
    const terms = q.split(/\s+/).filter((w) => w.length > 2);
    let hits = 0;
    for (const term of terms) if (t.includes(term)) hits++;
    return terms.length === 0 ? 0 : hits / terms.length;
  },
};

const results = [
  { id: "a", score: 0.5, metadata: { text: "Go is a compiled language from Google." } },
  { id: "b", score: 0.5, metadata: { text: "Rust guarantees memory safety without a GC." } },
  { id: "c", score: 0.5, metadata: { text: "Python is dynamically typed." } },
];

const reranked = await rerankWithScorer({
  results: results as any,
  query: "What is Go?",
  scorer,
  options: { topK: 2 },
});

output({
  count: reranked.length,
  topMatchesQuery:
    reranked.length > 0 &&
    String((reranked[0].result.metadata as any).text).toLowerCase().includes("go"),
  topHasScore: reranked.length > 0 && typeof reranked[0].score === "number",
  topHasDetails: reranked.length > 0 && typeof reranked[0].details === "object",
});
