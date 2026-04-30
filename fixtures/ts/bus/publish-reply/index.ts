// Test: bus.call() uses shared request/reply and subscriber uses msg.reply()
import type { BusMessage } from "kit";
import { bus, output } from "kit";

// Subscribe to a service topic
bus.subscribe("test.greet", (msg: BusMessage) => {
  msg.reply({ greeting: "hello " + (msg.payload as any).name });
});

const replyData = await bus.call("test.greet", { name: "brainkit" }, { timeoutMs: 5000 });

output({
  replied: replyData !== null,
  greeting: (replyData as any)?.greeting || "",
});
