import { bus, output } from "kit";

// Rapidly publish 50 messages — verify no crash
let count = 0;
for (let i = 0; i < 50; i++) {
  try {
    bus.publish("incoming.concurrent-test-" + i, { index: i });
    count++;
  } catch (e) {
    // Rate limit or other error — count what succeeded
  }
}
output({ published: count, allDone: count > 0 });
