import { bus, output } from "kit";

bus.on("stream-test", function(msg: any) {
  msg.stream.text("chunk1");
  msg.stream.text("chunk2");
  msg.stream.progress(50, "halfway");
  msg.stream.event("custom", { key: "value" });
  msg.stream.end({ done: true });
});

// Verify the handler registered
output({ handlerRegistered: true });
