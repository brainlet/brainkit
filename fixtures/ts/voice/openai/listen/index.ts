// Test: OpenAIVoice full roundtrip — speak a phrase, pipe the audio
// back through listen, and assert the transcript isn't empty.
import { OpenAIVoice } from "agent";
import { output } from "kit";

const voice = new OpenAIVoice();
let transcript = "";
let errorMsg = "";
try {
  const stream = await voice.speak("brainkit is a Go runtime", { responseFormat: "mp3" });
  transcript = await voice.listen(stream, { filetype: "mp3" });
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({
  hasTranscript: typeof transcript === "string" && transcript.length > 0,
  errorMsg,
});
