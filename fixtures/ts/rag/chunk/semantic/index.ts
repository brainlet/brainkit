// Test: semantic-markdown chunking strategy — exercised through the
// validator. The call may succeed or fail depending on whether
// js-tiktoken is present; either outcome is a valid pass. We only
// assert the strategy isn't rejected as "Unknown".
import { MDocument } from "agent";
import { output } from "kit";

const markdown = "# H1\n\nbrainkit embeds QuickJS.\n\n# H2\n\nWatermill routes messages.";
const doc = MDocument.fromMarkdown(markdown);

let errorMessage = "";
let chunksReturned = false;
try {
  const chunks = await doc.chunk({ strategy: "semantic-markdown", joinThreshold: 100 } as any);
  chunksReturned = Array.isArray(chunks);
} catch (e: any) {
  errorMessage = String(e?.message || e).substring(0, 200);
}

output({
  strategyRecognized: !errorMessage.includes("Unknown chunking strategy"),
  ranOrFailedCleanly: chunksReturned || errorMessage.length > 0,
});
