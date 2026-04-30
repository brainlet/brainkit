import { bus, output } from "kit";
const big = "x".repeat(50000);
const result = bus.publish("incoming.big-payload", { data: big });
output({ published: result === undefined, payloadSize: big.length });
