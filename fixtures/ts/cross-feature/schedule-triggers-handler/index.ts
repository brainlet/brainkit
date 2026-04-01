import { bus, output } from "kit";

// Schedule a message — verify the schedule API works from .ts
const schedId = bus.schedule("in 1h", "ts.schedule-triggers-handler.fire", { scheduled: true });

// Verify we got a schedule ID back
const hasId = schedId.length > 0;

// Unschedule it to clean up
bus.unschedule(schedId);

output({ scheduled: true, scheduleId: hasId, unscheduled: true });
