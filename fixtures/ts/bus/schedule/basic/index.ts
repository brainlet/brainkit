// Test: bus.schedule / bus.unschedule surface exists on the
// bus object. When the scheduler module is configured, a
// bus.schedule call returns a string id. When it isn't, the
// call throws a BrainkitError cleanly. Either branch is a
// pass — we're asserting the API shape + error envelope.
import { bus, output } from "kit";

const hasSchedule = typeof bus.schedule === "function";
const hasUnschedule = typeof bus.unschedule === "function";

let graceful = false;
try {
  const id = bus.schedule("in 1h", "ts.scheduled.noop", { tag: "basic" });
  graceful = typeof id === "string" && id.length > 0;
  if (graceful) bus.unschedule(id);
} catch (e: any) {
  graceful = e?.name === "BrainkitError" && typeof e?.code === "string" && e.code.length > 0;
}

output({ hasSchedule, hasUnschedule, graceful });
