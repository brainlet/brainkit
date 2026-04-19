// Test: bus.on() — deployment mailbox (ts.<source>.<localTopic>)
import type { BusMessage } from "kit";
import { bus, output } from "kit";

let received: any = null;

// bus.on("ask") subscribes to ts.mailbox-on.ask (deployment namespace)
bus.on("ask", (msg: BusMessage) => {
  received = msg.payload;
  msg.reply({ answer: "42" });
});

// Publish to the deployment namespace directly
const result = bus.publish("ts.mailbox-on.ask", { question: "meaning of life" });

// Subscribe to reply — unwrap the {ok, data} envelope produced by msg.reply.
let reply: any = null;
bus.subscribe(result.replyTo, (msg: BusMessage) => {
  const p: any = msg.payload;
  reply = p && typeof p === "object" && "ok" in p && "data" in p ? p.data : p;
});

await new Promise(r => setTimeout(r, 200));

output({
  received: received !== null,
  question: (received as any)?.question || "",
  replied: reply !== null,
  answer: (reply as any)?.answer || "",
});
