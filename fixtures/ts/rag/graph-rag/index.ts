// Test: GraphRAG availability check
import { output } from "kit";

// GraphRAG is an endowment from kit_runtime.js
try {
  const hasGraphRAG = typeof GraphRAG === "function";
  output({
    available: hasGraphRAG,
  });
} catch (e: any) {
  output({ available: false, error: e.message.substring(0, 100) });
}
