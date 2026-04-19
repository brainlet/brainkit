// Test: Agent with CompositeVoice — different providers for speak
// and listen. Verifies the agent accepts + exposes the composite.
import { Agent, CompositeVoice, OpenAIVoice } from "agent";
import { model, output } from "kit";

const voice = new CompositeVoice({
  speakProvider: new OpenAIVoice(),
  listenProvider: new OpenAIVoice(),
});

const agent = new Agent({
  name: "voice-composite",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
  voice,
});

output({
  attached: (agent as any).voice !== undefined,
  canSpeak: typeof ((agent as any).voice as any)?.speak === "function",
  canListen: typeof ((agent as any).voice as any)?.listen === "function",
});
