// Test: bus.schedule accepts cron expressions. Graceful either
// way — configured scheduler returns id, unconfigured throws
// BrainkitError.
import { bus, output } from "kit";

let graceful = false;
try {
  const id = bus.schedule("0 0 * * *", "ts.scheduled.cron", { tag: "cron" });
  graceful = typeof id === "string" && id.length > 0;
  if (graceful) bus.unschedule(id);
} catch (e: any) {
  graceful = e?.name === "BrainkitError" && typeof e?.code === "string";
}

output({ graceful });
