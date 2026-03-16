// Test: runEvals() batch evaluation
// Verifies: batch scoring against a dataset with a custom scorer
import { agent, createScorer, runEvals, output } from "brainlet";

const results = {};

try {
  // Create a simple agent
  const a = agent({
    model: "openai/gpt-4o-mini",
    instructions: "Answer questions concisely in one sentence.",
  });

  // Create a scorer that checks if output contains the ground truth.
  // NOTE: generateScore receives context with { run } where run = { input, output, groundTruth }.
  const accuracyScorer = createScorer({
    id: "accuracy",
    description: "Checks if the answer contains the expected content",
  }).generateScore(function(context) {
    var output = "";
    var truth = "";
    try {
      output = (context.run.output || "").toLowerCase();
      truth = (context.run.groundTruth || "").toLowerCase();
    } catch(e) {
      // Fallback: try direct properties
      output = (context.output || "").toLowerCase();
      truth = (context.groundTruth || "").toLowerCase();
    }
    if (!truth) return 1;
    return output.includes(truth) ? 1 : 0;
  });

  // Verify scorer works standalone first
  try {
    const testResult = await accuracyScorer.run({ input: "test", output: "the answer is 4", groundTruth: "4" });
    results.scorerStandalone = "ok: score=" + testResult.score;
  } catch(e) {
    results.scorerStandalone = "error: " + e.message;
  }

  // Run batch evaluation
  const evalResults = await runEvals({
    target: a,
    data: [
      { input: "What is 2 + 2?", groundTruth: "4" },
      { input: "What is the capital of France?", groundTruth: "paris" },
      { input: "What color is the sky on a clear day?", groundTruth: "blue" },
    ],
    scorers: [accuracyScorer],
    concurrency: 1,
  });

  results.hasScores = evalResults.scores ? "ok" : "no scores";
  results.totalItems = evalResults.summary?.totalItems || 0;

  // Verify scores
  results.scoreKeys = Object.keys(evalResults.scores || {}).join(",");
  results.accuracyScore = evalResults.scores?.accuracy;
  results.hasAccuracy = typeof evalResults.scores?.accuracy === "number" ? "ok" : "not a number";

  results.status = "ok";
} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 500);
  // Try to get the cause chain
  if (e.cause) {
    results.cause = typeof e.cause === "object" ? (e.cause.message || JSON.stringify(e.cause).substring(0, 200)) : String(e.cause);
  }
}

output(results);
