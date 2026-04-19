// Test: OpenAIRealtimeVoice — construct + check connect/disconnect
// surface. We don't actually open a WebSocket in the fixture to keep
// it deterministic; the round-trip lives in a Go suite test that
// spins up a mock Realtime endpoint.
import { OpenAIRealtimeVoice } from "agent";
import { output } from "kit";

const voice = new OpenAIRealtimeVoice();
output({
  constructed: voice !== null && voice !== undefined,
  hasConnect: typeof (voice as any).connect === "function",
  hasDisconnect: typeof (voice as any).disconnect === "function",
  hasSend: typeof (voice as any).send === "function",
  hasOn: typeof (voice as any).on === "function",
});
