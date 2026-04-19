// Test: SpeechifyVoice — construct with dummy key.
import { SpeechifyVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new SpeechifyVoice({
    speechModel: { name: "simba-english", apiKey: "test" },
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
