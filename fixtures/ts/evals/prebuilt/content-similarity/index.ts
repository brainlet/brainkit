// Test: createContentSimilarityScorer — code-based string similarity.
import { createContentSimilarityScorer } from "agent";
import { output } from "kit";

const msg = (role: "user" | "assistant", text: string) => ({
  id: `m-${role}-${Math.random().toString(36).slice(2)}`,
  role,
  createdAt: new Date(),
  content: { format: 2, parts: [{ type: "text", text }] },
});

const s = createContentSimilarityScorer({ ignoreCase: true, ignoreWhitespace: true });
const result = await (s as any).run({
  input: {
    inputMessages: [msg("user", "Hello world")],
    rememberedMessages: [],
    systemMessages: [],
    taggedSystemMessages: {},
  },
  output: [msg("assistant", "hello   WORLD")],
});
output({
  id: (s as any).id,
  hasScore: typeof result.score === "number",
  scoreInRange: result.score >= 0 && result.score <= 1,
});
