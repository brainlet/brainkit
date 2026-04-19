// Test: msg.send() for streaming chunks + msg.reply() for final
import type { BusMessage } from "kit";
import { bus, output } from "kit";

const chunks: any[] = [];

// Service sends 3 chunks then final reply
bus.subscribe("test.stream-svc", (msg: BusMessage) => {
  msg.send({ chunk: 1 });
  msg.send({ chunk: 2 });
  msg.send({ chunk: 3 });
  msg.reply({ done: true, total: 3 });
});

const result = bus.publish("test.stream-svc", { start: true });
bus.subscribe(result.replyTo, (msg: BusMessage) => {
  // msg.send chunks arrive raw; msg.reply arrives wrapped as
  // {ok:true, data:...} envelope. Unwrap the terminal reply so the
  // done flag surfaces.
  const p: any = msg.payload;
  chunks.push(
    p && typeof p === "object" && "ok" in p && "data" in p ? p.data : p,
  );
});

await new Promise(r => setTimeout(r, 300));

output({
  chunkCount: chunks.length,
  hasFinal: chunks.some((c: any) => c.done === true),
});
