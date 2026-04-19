// Test: UnicodeNormalizer construct + shape.
import { UnicodeNormalizer } from "agent";
import { output } from "kit";

const p = new UnicodeNormalizer({ stripControlChars: true });
output({
  id: p.id,
  hasProcessInput: typeof (p as any).processInput === "function",
  name: typeof p.name === "string",
});
