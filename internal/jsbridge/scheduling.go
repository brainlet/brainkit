package jsbridge

import quickjs "github.com/buke/quickjs-go"

// SchedulingPolyfill provides setImmediate, clearImmediate, setInterval, clearInterval.
// queueMicrotask is also provided as a fallback.
// These are used by Mastra's async tool execution pipeline and various SDK internals.
//
// NOTE: setTimeout/clearTimeout are handled by timers.go (Go-backed with Bridge.Go()).
// setImmediate uses queueMicrotask (fires within the same JS turn).
// setInterval simulates repeated execution via recursive setTimeout.
type SchedulingPolyfill struct{}

func Scheduling() *SchedulingPolyfill { return &SchedulingPolyfill{} }

func (p *SchedulingPolyfill) Name() string { return "scheduling" }

func (p *SchedulingPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
if (typeof queueMicrotask === "undefined") {
  globalThis.queueMicrotask = function(fn) { Promise.resolve().then(fn); };
}
if (typeof setImmediate === "undefined") {
  globalThis.setImmediate = function(fn) {
    var args = [];
    for (var i = 1; i < arguments.length; i++) args.push(arguments[i]);
    Promise.resolve().then(function() { fn.apply(null, args); });
    return 0;
  };
}
if (typeof clearImmediate === "undefined") {
  globalThis.clearImmediate = function() {};
}
if (typeof setInterval === "undefined") {
  var __intervals = {};
  var __intervalId = 0;
  globalThis.setInterval = function(fn, delay) {
    var args = [];
    for (var i = 2; i < arguments.length; i++) args.push(arguments[i]);
    __intervalId++;
    var id = __intervalId;
    function tick() {
      if (!__intervals[id]) return;
      fn.apply(null, args);
      __intervals[id] = setTimeout(tick, delay || 0);
    }
    __intervals[id] = setTimeout(tick, delay || 0);
    return id;
  };
  globalThis.clearInterval = function(id) {
    if (__intervals[id]) {
      clearTimeout(__intervals[id]);
      delete __intervals[id];
    }
  };
}
`)
}
