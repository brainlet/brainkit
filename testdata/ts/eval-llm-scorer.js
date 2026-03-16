// Test: LLM-based custom scorer with judge model (createPrompt pattern)
import { createScorer, output } from "brainlet";

try {
  const helpfulnessScorer = createScorer({
    id: "helpfulness",
    description: "Evaluates how helpful a response is",
    judge: {
      model: "openai/gpt-4o-mini",
      instructions: "You evaluate whether AI responses are helpful. Always respond with valid JSON.",
    },
  })
    .generateScore({
      description: "Rate helpfulness from 0 to 1",
      createPrompt: ({ run }) =>
        `Given this question: "${run.input}"\nAnd this response: "${run.output}"\n\nRate the helpfulness from 0.0 to 1.0. Return ONLY a JSON object: {"score": <number>}`,
    })
    .generateReason({
      description: "Explain the rating",
      createPrompt: ({ run, score }) =>
        `The response "${run.output}" to "${run.input}" was rated ${score} for helpfulness. Explain why in one sentence.`,
    });

  const result = await helpfulnessScorer.run({
    input: "What is the capital of France?",
    output: "The capital of France is Paris, a major European city known for the Eiffel Tower.",
  });

  output({
    score: result.score,
    reason: result.reason,
    hasScore: typeof result.score === "number" && result.score >= 0 && result.score <= 1,
    hasReason: typeof result.reason === "string" && result.reason.length > 0,
  });
} catch(e) {
  output({
    error: e.message,
    stack: (e.stack || "").substring(0, 500),
  });
}
