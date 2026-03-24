// Test: MDocument markdown chunking — splits by headers
import { MDocument } from "agent";
import { output } from "kit";

const markdown = `# Main Title

Some introduction text here.

## Section A

Content for section A with details.

## Section B

Content for section B with more details.

### Subsection B1

Nested content under B.
`;

const doc = MDocument.fromMarkdown(markdown);
const chunks = await doc.chunk({
  strategy: "markdown",
  headers: [
    ["#", "h1"],
    ["##", "h2"],
    ["###", "h3"],
  ],
});

output({
  chunkCount: chunks.length,
  hasMultiple: chunks.length > 1,
  texts: chunks.map(c => c.text?.substring(0, 40)),
  metadata: chunks.map(c => c.metadata),
});
