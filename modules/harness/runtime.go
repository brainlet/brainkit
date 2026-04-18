package harness

import (
	"context"

	quickjs "github.com/buke/quickjs-go"
)

// Runtime is the minimal interface harness needs from the Kit runtime.
// Implemented by *brainkit.Kit.
type Runtime interface {
	EvalTS(ctx context.Context, filename, code string) (string, error)
	BridgeIsEvalBusy() bool
	BridgeEval(filename, code string) (*quickjs.Value, error)
	BridgeEvalOnJSThread(filename, code string) (string, error)
	BridgeContext() *quickjs.Context
	BridgeGoContext() context.Context
}
