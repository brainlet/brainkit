package jsruntime

import (
	"context"
	"fmt"

	harnesscap "github.com/brainlet/brainkit/modulecap/harness"
	quickjs "github.com/buke/quickjs-go"
)

// HarnessRuntime adapts the runtime bridge surface onto the typed harness
// capability. The engine attachment still returns it as any so the root
// brainkit package does not import QuickJS-shaped harness contracts.
type HarnessRuntime struct{ r *Runtime }

var _ harnesscap.Runtime = (*HarnessRuntime)(nil)

func (r *Runtime) HarnessRuntime() any {
	return r.harnessRuntime()
}

func (r *Runtime) harnessRuntime() *HarnessRuntime {
	if r.bridge == nil {
		return nil
	}
	return &HarnessRuntime{r: r}
}

func (h *HarnessRuntime) EvalJS(ctx context.Context, filename, code string) (string, error) {
	return h.r.EvalJS(ctx, filename, code)
}

func (h *HarnessRuntime) RuntimeContext() context.Context { return h.r.bridge.GoContext() }

func (h *HarnessRuntime) RegisterEventBridge(handler func(string)) error {
	if handler == nil {
		return fmt.Errorf("harness event bridge requires a handler")
	}
	qctx := h.r.bridge.Context()
	qctx.Globals().Set("__go_harness_event",
		qctx.NewFunction(func(qctx *quickjs.Context, _ *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.NewUndefined()
			}
			handler(args[0].String())
			return qctx.NewUndefined()
		}))
	return nil
}

func (h *HarnessRuntime) RegisterLockBridge(acquire func(string) error, release func(string) error) error {
	if acquire == nil || release == nil {
		return fmt.Errorf("harness lock bridge requires acquire and release handlers")
	}
	qctx := h.r.bridge.Context()
	qctx.Globals().Set("__go_harness_lock_acquire",
		qctx.NewFunction(func(qctx *quickjs.Context, _ *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("harness lock acquire: expected threadId"))
			}
			if err := acquire(args[0].String()); err != nil {
				return qctx.NewString(err.Error())
			}
			return qctx.NewNull()
		}))
	qctx.Globals().Set("__go_harness_lock_release",
		qctx.NewFunction(func(qctx *quickjs.Context, _ *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("harness lock release: expected threadId"))
			}
			if err := release(args[0].String()); err != nil {
				return qctx.NewString(err.Error())
			}
			return qctx.NewNull()
		}))
	return nil
}

func (h *HarnessRuntime) EvalControl(_ context.Context, filename, code string) (string, error) {
	if h.r.bridge.IsEvalBusy() {
		return h.r.bridge.EvalOnJSThread(filename, code)
	}
	val, err := h.r.bridge.Eval(filename, code)
	if err != nil {
		return "", err
	}
	if val == nil {
		return "", nil
	}
	defer val.Free()
	return val.ToString(), nil
}
