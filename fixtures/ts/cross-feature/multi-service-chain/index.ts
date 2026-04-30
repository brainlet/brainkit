import { bus, output } from "kit";

// This fixture deploys as service A.
// It tests the bus.publish fire-and-forget routing mechanism.

const result = bus.publish("incoming.chain-test", { step: "A" });
output({ 
  published: result === undefined,
});
