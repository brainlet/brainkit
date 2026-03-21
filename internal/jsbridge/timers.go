package jsbridge

import (
	"context"
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

// TimersPolyfill provides setTimeout and clearTimeout.
// delay=0 fires via queueMicrotask (processed during Await's JS_ExecutePendingJob).
// delay>0 fires via Bridge.Go() + ctx.Schedule() (processed during Await's polling loop).
type TimersPolyfill struct {
	bridge *Bridge
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

func (p *TimersPolyfill) SetBridge(b *Bridge) { p.bridge = b }

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

	// __go_schedule_timeout(id, delay) — schedules a Go-side timer that fires the JS callback
	// via ctx.Schedule(). This is used for non-zero delays so they actually wait.
	ctx.Globals().Set("__go_schedule_timeout", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 || p.bridge == nil {
			return qctx.NewUndefined()
		}
		id := args[0].ToInt32()
		delayMs := args[1].ToInt32()

		p.bridge.Go(func(goCtx context.Context) {
			select {
			case <-time.After(time.Duration(delayMs) * time.Millisecond):
			case <-goCtx.Done():
				return
			}
			ids := strconv.FormatInt(int64(id), 10)
			qctx.Schedule(func(qctx *quickjs.Context) {
				qctx.Eval(`(function() {
					if (!__timer_cleared.has(` + ids + `)) {
						var cb = __timer_cbs.get(` + ids + `);
						if (cb) cb();
					}
					__timer_cbs.delete(` + ids + `);
					__timer_cleared.delete(` + ids + `);
				})()`)
			})
		})
		return qctx.NewUndefined()
	}))

	// setTimeout/clearTimeout JS wrappers.
	// delay=0: use queueMicrotask (fires during Await's JS_ExecutePendingJob).
	// delay>0: use Go-side timer via __go_schedule_timeout (fires during Await's polling loop).
	return evalJS(ctx, `
if (typeof queueMicrotask === "undefined") {
  globalThis.queueMicrotask = function(fn) { Promise.resolve().then(fn); };
}
var __timer_cleared = new Set();
var __timer_cbs = new Map();
var __timer_next_id = 0;
globalThis.setTimeout = function(fn, delay) {
  __timer_next_id++;
  var id = __timer_next_id;
  var args = [];
  for (var i = 2; i < arguments.length; i++) args.push(arguments[i]);
  var wrapped = function() { fn.apply(null, args); };

  if (!delay || delay <= 0) {
    // Zero delay: fire as microtask (Mastra workflow step scheduling)
    queueMicrotask(function() {
      if (!__timer_cleared.has(id)) {
        wrapped();
      }
      __timer_cleared.delete(id);
    });
  } else {
    // Non-zero delay: Go-side timer via Bridge.Go() + ctx.Schedule()
    __timer_cbs.set(id, wrapped);
    __go_schedule_timeout(id, delay);
  }
  return id;
};
globalThis.clearTimeout = function(id) {
  __timer_cleared.add(id);
  __timer_cbs.delete(id);
};
`)
}
