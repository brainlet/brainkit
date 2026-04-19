// Test: Audio polyfill accepts a Node Readable source and
// drains it before playing.
import { output } from "kit";

// Build a tiny Node-shaped Readable over a few MP3-shaped
// bytes. `stream` is the module polyfill on globalThis.
const src = new stream.Readable({ read() {} });
src.push(new Uint8Array([0x49, 0x44, 0x33]));
src.push(new Uint8Array([0x04, 0x00, 0x00, 0x00]));
src.push(new Uint8Array([0x00, 0x00, 0x10]));
src.push(null);

const audio = new Audio(src);
await audio.play();

output({
  endedAfterPlay: audio.ended === true,
  pausedAfterPlay: audio.paused === true,
});
