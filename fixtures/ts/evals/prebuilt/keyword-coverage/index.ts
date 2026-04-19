// Test: createKeywordCoverageScorer — code-based keyword overlap.
import { createKeywordCoverageScorer } from "agent";
import { output } from "kit";

const msg = (role: "user" | "assistant", text: string) => ({
  id: `m-${role}-${Math.random().toString(36).slice(2)}`,
  role,
  createdAt: new Date(),
  content: { format: 2, parts: [{ type: "text", text }] },
});

const s = createKeywordCoverageScorer();
const result = await (s as any).run({
  input: {
    inputMessages: [msg("user", "Explain compiler and linker")],
    rememberedMessages: [],
    systemMessages: [],
    taggedSystemMessages: {},
  },
  output: [msg("assistant", "A compiler transforms source; a linker resolves symbols.")],
});
output({
  id: (s as any).id,
  hasScore: typeof result.score === "number",
  scoreInRange: result.score >= 0 && result.score <= 1,
});
