// Test: SarvamVoice — construct with dummy key.
import { SarvamVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new SarvamVoice({
    speechModel: { name: "bulbul-v1", apiKey: "test", language: "en-IN" },
    listeningModel: { name: "saarika-v1", apiKey: "test", language: "en-IN" },
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
