package jsbridge

import (
	"sync"
	"time"

	"github.com/fastschema/qjs"
)

type timerEntry struct {
	id       int32
	callback *qjs.Value
	delay    time.Duration
	cleared  bool
}

// TimersPolyfill provides setTimeout and clearTimeout with a drain mechanism.
// QuickJS is single-threaded, so timers are stored and executed via Drain().
type TimersPolyfill struct {
	mu     sync.Mutex
	timers map[int32]*timerEntry
	nextID int32
}

// Timers creates a timers polyfill.
func Timers() *TimersPolyfill {
	return &TimersPolyfill{timers: make(map[int32]*timerEntry)}
}

func (p *TimersPolyfill) Name() string { return "timers" }

// Drain executes all pending timers in delay order.
// Call after Eval() to fire setTimeout callbacks.
func (p *TimersPolyfill) Drain(ctx *qjs.Context) error {
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

		cb := best.callback
		delay := best.delay
		delete(p.timers, best.id)
		p.mu.Unlock()

		if delay > 0 {
			time.Sleep(delay)
		}

		result, err := ctx.Invoke(cb, ctx.Global())
		if err != nil {
			cb.Free()
			return err
		}
		if result != nil {
			result.Free()
		}
		cb.Free()
	}
}

func (p *TimersPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_set_timeout", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return this.Context().NewInt32(0), nil
		}

		callback := args[0].Clone()
		delay := int32(0)
		if len(args) > 1 {
			delay = args[1].Int32()
		}

		p.mu.Lock()
		p.nextID++
		id := p.nextID
		p.timers[id] = &timerEntry{
			id:       id,
			callback: callback,
			delay:    time.Duration(delay) * time.Millisecond,
		}
		p.mu.Unlock()

		return this.Context().NewInt32(id), nil
	})

	ctx.SetFunc("__go_clear_timeout", func(this *qjs.This) (*qjs.Value, error) {
		if args := this.Args(); len(args) > 0 {
			id := args[0].Int32()
			p.mu.Lock()
			if t, ok := p.timers[id]; ok {
				t.cleared = true
				t.callback.Free()
				delete(p.timers, id)
			}
			p.mu.Unlock()
		}
		return this.Context().NewUndefined(), nil
	})

	return evalJS(ctx, `
globalThis.setTimeout = (fn, delay) => __go_set_timeout(fn, delay || 0);
globalThis.clearTimeout = (id) => __go_clear_timeout(id);
`)
}
