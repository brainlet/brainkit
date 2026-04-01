import { bus, output } from "kit";

// Test bus.sendTo routing — sends to a service mailbox
// We don't have a receiver deployed, so we just verify the publish works
const result = bus.sendTo("my-service.ts", "ask", { question: "hello" });
output({ published: true, hasReplyTo: result.replyTo.length > 0 });
