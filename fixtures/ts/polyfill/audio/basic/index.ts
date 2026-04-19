// Test: globalThis.Audio polyfill — construct, paused flag,
// and addEventListener surface. No configured sink means
// play() resolves silently (NullSink default).
import { output } from "kit";

const tinyMp3 = new Uint8Array([0x49, 0x44, 0x33, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10]);
const audio = new Audio(tinyMp3);

let endedFired = false;
audio.addEventListener("ended", () => { endedFired = true; });

const hasAudioClass = typeof Audio === "function";
const initiallyPaused = audio.paused === true;
const endedBeforePlay = audio.ended === false;

await audio.play();

output({
  hasAudioClass,
  initiallyPaused,
  endedBeforePlay,
  endedAfterPlay: audio.ended === true,
  endedEventFired: endedFired,
  pausedAfterPlay: audio.paused === true,
});
