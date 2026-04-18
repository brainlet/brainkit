package engine

import (
	"context"

	quickjs "github.com/buke/quickjs-go"
)

// HarnessRuntime adapts the kernel's bridge surface onto the narrow
// interface that modules/harness expects. It is deliberately untyped
// (any) in the Kit accessor so the brainkit package doesn't have to
// import quickjs-go; the harness module type-asserts onto its local
// Runtime interface.
type HarnessRuntime struct{ k *Kernel }

func (k *Kernel) HarnessRuntime() *HarnessRuntime {
	if k.bridge == nil {
		return nil
	}
	return &HarnessRuntime{k: k}
}

func (r *HarnessRuntime) EvalTS(ctx context.Context, filename, code string) (string, error) {
	return r.k.EvalTS(ctx, filename, code)
}

func (r *HarnessRuntime) BridgeIsEvalBusy() bool { return r.k.bridge.IsEvalBusy() }

func (r *HarnessRuntime) BridgeEval(filename, code string) (*quickjs.Value, error) {
	return r.k.bridge.Eval(filename, code)
}

func (r *HarnessRuntime) BridgeEvalOnJSThread(filename, code string) (string, error) {
	return r.k.bridge.EvalOnJSThread(filename, code)
}

func (r *HarnessRuntime) BridgeContext() *quickjs.Context { return r.k.bridge.Context() }

func (r *HarnessRuntime) BridgeGoContext() context.Context { return r.k.bridge.GoContext() }
