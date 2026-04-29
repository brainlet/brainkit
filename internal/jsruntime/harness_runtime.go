package jsruntime

import (
	"context"

	quickjs "github.com/buke/quickjs-go"
)

// HarnessRuntime adapts the runtime bridge surface onto the narrow
// interface that modules/harness expects. It is deliberately untyped
// (any) in the Kit accessor so the brainkit package doesn't have to
// import quickjs-go; the harness module type-asserts onto its local
// Runtime interface.
type HarnessRuntime struct{ r *Runtime }

func (r *Runtime) HarnessRuntime() any {
	return r.harnessRuntime()
}

func (r *Runtime) harnessRuntime() *HarnessRuntime {
	if r.bridge == nil {
		return nil
	}
	return &HarnessRuntime{r: r}
}

func (h *HarnessRuntime) EvalTS(ctx context.Context, filename, code string) (string, error) {
	return h.r.EvalTS(ctx, filename, code)
}

func (h *HarnessRuntime) BridgeIsEvalBusy() bool { return h.r.bridge.IsEvalBusy() }

func (h *HarnessRuntime) BridgeEval(filename, code string) (*quickjs.Value, error) {
	return h.r.bridge.Eval(filename, code)
}

func (h *HarnessRuntime) BridgeEvalOnJSThread(filename, code string) (string, error) {
	return h.r.bridge.EvalOnJSThread(filename, code)
}

func (h *HarnessRuntime) BridgeContext() *quickjs.Context { return h.r.bridge.Context() }

func (h *HarnessRuntime) BridgeGoContext() context.Context { return h.r.bridge.GoContext() }
