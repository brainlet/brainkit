// Test: stream.Readable.from, pipe, data events
import { output } from "kit";

// Readable.from — creates a Readable from an iterable
const r = stream.Readable.from(["chunk1", "chunk2", "chunk3"]);

// Read buffered data
const chunks: string[] = [];
let c: any;
while ((c = r.read()) !== null) {
  chunks.push(String(c));
}

// Pipe test
const r2 = new stream.Readable();
const collected: string[] = [];
const w = new stream.Writable({
  write(chunk: any, _enc: string, cb: () => void) {
    collected.push(typeof chunk === "string" ? chunk : String(chunk));
    cb();
  }
});
r2.pipe(w);
r2.push("pipe-a");
r2.push("pipe-b");
r2.push(null);

await new Promise(r => setTimeout(r, 50));

output({
  fromChunks: chunks,
  fromCount: chunks.length,
  pipeCollected: collected,
  pipeCount: collected.length,
});
