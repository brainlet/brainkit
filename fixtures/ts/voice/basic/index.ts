// Test: OpenAIVoice base shape — construct and assert speak/listen
// are callable. Actual audio round-trip lives in openai/speak.
import { OpenAIVoice } from "agent";
import { output } from "kit";

const voice = new OpenAIVoice();
output({
  constructed: voice !== null && voice !== undefined,
  hasSpeak: typeof (voice as any).speak === "function",
  hasListen: typeof (voice as any).listen === "function",
});
