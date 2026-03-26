// Test: bus.sendTo() — address a deployed .ts service by name
import { bus, output } from "kit";

// Deploy a handler on this service's mailbox
bus.on("greet", (msg: any) => {
  msg.reply({ greeting: "hello " + (msg.payload?.name || "world") });
});

// Use bus.sendTo to reach this service (self-addressing for test)
// Resolves: "send-to-service.ts" + "greet" → "ts.send-to-service.greet"
const result = bus.sendTo("send-to-service.ts", "greet", { name: "brainkit" });

let reply: any = null;
bus.subscribe(result.replyTo, (msg: any) => {
  reply = msg.payload;
});

await new Promise(r => setTimeout(r, 300));

output({
  hasReplyTo: result.replyTo.length > 0,
  gotReply: reply !== null,
  greeting: reply?.greeting || "",
});
