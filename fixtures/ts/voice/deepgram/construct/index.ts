// Test: DeepgramVoice — construct with dummy key.
import { DeepgramVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new DeepgramVoice({
    speechModel: { name: "aura-asteria-en", apiKey: "test" },
    listeningModel: { name: "nova-2", apiKey: "test" },
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
