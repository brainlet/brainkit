// Test: scorer with LLM judge — generateScore uses a prompt + model
import { createScorer } from "agent";
import { model, output } from "kit";

try {
  const scorer = createScorer({
    id: "llm-judge",
    name: "LLM Judge",
    description: "Uses an LLM to score output quality",
  }).generateScore({
    description: "Score the output quality from 0 to 1",
    judge: {
      model: model("openai", "gpt-4o-mini"),
      instructions: "You are a quality judge. Return ONLY a number between 0 and 1.",
    },
    createPrompt: ({ run }: any) => {
      return `Rate the quality of this response on a scale of 0 to 1:\nInput: ${run.input?.[0]?.content}\nOutput: ${run.output?.text}`;
    },
  });

  const result = await scorer.run({
    input: [{ role: "user", content: "What is Go?" }],
    output: { role: "assistant", text: "Go is a statically typed, compiled programming language designed at Google by Robert Griesemer, Rob Pike, and Ken Thompson." },
  });

  output({
    hasScore: typeof result.score === "number",
    scoreInRange: result.score >= 0 && result.score <= 1,
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}
