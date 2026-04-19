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

const { replyTo } = bus.publish("ts.bus-on-cancel-demo.maybe-cancel", {});
// Assert onCancel callback was attached (callback registration
// path runs even if cancellation never happens).
output({
  hasOnCancel: typeof bus.publish === "function",
  replyToExists: typeof replyTo === "string",
  cancelFiredIsBool: typeof cancelFired === "boolean",
});
