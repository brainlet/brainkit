// Test: bus.callService() — request/reply with a deployed .ts service by name
import { bus, output } from "kit";

// Deploy a handler on this service's mailbox
bus.on("greet", (msg: any) => {
  msg.reply({ greeting: "hello " + (msg.payload?.name || "world") });
});

// Use bus.callService to reach this service (self-addressing for test)
// Resolves: "send-to-service.ts" + "greet" → "ts.send-to-service.greet"
const reply: any = await bus.callService("send-to-service.ts", "greet", { name: "brainkit" }, { timeoutMs: 5000 });

output({
  gotReply: reply !== null,
  greeting: reply?.greeting || "",
});
