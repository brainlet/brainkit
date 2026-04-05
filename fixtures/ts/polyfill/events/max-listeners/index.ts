// Test: EventEmitter.setMaxListeners, prependListener, eventNames, statics
import { output } from "kit";

const ee = new EventEmitter();

// setMaxListeners + getMaxListeners
ee.setMaxListeners(100);

// prependListener — should fire before normal listeners
const order: string[] = [];
ee.on("test", () => order.push("normal"));
ee.prependListener("test", () => order.push("prepended"));
ee.emit("test");

// eventNames
ee.on("foo", () => {});
ee.on("bar", () => {});

// off (alias for removeListener)
const fn = () => {};
ee.on("removeme", fn);
ee.off("removeme", fn);

output({
  maxListeners: ee.getMaxListeners(),
  order,
  eventNames: ee.eventNames().sort(),
  removedCount: ee.listenerCount("removeme"),
  captureRejections: EventEmitter.captureRejections,
  defaultMaxListeners: EventEmitter.defaultMaxListeners,
});
