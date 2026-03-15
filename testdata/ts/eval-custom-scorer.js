// Test: custom scorer + pre-built rule-based scorer with plain strings
import { createScorer, scorers, output } from "brainlet";

// Test 1: custom scorer with bare function API
const keywordScorer = createScorer({
  id: "keyword-check",
  description: "Checks if the response mentions keywords",
})
  .generateScore(({ run }) => {
    const keywords = ["hello", "world"];
    const lower = String(run.output).toLowerCase();
    const matches = keywords.filter(k => lower.includes(k)).length;
    return matches / keywords.length;
  })
  .generateReason(({ score }) => {
    if (score >= 1) return "All keywords found";
    if (score > 0) return "Some keywords found";
    return "No keywords found";
  });

const r1 = await keywordScorer.run({
  input: "Say hello world",
  output: "hello world, how are you?",
});

// Test 2: pre-built scorer with plain strings (wrapper converts to MastraDBMessage format)
const similarityScorer = scorers.contentSimilarity();
const r2 = await similarityScorer.run({
  input: "hello world",
  output: "hello world",
});

// Test 3: pre-built scorer with different strings
const r3 = await similarityScorer.run({
  input: "hello world",
  output: "goodbye universe",
});

output({
  customScore: r1.score,
  customReason: r1.reason,
  similarityExact: r2.score,
  similarityDifferent: r3.score,
  allCorrect: r1.score === 1 && r2.score > r3.score,
});
