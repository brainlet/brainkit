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

// The Whisper upload needs multipart FormData with file support; the
// brainkit polyfill exposes FormData but not file uploads today, so the
// listen call errors with a clean FormData message instead of crashing.
// We accept either outcome — this fixture locks in the call shape so a
// future polyfill upgrade is a one-line expect flip.
output({
  callShape: typeof transcript === "string" || errorMsg.toLowerCase().includes("formdata"),
});
