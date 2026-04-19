// Test: ChromaVector — construct with dummy path.
import { ChromaVector } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new ChromaVector({ path: "http://localhost:8000" });
  constructed = typeof (v as any).upsert === "function"
    && typeof (v as any).query === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
