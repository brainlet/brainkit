// Test: MDocument from different content types
import { MDocument } from "agent";
import { output } from "kit";

try {
  // From plain text
  const textDoc = MDocument.fromText("Hello world. This is a test.");
  
  // From HTML (if supported)
  let htmlWorks = false;
  try {
    const htmlDoc = MDocument.fromHTML("<p>Hello</p><p>World</p>");
    htmlWorks = htmlDoc !== null;
  } catch { htmlWorks = false; }

  output({
    textCreated: textDoc !== null && textDoc !== undefined,
    htmlWorks,
  });
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}
