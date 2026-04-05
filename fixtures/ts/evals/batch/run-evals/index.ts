// Test: runEvals — batch evaluation with multiple scorers
import { createScorer } from "agent";
import { output } from "kit";

const lengthScorer = createScorer({
  id: "length",
  name: "Length Scorer",
  description: "Scores by output length",
}).generateScore(({ run }: any) => {
  const text = run.output?.text || "";
  return Math.min(text.length / 50, 1);
});

const keywordScorer = createScorer({
  id: "keyword",
  name: "Keyword Scorer",
  description: "Scores by keyword presence",
}).generateScore(({ run }: any) => {
  const text = (run.output?.text || "").toLowerCase();
  const keywords = ["go", "language", "programming"];
  const matches = keywords.filter(k => text.includes(k)).length;
  return matches / keywords.length;
});

// Run both scorers on the same input/output
const testInput = [{ role: "user", content: "Tell me about Go" }];
const testOutput = { role: "assistant", text: "Go is a programming language designed at Google." };

const r1 = await lengthScorer.run({ input: testInput, output: testOutput });
const r2 = await keywordScorer.run({ input: testInput, output: testOutput });

output({
  lengthScore: r1.score,
  keywordScore: r2.score,
  bothScored: typeof r1.score === "number" && typeof r2.score === "number",
  keywordPositive: r2.score > 0,
});
