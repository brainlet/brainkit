// Test: bus.withCancelController returns { signal, cleanup } —
// signal can be passed to fetch / any AbortSignal consumer and
// cleanup releases the bus-side watcher.
import { bus, output } from "kit";

let handlerGotCtrl = false;
let signalWasAborted = false;

bus.on("abort-demo", async (msg) => {
  const ctrl = bus.withCancelController(msg);
  handlerGotCtrl = typeof ctrl === "object" && ctrl !== null &&
    typeof ctrl.signal === "object" && typeof ctrl.cleanup === "function";
  if (handlerGotCtrl) {
    signalWasAborted = ctrl.signal.aborted === false;
    ctrl.cleanup();
  }
  msg.reply({ ok: true });
});

bus.publish("ts.bus-cancel-fetch-abort-demo.abort-demo", {});
output({ handlerGotCtrl, signalNotInitiallyAborted: signalWasAborted });
