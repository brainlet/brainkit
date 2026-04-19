// Test: MDocument.fromCSV — parses a CSV string (papaparse under the
// hood), returns an MDocument whose text concatenates rows with
// pipe separators. Then chunks by sentence for downstream embedding.
import { MDocument } from "agent";
import { output } from "kit";

const csv = [
  "name,role,city",
  "Alice,Engineer,Berlin",
  "Bob,Designer,Tokyo",
  "Carol,PM,Austin",
].join("\n");

const doc = (MDocument as any).fromCSV(csv);
const chunks = await doc.chunk({ strategy: "recursive", maxSize: 200, overlap: 0 });

const texts: string[] = Array.isArray(doc.getText()) ? doc.getText() : [doc.getText()];

output({
  chunkCount: chunks.length,
  allHaveText: chunks.every((c: any) => typeof c.text === "string" && c.text.length > 0),
  containsAlice: texts.join(" ").toLowerCase().includes("alice"),
  containsCity: texts.join(" ").toLowerCase().includes("berlin"),
});
