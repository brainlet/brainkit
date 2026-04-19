// Test: Agent with an OpenAIVoice attached. Verify the voice
// provider is accessible from agent.voice + agent.getVoice().
import { Agent, OpenAIVoice } from "agent";
import { model, output } from "kit";

const agent = new Agent({
  name: "voice-basic",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Reply briefly.",
  voice: new OpenAIVoice(),
});

const resolved = await agent.getVoice();

output({
  attached: (agent as any).voice !== undefined && (agent as any).voice !== null,
  canSpeak: typeof ((agent as any).voice as any)?.speak === "function",
  resolvesViaGetter: resolved !== undefined && resolved !== null,
});
