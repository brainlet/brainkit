// Test: MDocument creation
import { MDocument } from "agent";
import { output } from "kit";

try {
  const doc = MDocument.fromText("Go is a language.");
  output({
    created: doc !== null && doc !== undefined,
    type: typeof doc,
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}
