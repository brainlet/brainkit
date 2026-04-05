// Test: agent.generate with per-call scorers
import { Agent, createScorer } from "agent";
import { model, output } from "kit";

const lengthScorer = createScorer({
  id: "length",
  name: "Length",
  description: "Scores by text length",
}).generateScore(({ run }: any) => {
  return Math.min((run.output?.text || "").length / 50, 1);
});

const agent = new Agent({
  name: "scored-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Answer concisely.",
});

try {
  const result = await agent.generate("What is Go?", {
    scorers: { length: { scorer: lengthScorer } },
    returnScorerData: true,
  });

  output({
    hasText: result.text.length > 0,
    hasScoringData: (result as any).scoringData !== undefined,
  });
} catch (e: any) {
  // scorers might not be fully supported in our bundle
  output({ hasText: false, error: e.message.substring(0, 200) });
}
