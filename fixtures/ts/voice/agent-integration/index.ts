// Test: Agent.voice wiring — Agent accepts a voice provider and
// exposes it via `agent.voice`. Surface-only so we don't need a
// full TTS round-trip to validate the wiring.
import { Agent, OpenAIVoice } from "agent";
import { model, output } from "kit";

const voice = new OpenAIVoice();
const agent = new Agent({
  name: "voice-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions: "You are a helpful voice assistant.",
  voice,
});

output({
  hasVoice: (agent as any).voice !== undefined && (agent as any).voice !== null,
  canSpeak: typeof ((agent as any).voice as any)?.speak === "function",
  canListen: typeof ((agent as any).voice as any)?.listen === "function",
});
