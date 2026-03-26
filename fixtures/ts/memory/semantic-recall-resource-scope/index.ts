// Test: semantic recall configuration — validates that Memory accepts scope config
// Note: full semantic recall requires vector store + embedder; this tests config acceptance
import { Memory, InMemoryStore } from "agent";
import { output } from "kit";

try {
  const store = new InMemoryStore();
  
  // Without vector store, semantic recall should gracefully degrade or throw clear error
  let errorMsg = "";
  try {
    const mem = new Memory({
      storage: store,
      options: {
        lastMessages: 10,
        semanticRecall: { topK: 3, scope: "resource" },
      },
    });
    // If it doesn't throw, config was accepted
    output({ configAccepted: true, needsVector: false });
  } catch (e: any) {
    errorMsg = e.message;
    // Expected: "Semantic recall requires a vector store"
    output({
      configAccepted: false,
      needsVector: errorMsg.includes("vector store"),
      error: errorMsg.substring(0, 100),
    });
  }
} catch (e: any) {
  output({ error: e.message.substring(0, 200) });
}
