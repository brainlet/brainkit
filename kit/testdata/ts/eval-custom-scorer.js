// Test: custom scorer + pre-built rule-based scorer with plain strings
// NOTE: scorers (pre-built) is a removed API. Only createScorer remains.
import { createScorer } from "agent";
import { output } from "kit";

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

// Test 2: custom content similarity scorer (replaces removed scorers.contentSimilarity)
const similarityScorer = createScorer({
  id: "content-similarity",
  description: "Checks if output matches input (simple equality check)",
})
  .generateScore(({ run }) => {
    const input = String(run.input).toLowerCase();
    const out = String(run.output).toLowerCase();
    return input === out ? 1 : 0;
  });

const r2 = await similarityScorer.run({
  input: "hello world",
  output: "hello world",
});

// Test 3: different strings
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
