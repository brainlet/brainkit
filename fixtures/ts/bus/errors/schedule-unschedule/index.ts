import { bus, output } from "kit";

const ids: string[] = [];
for (let i = 0; i < 5; i++) {
  const id = bus.schedule("in 1h", `sched-test-${i}`, { index: i });
  ids.push(id);
}

// Unschedule all
for (const id of ids) {
  bus.unschedule(id);
}

output({ scheduled: ids.length, allHaveIds: ids.every(id => id.length > 0) });
