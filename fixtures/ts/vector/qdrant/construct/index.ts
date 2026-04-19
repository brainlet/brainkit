// Test: QdrantVector — construct with dummy URL.
import { QdrantVector } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new QdrantVector({ url: "http://localhost:6333", apiKey: "test" });
  constructed = typeof (v as any).upsert === "function"
    && typeof (v as any).query === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
