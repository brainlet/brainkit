// Test: createTextualDifferenceScorer — code-based diff ratio.
import { createTextualDifferenceScorer } from "agent";
import { output } from "kit";

const msg = (role: "user" | "assistant", text: string) => ({
  id: `m-${role}-${Math.random().toString(36).slice(2)}`,
  role,
  createdAt: new Date(),
  content: { format: 2, parts: [{ type: "text", text }] },
});

const s = createTextualDifferenceScorer();
const result = await (s as any).run({
  input: {
    inputMessages: [msg("user", "The quick brown fox")],
    rememberedMessages: [],
    systemMessages: [],
    taggedSystemMessages: {},
  },
  output: [msg("assistant", "The quick red fox")],
});
output({
  id: (s as any).id,
  hasScore: typeof result.score === "number",
  scoreInRange: result.score >= 0 && result.score <= 1,
});
