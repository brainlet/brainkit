// Test: bus.emit() — fire and forget, no replyTo
import type { BusMessage } from "kit";
import { bus, output } from "kit";

const received: any[] = [];

bus.subscribe("test.events.fire", (msg: BusMessage) => {
  received.push(msg.payload);
});

bus.emit("test.events.fire", { kind: "ping" });
bus.emit("test.events.fire", { kind: "pong" });

await new Promise(r => setTimeout(r, 200));

output({
  count: received.length,
  kinds: received.map((r: any) => r.kind),
});
