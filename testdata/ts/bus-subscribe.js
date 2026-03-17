// Test: bus.subscribe() — receive messages from the platform bus
import { bus, output } from "kit";

const received = [];

// Subscribe to a topic
const subId = bus.subscribe("test.events", (msg) => {
  received.push({
    topic: msg.topic,
    payload: msg.payload,
  });
});

// Publish a message (goes through Go bus, comes back to our subscription)
await bus.publish("test.events", { value: 42 });
await bus.publish("test.events", { value: 99 });

// Give the Schedule callbacks time to fire
await new Promise(r => setTimeout(r, 100));

// Unsubscribe
bus.unsubscribe(subId);

// This one should NOT be received
await bus.publish("test.events", { value: 999 });
await new Promise(r => setTimeout(r, 100));

output({
  subId: typeof subId === "string" && subId.length > 0,
  count: received.length,
  values: received.map(r => r.payload?.value),
  topics: received.map(r => r.topic),
});
