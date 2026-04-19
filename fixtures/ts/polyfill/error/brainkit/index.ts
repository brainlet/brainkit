// Test: BrainkitError class with code + details — instanceof
// narrows on catch.
import { output } from "kit";

let caught = false;
let caughtCode = "";
let caughtDetails: any = null;
let isInstance = false;

try {
  throw new BrainkitError("something broke", "CUSTOM_CODE", { traceId: "abc" });
} catch (e) {
  caught = true;
  isInstance = e instanceof BrainkitError;
  if (isInstance) {
    caughtCode = (e as any).code;
    caughtDetails = (e as any).details;
  }
}

output({
  hasClass: typeof BrainkitError === "function",
  caught,
  isInstance,
  caughtCode,
  hasDetails: caughtDetails !== null && caughtDetails.traceId === "abc",
});
