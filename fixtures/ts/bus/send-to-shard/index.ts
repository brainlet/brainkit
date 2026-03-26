// Test: bus.sendToShard() — address a WASM shard's handler topic
import { bus, output } from "kit";

// WASM shards subscribe to arbitrary topics (not namespaced).
// For this test, we simulate by subscribing to a shard-like topic.
bus.subscribe("counter.increment", (msg: any) => {
  const amount = msg.payload?.amount || 1;
  msg.reply({ incremented: true, newValue: 10 + amount });
});

// Use bus.sendToShard to reach the handler
const result = bus.sendToShard("counter", "counter.increment", { amount: 5 });

let reply: any = null;
bus.subscribe(result.replyTo, (msg: any) => {
  reply = msg.payload;
});

await new Promise(r => setTimeout(r, 300));

output({
  hasReplyTo: result.replyTo.length > 0,
  gotReply: reply !== null,
  newValue: reply?.newValue || 0,
  incremented: reply?.incremented || false,
});
