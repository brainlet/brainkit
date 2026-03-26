// Test: createScorer with preprocess + analyze + generateScore chain
import { createScorer } from "agent";
import { output } from "kit";

const scorer = createScorer({
  id: "word-overlap-scorer",
  name: "Word Overlap Scorer",
  description: "Measures word overlap between input and output",
}).preprocess(({ run }: any) => {
  const inputText = run.input?.[0]?.content || "";
  const outputText = run.output?.text || "";
  return {
    inputWords: new Set(inputText.toLowerCase().split(/\s+/)),
    outputWords: new Set(outputText.toLowerCase().split(/\s+/)),
  };
}).analyze(({ results }: any) => {
  const inputWords: Set<string> = results.preprocessStepResult?.inputWords || new Set();
  const outputWords: Set<string> = results.preprocessStepResult?.outputWords || new Set();
  const overlap = [...inputWords].filter(w => outputWords.has(w));
  return {
    overlapCount: overlap.length,
    inputCount: inputWords.size,
  };
}).generateScore(({ results }: any) => {
  const total = results.analyzeStepResult?.inputCount || 0;
  const overlap = results.analyzeStepResult?.overlapCount || 0;
  return total > 0 ? overlap / total : 0;
});

const result = await scorer.run({
  input: [{ role: "user", content: "Tell me about Go programming language" }],
  output: { role: "assistant", text: "Go is a programming language created by Google" },
});

output({
  hasScore: typeof result.score === "number",
  scorePositive: result.score > 0,
  scoreInRange: result.score >= 0 && result.score <= 1,
});
