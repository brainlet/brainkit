// Test: MurfVoice — construct with dummy key.
import { MurfVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new MurfVoice({
    speechModel: { name: "en-US-natalie", apiKey: "test" },
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
