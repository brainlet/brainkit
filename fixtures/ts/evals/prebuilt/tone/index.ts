// Test: createToneScorer — code-based sentiment consistency.
import { createToneScorer } from "agent";
import { output } from "kit";

const msg = (role: "user" | "assistant", text: string) => ({
  id: `m-${role}-${Math.random().toString(36).slice(2)}`,
  role,
  createdAt: new Date(),
  content: { format: 2, parts: [{ type: "text", text }] },
});

const s = createToneScorer();
const result = await (s as any).run({
  input: {
    inputMessages: [msg("user", "Tell me something cheerful")],
    rememberedMessages: [],
    systemMessages: [],
    taggedSystemMessages: {},
  },
  output: [msg("assistant", "What a wonderful day! The sun is shining and everything feels great.")],
});
output({
  id: (s as any).id,
  hasScore: typeof result.score === "number",
  scoreInRange: result.score >= 0 && result.score <= 1,
});
