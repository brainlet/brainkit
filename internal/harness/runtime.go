// Package harness provides the QuickJS runtime integration layer for the
// Harness orchestrator. It isolates all direct QuickJS/bridge interactions
// behind a Runtime struct, so the root brainkit.Harness delegates JS calls
// here without the public type surface needing to move.
package harness

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/jsbridge"
	quickjs "github.com/buke/quickjs-go"
)

// EventHandler is called synchronously for every Harness event from JS.
type EventHandler func(jsonStr string)

// LockFuncs provides thread lock functions for JS bridge registration.
type LockFuncs struct {
	Acquire func(threadID string) error
	Release func(threadID string) error
}

// EvalFunc evaluates TypeScript code via the Kit's QuickJS runtime.
type EvalFunc func(ctx context.Context, filename, code string) (string, error)

// Runtime wraps all QuickJS interactions for the Harness.
// It registers bridge globals, provides JS call helpers, and dispatches events.
type Runtime struct {
	bridge *jsbridge.Bridge
	evalTS EvalFunc
}

// New creates a Runtime from the given bridge and eval function.
func New(bridge *jsbridge.Bridge, evalTS EvalFunc) *Runtime {
	return &Runtime{
		bridge: bridge,
		evalTS: evalTS,
	}
}

// RegisterEventBridge registers the __go_harness_event global.
// JS calls this for every Harness event: __go_harness_event(jsonString)
func (r *Runtime) RegisterEventBridge(handler EventHandler) {
	qctx := r.bridge.Context()
	qctx.Globals().Set("__go_harness_event",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.NewUndefined()
			}
			jsonStr := args[0].String()
			handler(jsonStr)
			return qctx.NewUndefined()
		}))
}

// RegisterLockBridges registers __go_harness_lock_acquire and _release globals.
func (r *Runtime) RegisterLockBridges(lock LockFuncs) {
	qctx := r.bridge.Context()

	qctx.Globals().Set("__go_harness_lock_acquire",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("harness lock acquire: expected threadId"))
			}
			threadID := args[0].String()
			if err := lock.Acquire(threadID); err != nil {
				return qctx.NewString(err.Error())
			}
			return qctx.NewNull()
		}))

	qctx.Globals().Set("__go_harness_lock_release",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("harness lock release: expected threadId"))
			}
			threadID := args[0].String()
			if err := lock.Release(threadID); err != nil {
				return qctx.NewString(err.Error())
			}
			return qctx.NewNull()
		}))
}

// CallJS calls a method on the JS __brainkit_harness object and returns the result.
// The method is awaited (most Harness methods are async).
func (r *Runtime) CallJS(method string, argsJSON string) (string, error) {
	var code string
	if argsJSON == "" {
		code = fmt.Sprintf(`return JSON.stringify(await __brainkit_harness.%s())`, method)
	} else {
		code = fmt.Sprintf(`return JSON.stringify(await __brainkit_harness.%s(JSON.parse(%s)))`, method, QuoteJSString(argsJSON))
	}
	return r.evalTS(r.bridge.GoContext(), "harness-call.ts", code)
}

// CallJSVoid calls a JS method and discards the result.
func (r *Runtime) CallJSVoid(method string, argsJSON string) error {
	_, err := r.CallJS(method, argsJSON)
	return err
}

// CallJSSimple calls a method that returns a primitive (string, bool, number).
func (r *Runtime) CallJSSimple(method string) (string, error) {
	code := fmt.Sprintf(`return JSON.stringify(await __brainkit_harness.%s())`, method)
	return r.evalTS(r.bridge.GoContext(), "harness-simple.ts", code)
}

// EvalDirect evaluates a code string via EvalTS.
func (r *Runtime) EvalDirect(filename, code string) (string, error) {
	return r.evalTS(r.bridge.GoContext(), filename, code)
}

// EvalBridgeDirect evaluates code on the bridge, choosing the reentrant path
// if the bridge is already busy (e.g., during agent stream).
func (r *Runtime) EvalBridgeDirect(filename, code string) (string, error) {
	if r.bridge.IsEvalBusy() {
		return r.bridge.EvalOnJSThread(filename, code)
	}
	val, err := r.bridge.Eval(filename, code)
	if err != nil {
		return "", err
	}
	if val != nil {
		defer val.Free()
		return val.String(), nil
	}
	return "", nil
}

// QuoteJSString returns a JSON-encoded string (valid JS string literal).
func QuoteJSString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// InitHarnessJS creates the JS Harness via createHarness() and calls init().
func (r *Runtime) InitHarnessJS(configJSON string) error {
	createCode := fmt.Sprintf(`await __kit.createHarness(%s)`, QuoteJSString(configJSON))
	if _, err := r.evalTS(context.Background(), "harness-create.ts", createCode); err != nil {
		return fmt.Errorf("harness: create JS harness: %w", err)
	}

	if _, err := r.evalTS(context.Background(), "harness-init.ts", `await __brainkit_harness.init()`); err != nil {
		return fmt.Errorf("harness: init: %w", err)
	}
	return nil
}
