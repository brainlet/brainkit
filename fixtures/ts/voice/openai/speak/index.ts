// Test: OpenAIVoice.speak — turn a short prompt into audio and assert
// the returned stream yields at least one non-empty byte chunk.
import { OpenAIVoice } from "agent";
import { output } from "kit";

const voice = new OpenAIVoice();
let totalBytes = 0;
let chunks = 0;
let errorMsg = "";
try {
  const stream: any = await voice.speak("brainkit", { responseFormat: "mp3" });
  for await (const chunk of stream) {
    const n = chunk?.byteLength ?? chunk?.length ?? 0;
    totalBytes += typeof n === "number" ? n : 0;
    chunks += 1;
    if (chunks > 200) break;
  }
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({
  hadChunks: chunks > 0,
  hadBytes: totalBytes > 0,
  errorIfAny: errorMsg,
});
