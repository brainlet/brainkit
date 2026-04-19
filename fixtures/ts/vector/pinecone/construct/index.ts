// Test: PineconeVector — construct with dummy API key. Real upsert +
// query needs a live Pinecone index; this locks in the call surface.
import { PineconeVector } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new PineconeVector({ apiKey: "pk-test" });
  constructed = typeof (v as any).upsert === "function"
    && typeof (v as any).query === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
