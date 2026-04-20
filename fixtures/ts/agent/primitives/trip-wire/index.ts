// `TripWire` is the exception Mastra throws when a processor (or
// user code) aborts an in-flight run — e.g. a PIIDetector calls
// `abort()` mid-stream. User code that wants to differentiate
// "agent bailed intentionally" from "real error" catches TripWire.
import { TripWire } from "agent";
import { output } from "kit";

// Throw and catch to prove the class is instantiable + catchable.
let caught: TripWire | null = null;
let message = "";
try {
  throw new (TripWire as any)("simulated abort");
} catch (e: any) {
  if (e instanceof (TripWire as any)) {
    caught = e;
    message = String(e.message || "");
  }
}

// Non-TripWire errors must NOT match — prove the discrimination.
let plainCaught = false;
try {
  throw new Error("plain error");
} catch (e: any) {
  plainCaught = e instanceof (TripWire as any);
}

output({
  caughtAsTripWire: caught !== null,
  caughtMessage: message,
  plainErrorNotTripWire: !plainCaught,
  isErrorSubclass: caught instanceof Error,
});
