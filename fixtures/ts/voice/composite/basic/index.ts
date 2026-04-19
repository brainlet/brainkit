// Test: CompositeVoice — route speak + listen through separate
// provider instances.
import { CompositeVoice, OpenAIVoice } from "agent";
import { output } from "kit";

const speakProvider = new OpenAIVoice();
const listenProvider = new OpenAIVoice();

const composite = new CompositeVoice({ speakProvider, listenProvider });
output({
  constructed: composite !== null && composite !== undefined,
  hasSpeak: typeof (composite as any).speak === "function",
  hasListen: typeof (composite as any).listen === "function",
});
