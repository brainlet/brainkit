// Test: rerank() — uses an LLM as the semantic relevance provider.
// Passes synthetic vector search results through the rerank pipeline
// and asserts the top result carries a numeric score + detail object.
import { rerank } from "agent";
import { model, output } from "kit";

const results = [
  { id: "a", score: 0.9, metadata: { text: "Go is a compiled language from Google." } },
  { id: "b", score: 0.6, metadata: { text: "Rust guarantees memory safety without a GC." } },
  { id: "c", score: 0.5, metadata: { text: "Python is dynamically typed." } },
];

try {
  const reranked = await rerank(
    results as any,
    "What is Go?",
    model("openai", "gpt-4o-mini"),
    { topK: 2 },
  );
  output({
    called: true,
    count: reranked.length,
    topHasScore: reranked.length > 0 && typeof reranked[0].score === "number",
    topHasDetails: reranked.length > 0 && typeof reranked[0].details === "object",
  });
} catch (e: any) {
  output({ called: false, error: String(e?.message || e).substring(0, 200) });
}
