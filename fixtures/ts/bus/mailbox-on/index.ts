// Test: bus.on() — deployment mailbox (ts.<source>.<localTopic>)
import type { BusMessage } from "kit";
import { bus, output } from "kit";

let received: any = null;

// bus.on("ask") subscribes to ts.mailbox-on.ask (deployment namespace)
bus.on("ask", (msg: BusMessage) => {
  received = msg.payload;
  msg.reply({ answer: "42" });
});

const reply: any = await bus.call("ts.mailbox-on.ask", { question: "meaning of life" }, { timeoutMs: 5000 });

output({
  received: received !== null,
  question: (received as any)?.question || "",
  replied: reply !== null,
  answer: (reply as any)?.answer || "",
});
