package jsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	quickjs "github.com/buke/quickjs-go"
)

// registerApprovalBridges adds __go_brainkit_await_approval for bus-based HITL tool approval.
func (r *Runtime) registerApprovalBridges(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeAwaitApproval,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 3 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "await_approval: expected 3 args"})
			}
			approvalTopic := args[0].String()
			payload := json.RawMessage(args[1].String())
			timeoutMs := args[2].ToInt64()
			if timeoutMs <= 0 {
				timeoutMs = 30000
			}

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				r.bridge.Go(func(goCtx context.Context) {
					timeout := time.Duration(timeoutMs) * time.Millisecond
					waitCtx, waitCancel := context.WithTimeout(goCtx, timeout)
					defer waitCancel()

					caller := r.bus.Caller()
					if caller == nil {
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("await_approval: caller not initialized"))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}

					responseJSON, callErr := caller.Call(waitCtx, approvalTopic, payload, sdk.CallerConfig{})
					if callErr != nil {
						var timeoutErr *sdk.CallTimeoutError
						var cancelledErr *sdk.CallCancelledError
						if errors.As(callErr, &timeoutErr) || errors.As(callErr, &cancelledErr) {
							timeoutJSON := `{"approved":false,"reason":"timeout"}`
							qctx.Schedule(func(qctx *quickjs.Context) {
								resolve(qctx.NewString(timeoutJSON))
							})
							return
						}
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("await_approval: call: %w", callErr))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}
					qctx.Schedule(func(qctx *quickjs.Context) {
						resolve(qctx.NewString(string(responseJSON)))
					})
				})
			})
		}))
}
