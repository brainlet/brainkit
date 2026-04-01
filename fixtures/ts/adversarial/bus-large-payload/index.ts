import { bus, output } from "kit";
const big = "x".repeat(50000);
const result = bus.publish("incoming.big-payload", { data: big });
output({ published: true, hasReplyTo: result.replyTo.length > 0, payloadSize: big.length });
