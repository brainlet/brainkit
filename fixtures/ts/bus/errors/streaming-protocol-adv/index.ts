import { bus, output } from "kit";

// Exercise every stream method — proves wire format works
bus.on("stream-test", function(msg: any) {
  msg.stream.text("chunk1");
  msg.stream.text("chunk2");
  msg.stream.progress(50, "halfway");
  msg.stream.object({ partial: true });
  msg.stream.event("custom", { key: "value" });
  msg.stream.end({ done: true });
});

output({ handlerRegistered: true });
