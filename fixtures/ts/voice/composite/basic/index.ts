// Test: CompositeVoice — route speak + listen through separate
// provider instances.
import { CompositeVoice, OpenAIVoice } from "agent";
import { output } from "kit";

const outputVoice = new OpenAIVoice();
const inputVoice = new OpenAIVoice();

const composite = new CompositeVoice({ output: outputVoice, input: inputVoice });
output({
  constructed: composite !== null && composite !== undefined,
  hasSpeak: typeof (composite as any).speak === "function",
  hasListen: typeof (composite as any).listen === "function",
});
