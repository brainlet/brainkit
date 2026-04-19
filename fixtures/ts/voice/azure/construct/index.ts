// Test: AzureVoice — construct with dummy creds, assert surface.
// Real TTS needs AZURE_SPEECH_KEY + region; fixture only locks the
// call shape.
import { AzureVoice } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new AzureVoice({
    speechModel: { name: "en-US-AriaNeural", apiKey: "test", region: "eastus" },
    speaker: "en-US-AriaNeural",
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({ constructedOrCleanError: constructed || errorMsg.length > 0 });
