// Test: MDocument token chunking — splits by token count (uses js-tiktoken)
import { MDocument, output } from "brainlet";

const sentences = Array.from({ length: 20 }, (_, i) =>
  `This is sentence number ${i + 1} in our test document about artificial intelligence and machine learning.`
);
const text = sentences.join(" ");

const doc = MDocument.fromText(text);

try {
  const chunks = await doc.chunk({
    strategy: "token",
    maxSize: 50,
    overlap: 10,
  });

  output({
    chunkCount: chunks.length,
    hasMultiple: chunks.length > 1,
    allHaveText: chunks.every(c => c.text && c.text.length > 0),
    totalTextLength: text.length,
  });
} catch(e) {
  output({
    error: e.message,
    stack: (e.stack || "").substring(0, 500),
  });
}
