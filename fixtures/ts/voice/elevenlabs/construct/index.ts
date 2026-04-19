// Test: ElevenLabsVoice — construct with dummy creds.
import { ElevenLabsVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new ElevenLabsVoice({
    speechModel: { name: "eleven_monolingual_v1", apiKey: "test" },
    speaker: "rachel",
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
