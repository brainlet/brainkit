// Test: createScorer with generateScore + generateReason chain
import { createScorer } from "agent";
import { output } from "kit";

const scorer = createScorer({
  id: "explained-scorer",
  name: "Explained Scorer",
  description: "Scores and explains the score",
}).generateScore(({ run }: any) => {
  const text = run.output?.text || "";
  const hasKeywords = ["Go", "language", "programming"].some(w => text.includes(w));
  return hasKeywords ? 1.0 : 0.0;
}).generateReason(({ results, run }: any) => {
  const score = results.generateScoreStepResult;
  if (score >= 1.0) {
    return "Output contains relevant programming keywords";
  }
  return "Output is missing expected programming keywords";
});

const result = await scorer.run({
  input: [{ role: "user", content: "Tell me about Go" }],
  output: { role: "assistant", text: "Go is a programming language by Google" },
});

output({
  hasScore: typeof result.score === "number",
  score: result.score,
  hasReason: typeof result.reason === "string" && result.reason.length > 0,
  reason: result.reason,
});
