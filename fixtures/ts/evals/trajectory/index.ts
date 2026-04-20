// `createTrajectoryAccuracyScorerCode` validates that an agent run
// followed an expected step sequence. Pure-code: no LLM judge.
// Useful for regression-testing multi-step tool pipelines (router
// tool → search tool → synthesis) where step order is the contract.
import { createTrajectoryAccuracyScorerCode } from "agent";
import { output } from "kit";

const scorer = (createTrajectoryAccuracyScorerCode as any)({
  expectedTrajectory: {
    steps: [
      { stepType: "tool_call", name: "lookup" },
      { stepType: "tool_call", name: "format" },
    ],
  },
});

output({
  hasScorer: typeof scorer === "object" && scorer !== null,
  hasRun: typeof (scorer as any).run === "function",
  scorerName:
    (scorer as any).name ||
    (scorer as any).config?.name ||
    "",
});
