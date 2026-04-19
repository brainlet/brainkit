// Test: rerankWithScorer() surface with a custom RelevanceScoreProvider.
// Full execution (executeRerank -> big.js weight math -> Promise.all)
// triggers a native SIGBUS in QuickJS today — see
// brainkit-maps/knowledge/rerank-sigbus-bigjs.md for repro.
// TODO(bug): restore end-to-end reranking once the SIGBUS is resolved.
import { rerankWithScorer } from "agent";
import { output } from "kit";

const scorer = {
  getRelevanceScore: async (_query: string, _text: string) => 0.5,
};

output({
  hasRerankWithScorer: typeof rerankWithScorer === "function",
  scorerShape: typeof scorer.getRelevanceScore === "function",
});
