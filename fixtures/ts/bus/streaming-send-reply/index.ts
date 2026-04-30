// Test: msg.send() for streaming chunks + msg.reply() for final.
// JS bus.callStream delivers intermediate chunks through the shared caller
// stream handler, then resolves with the terminal reply.
import type { BusMessage } from "kit";
import { bus, output } from "kit";

// Service sends 3 chunks then final reply
bus.subscribe("test.stream-svc", (msg: BusMessage) => {
  msg.send({ chunk: 1 });
  msg.send({ chunk: 2 });
  msg.send({ chunk: 3 });
  msg.reply({ done: true, total: 3 });
});

const chunks: any[] = [];
const final: any = await bus.callStream("test.stream-svc", { start: true }, {
  timeoutMs: 5000,
  onChunk: (chunk: any) => {
    chunks.push(chunk);
  },
});

output({
  hasFinal: final?.done === true,
  total: final?.total || 0,
  chunks: chunks.length,
});
