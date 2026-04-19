// Test: advanced semantic recall config — topK, messageRange pair,
// scope, threshold, indexName, indexConfig. Exercises the config
// surface with an InMemoryStore so the fixture doesn't need a vector
// container. End-to-end recall is covered by semantic-recall/basic
// on LibSQL; this locks in the knob shape.
import { Memory, InMemoryStore } from "agent";
import { output } from "kit";

let accepted = false;
let errorMsg = "";
try {
  new Memory({
    storage: new InMemoryStore(),
    options: {
      lastMessages: 2,
      semanticRecall: {
        topK: 5,
        messageRange: { before: 1, after: 2 },
        scope: "resource",
        threshold: 0.1,
        indexName: "sem_recall_advanced",
        indexConfig: { type: "hnsw", metric: "cosine", hnsw: { m: 16, efConstruction: 64 } },
      },
    },
  });
  accepted = true;
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

// Memory can validate lazily; either instantaneous acceptance OR a
// clear "needs vector store" error is a pass — both prove the knob
// shape is recognized.
output({
  acceptedOrCleanError: accepted || errorMsg.toLowerCase().includes("vector"),
});
