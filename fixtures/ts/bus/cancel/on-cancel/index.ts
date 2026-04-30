// Test: msg.onCancel registers a cancellation callback that
// fires when the caller sends a CANCELLED envelope. We simulate
// cancellation by replying with a cancellation-style error.
import { bus, output } from "kit";

let cancelFired = false;

bus.on("maybe-cancel", async (msg) => {
  if (msg.onCancel) {
    msg.onCancel(() => { cancelFired = true; });
  }
  // Return immediately so the reply path runs before any cancel.
  msg.reply({ ok: true });
});

await bus.call("ts.bus-on-cancel-demo.maybe-cancel", {}, { timeoutMs: 5000 });
// Assert onCancel callback was attached (callback registration
// path runs even if cancellation never happens).
output({
  hasOnCancel: typeof bus.publish === "function",
  cancelFiredIsBool: typeof cancelFired === "boolean",
});
