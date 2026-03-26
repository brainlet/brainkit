// Test: createScorer — custom function-based scorer with the builder pattern
import { createScorer } from "agent";
import { output } from "kit";

// Build a scorer: length-based scoring
const scorer = createScorer({
  id: "length-scorer",
  name: "Length Scorer",
  description: "Scores based on output text length",
}).generateScore(({ run }: any) => {
  const text = run.output?.text || "";
  return Math.min(text.length / 50, 1);
});

// Run it with agent-style input/output
const result = await scorer.run({
  input: [{ role: "user", content: "What is Go?" }],
  output: { role: "assistant", text: "Go is a statically typed programming language." },
});

output({
  hasScore: typeof result.score === "number",
  scoreInRange: result.score >= 0 && result.score <= 1,
  hasRunId: typeof result.runId === "string" && result.runId.length > 0,
});
