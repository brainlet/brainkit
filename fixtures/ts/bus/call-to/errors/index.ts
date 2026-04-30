// Test: bus.sendTo to a non-existent service is fire-and-forget.
import { bus, output } from "kit";

let threw = false;
let errorName = "";
let errorCode = "";

try {
  // Call a service that isn't deployed.
  bus.sendTo("ghost-service", "some-topic", { probe: true });
} catch (e: any) {
  threw = true;
  errorName = e?.name || "";
  errorCode = e?.code || "";
}

output({
  sendToDidNotThrow: !threw,
  hasSendTo: typeof bus.sendTo === "function",
  errorNameShape: errorName === "" || errorName === "BrainkitError",
  errorCodeIsString: typeof errorCode === "string",
});
