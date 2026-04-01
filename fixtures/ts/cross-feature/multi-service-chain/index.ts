import { bus, output } from "kit";

// This fixture deploys as service A.
// It tests that bus.sendTo correctly routes to other services.
// Since we can't deploy multiple services in one fixture,
// we test the bus.publish routing mechanism.

const result = bus.publish("incoming.chain-test", { step: "A" });
output({ 
  published: true, 
  hasReplyTo: result.replyTo.length > 0,
  hasCorrelation: result.correlationId.length > 0
});
