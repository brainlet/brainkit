# Evals in Brainkit

Brainkit supports two evaluation modes: **live scoring** (per-call, fire-and-forget) and **batch evaluation** (run a dataset through an agent with scorers).

---

## Quick Start

### Live Scoring (per-call)

```ts
import { agent, createScorer } from "brainlet";

const qualityScorer = createScorer({ id: "quality" })
  .generateScore(({ run }) => {
    return run.output.length > 10 ? 1 : 0;
  });

const a = agent({
  model: "openai/gpt-4o-mini",
  scorers: { quality: { scorer: qualityScorer } },
});

// Scoring happens automatically on every generate()/stream() call
await a.generate("Hello");
```

### Batch Evaluation

```ts
import { agent, createScorer, runEvals } from "brainlet";

const a = agent({ model: "openai/gpt-4o-mini" });

const accuracy = createScorer({ id: "accuracy" })
  .generateScore(({ run }) => {
    const output = (run.output || "").toLowerCase();
    const truth = (run.groundTruth || "").toLowerCase();
    return output.includes(truth) ? 1 : 0;
  });

const results = await runEvals({
  target: a,
  data: [
    { input: "What is 2+2?", groundTruth: "4" },
    { input: "Capital of France?", groundTruth: "paris" },
  ],
  scorers: [accuracy],
});

console.log(results.scores);   // { accuracy: 0.95 }
console.log(results.summary);  // { totalItems: 2 }
```

---

## createScorer

Build a scorer pipeline with composable steps:

```ts
const scorer = createScorer({
  id: "relevance",
  description: "Checks answer relevance",
})
  .preprocess(({ run }) => {
    // Optional: extract/transform data before scoring
    return { cleanOutput: run.output.trim().toLowerCase() };
  })
  .generateScore(({ run, results }) => {
    // Required: return a number (0-1)
    const clean = results.preprocessStepResult.cleanOutput;
    return clean.includes(run.groundTruth) ? 1 : 0;
  })
  .generateReason(({ run, score }) => {
    // Optional: explain the score
    return score === 1 ? "Contains expected answer" : "Missing expected answer";
  });
```

### LLM-based Scoring (Judge Pattern)

```ts
const scorer = createScorer({
  id: "helpfulness",
  description: "LLM judges helpfulness",
  judge: { model: "openai/gpt-4o-mini" },
})
  .generateScore({
    description: "Rate helpfulness 0-1",
    outputSchema: z.object({ score: z.number() }),
    createPrompt: ({ run }) => `Rate helpfulness of this response (0-1):
      Question: ${run.input}
      Answer: ${run.output}
      Return JSON: { "score": <number> }`,
  });
```

---

## Pre-Built Scorers

### Rule-Based (no LLM needed)

```ts
import { scorers } from "brainlet";

scorers.completeness()        // Checks output completeness vs input
scorers.contentSimilarity()   // String similarity between output and ground truth
scorers.textualDifference()   // Levenshtein distance
scorers.keywordCoverage()     // Keyword overlap
scorers.tone()                // Sentiment analysis
```

### LLM-Based (require a model)

```ts
scorers.hallucination({ model: "openai/gpt-4o-mini" })
scorers.faithfulness({ model: "openai/gpt-4o-mini" })
scorers.toxicity({ model: "openai/gpt-4o-mini" })
scorers.bias({ model: "openai/gpt-4o-mini" })
scorers.answerRelevancy({ model: "openai/gpt-4o-mini" })
scorers.answerSimilarity({ model: "openai/gpt-4o-mini" })
scorers.contextPrecision({ model: "openai/gpt-4o-mini" })
scorers.contextRelevance({ model: "openai/gpt-4o-mini" })
scorers.noiseSensitivity({ model: "openai/gpt-4o-mini" })
scorers.promptAlignment({ model: "openai/gpt-4o-mini" })
scorers.toolCallAccuracy({ model: "openai/gpt-4o-mini" })
```

---

## runEvals()

Run a dataset through an agent and score every response.

```ts
const results = await runEvals({
  target: myAgent,
  data: [
    { input: "Question 1", groundTruth: "Expected answer 1" },
    { input: "Question 2", groundTruth: "Expected answer 2" },
  ],
  scorers: [scorer1, scorer2],
  concurrency: 3,                // parallel evaluations (default: 1)
  onItemComplete: ({ item, targetResult, scorerResults }) => {
    console.log(`Scored: ${item.input} → ${JSON.stringify(scorerResults)}`);
  },
});
```

| Field | Type | Description |
|-------|------|-------------|
| `target` | `Agent` | Agent to evaluate |
| `data` | `RunEvalsDataItem[]` | Dataset items with input + optional groundTruth |
| `scorers` | `ScorerBuilder[]` | Scorers to run on each output |
| `concurrency` | `number` | Max parallel evaluations (default: 1) |
| `targetOptions` | `GenerateOptions` | Options for every generate() call |
| `onItemComplete` | `function` | Callback after each item is scored |

Returns `{ scores: Record<string, number>, summary: { totalItems: number } }`.

---

## Testing

| Test | What it proves |
|------|---------------|
| `TestRunEvals` | Batch evaluation with 3 items, custom accuracy scorer, all matched ground truth |
| `eval-custom-scorer.js` | Custom scorer pipeline with generateScore + generateReason |
| `eval-llm-scorer.js` | LLM judge pattern with GPT-4o-mini structured output |
