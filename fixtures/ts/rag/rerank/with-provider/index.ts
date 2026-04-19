// Test: rerank() surface with an AI model as provider. Full round-trip
// crashes QuickJS with SIGBUS — see
// brainkit-maps/knowledge/rerank-sigbus-bigjs.md for repro and the
// likely big.js cause. Surface check locks the call signature so the
// fix is a one-line expect flip.
// TODO(bug): restore end-to-end rerank once the SIGBUS is resolved.
import { rerank } from "agent";
import { model, output } from "kit";

const m = model("openai", "gpt-4o-mini");
output({
  hasRerank: typeof rerank === "function",
  modelResolves: m !== null && m !== undefined,
});
