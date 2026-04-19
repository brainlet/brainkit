// Test: rerank() surface with an AI model as provider.
// Full LLM round-trip through rerank is deferred — we've seen the
// native path crash QuickJS when LLM call-backs re-enter the bundle
// from inside MastraAgentRelevanceScorer. Surface check is enough
// to lock in the call signature until that lane is hardened.
import { rerank } from "agent";
import { model, output } from "kit";

const m = model("openai", "gpt-4o-mini");
output({
  hasRerank: typeof rerank === "function",
  modelResolves: m !== null && m !== undefined,
});
