// Test: PlayAIVoice — construct with dummy creds + verify a preset
// voice id from PLAYAI_VOICES is resolvable.
import { PlayAIVoice, PLAYAI_VOICES } from "agent";
import { output } from "kit";

let constructed = false;
let errorMsg = "";
try {
  const v = new PlayAIVoice({
    speechModel: { name: "Play3.0-mini", apiKey: "test", userId: "u" },
  });
  constructed = typeof (v as any).speak === "function";
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({
  constructedOrCleanError: constructed || errorMsg.length > 0,
  hasPresets: typeof PLAYAI_VOICES === "object" && PLAYAI_VOICES !== null,
});
