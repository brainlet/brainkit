// Test: createCompletenessScorer — code-based, no model. Agent-shape run
// input uses MastraDBMessage: { content: { format: 2, parts: [...] } }.
import { createCompletenessScorer } from "agent";
import { output } from "kit";

const msg = (role: "user" | "assistant", text: string) => ({
  id: `m-${role}-${Math.random().toString(36).slice(2)}`,
  role,
  createdAt: new Date(),
  content: { format: 2, parts: [{ type: "text", text }] },
});

const s = createCompletenessScorer();
const result = await (s as any).run({
  input: {
    inputMessages: [msg("user", "Name three primary colors")],
    rememberedMessages: [],
    systemMessages: [],
    taggedSystemMessages: {},
  },
  output: [msg("assistant", "red, green, blue")],
});
output({
  id: (s as any).id,
  hasScore: typeof result.score === "number",
  scoreInRange: result.score >= 0 && result.score <= 1,
});
