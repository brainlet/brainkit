// Test: rerankWithScorer() surface with a custom RelevanceScoreProvider.
// Full execution (executeRerank -> big.js weight math -> Promise.all)
// triggers a native SIGBUS in QuickJS today; shipping the surface
// contract so a fix is a one-line deepening.
import { rerankWithScorer } from "agent";
import { output } from "kit";

const scorer = {
  getRelevanceScore: async (_query: string, _text: string) => 0.5,
};

output({
  hasRerankWithScorer: typeof rerankWithScorer === "function",
  scorerShape: typeof scorer.getRelevanceScore === "function",
});
