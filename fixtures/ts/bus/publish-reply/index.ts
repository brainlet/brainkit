// Test: bus.publish() returns replyTo, subscriber uses msg.reply()
import type { BusMessage } from "kit";
import { bus, output } from "kit";

let replyData: any = null;

// Subscribe to a service topic
bus.subscribe("test.greet", (msg: BusMessage) => {
  msg.reply({ greeting: "hello " + (msg.payload as any).name });
});

// Publish and subscribe to the replyTo topic
const result = bus.publish("test.greet", { name: "brainkit" });
bus.subscribe(result.replyTo, (msg: BusMessage) => {
  replyData = msg.payload;
});

await new Promise(r => setTimeout(r, 200));

output({
  hasReplyTo: typeof result.replyTo === "string" && result.replyTo.length > 0,
  hasCorrelationId: typeof result.correlationId === "string" && result.correlationId.length > 0,
  replied: replyData !== null,
  greeting: (replyData as any)?.greeting || "",
});
