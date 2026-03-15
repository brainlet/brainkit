package jsbridge

import (
	"strconv"
	"sync"
	"time"

	quickjs "github.com/buke/quickjs-go"
)

type timerEntry struct {
	id      int32
	delay   time.Duration
	cleared bool
}

// TimersPolyfill provides setTimeout and clearTimeout with a drain mechanism.
// QuickJS is single-threaded, so timers are stored and executed via Drain().
type TimersPolyfill struct {
	mu     sync.Mutex
	timers map[int32]*timerEntry
	nextID int32
	ctx    *quickjs.Context
}

// Timers creates a timers polyfill.
func Timers() *TimersPolyfill {
	return &TimersPolyfill{timers: make(map[int32]*timerEntry)}
}

func (p *TimersPolyfill) Name() string { return "timers" }

// Drain executes all pending timers in delay order.
// Call after Eval() to fire setTimeout callbacks.
func (p *TimersPolyfill) Drain(ctx *quickjs.Context) error {
	for {
		p.mu.Lock()
		if len(p.timers) == 0 {
			p.mu.Unlock()
			return nil
		}

		var best *timerEntry
		for _, t := range p.timers {
			if t.cleared {
				continue
			}
			if best == nil || t.delay < best.delay {
				best = t
			}
		}
		if best == nil {
			p.mu.Unlock()
			return nil
		}

		id := best.id
		delay := best.delay
		delete(p.timers, id)
		p.mu.Unlock()

		if delay > 0 {
			time.Sleep(delay)
		}

		// Retrieve and invoke the callback via the JS-side storage map.
		// __timer_cbs.get(id) returns the function; __timer_cbs.delete(id) cleans up.
		ids := strconv.FormatInt(int64(id), 10)
		result := ctx.Eval(`(function() { var cb = __timer_cbs.get(` + ids + `); __timer_cbs.delete(` + ids + `); return cb(); })()`)
		if result.IsException() {
			err := ctx.Exception()
			result.Free()
			return err
		}
		result.Free()
	}
}

func (p *TimersPolyfill) Setup(ctx *quickjs.Context) error {
	p.ctx = ctx

	ctx.Globals().Set("__go_set_timeout", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.NewInt32(0)
		}

		delay := int32(0)
		if len(args) > 1 {
			delay = args[1].ToInt32()
		}

		p.mu.Lock()
		p.nextID++
		id := p.nextID
		p.timers[id] = &timerEntry{
			id:    id,
			delay: time.Duration(delay) * time.Millisecond,
		}
		p.mu.Unlock()

		return ctx.NewInt32(id)
	}))

	ctx.Globals().Set("__go_clear_timeout", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) > 0 {
			id := args[0].ToInt32()
			p.mu.Lock()
			if t, ok := p.timers[id]; ok {
				t.cleared = true
				delete(p.timers, id)
			}
			p.mu.Unlock()
		}
		return ctx.NewUndefined()
	}))

	// Initialize the JS-side callback storage map and setTimeout/clearTimeout wrappers.
	// For delay <= 0 (nextTick-style scheduling), use queueMicrotask so callbacks
	// fire during the Await loop's JS_ExecutePendingJob processing. This is critical
	// for Mastra's internal workflow step scheduling which uses setTimeout(fn, 0).
	// Delayed timers still use the Go-side Drain() mechanism.
	return evalJS(ctx, `
if (typeof queueMicrotask === "undefined") {
  globalThis.queueMicrotask = function(fn) { Promise.resolve().then(fn); };
}
var __timer_cleared = new Set();
var __timer_next_id = 0;
globalThis.setTimeout = function(fn, delay) {
  __timer_next_id++;
  var id = __timer_next_id;
  var args = [];
  for (var i = 2; i < arguments.length; i++) args.push(arguments[i]);
  // ALL setTimeout calls use queueMicrotask so they fire during the Await loop.
  // Go-side Drain() is never called during EvalAsync, so timers sent to Go
  // would never fire. queueMicrotask ensures callbacks run via JS_ExecutePendingJob.
  // The delay is ignored — in our single-threaded QuickJS, there's no real
  // async scheduling, just microtask ordering.
  queueMicrotask(function() {
    if (!__timer_cleared.has(id)) {
      fn.apply(null, args);
    }
    __timer_cleared.delete(id);
  });
  return id;
};
globalThis.clearTimeout = function(id) {
  __timer_cleared.add(id);
};
`)
}
