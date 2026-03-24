// Test: bus.subscribe() — receive messages from the platform bus
import type { BusMessage } from "kit";
import { bus, output } from "kit";

interface ReceivedEvent {
  topic: string;
  payload: { value: number };
}

const received: ReceivedEvent[] = [];

// Subscribe to a topic
const subId = bus.subscribe("test.events", (msg: BusMessage) => {
  received.push({
    topic: msg.topic,
    payload: msg.payload as { value: number },
  });
});

// Publish a message (goes through Go bus, comes back to our subscription)
bus.publish("test.events", { value: 42 });
bus.publish("test.events", { value: 99 });

// Give the Schedule callbacks time to fire
await new Promise(r => setTimeout(r, 100));

// Unsubscribe
bus.unsubscribe(subId);

// This one should NOT be received
bus.publish("test.events", { value: 999 });
await new Promise(r => setTimeout(r, 100));

output({
  subId: typeof subId === "string" && subId.length > 0,
  count: received.length,
  values: received.map(r => r.payload.value),
  topics: received.map(r => r.topic),
});
