// Test: OpenAIRealtimeVoice event wiring — register a "speaker" listener
// and confirm the on()/off() API accepts + removes it. Full round-trip
// requires a mock Realtime endpoint (Go suite test).
import { OpenAIRealtimeVoice } from "agent";
import { output } from "kit";

const voice = new OpenAIRealtimeVoice();
const listener = (_stream: any) => { /* noop */ };

let registered = false;
let removed = false;
try {
  (voice as any).on("speaker", listener);
  registered = true;
  (voice as any).off("speaker", listener);
  removed = true;
} catch (e: any) {
  // Accept + report but don't fail — different providers may be strict.
}

output({
  registered,
  removed,
});
