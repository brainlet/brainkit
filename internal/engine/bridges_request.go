package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// registerRequestBridges adds __go_brainkit_request (sync) and __go_brainkit_request_async bridges.
func (k *Kernel) registerRequestBridges(qctx *quickjs.Context, invoker *LocalInvoker) {
	// __go_brainkit_request(topic, payloadJSON) → resultJSON (SYNCHRONOUS)
	qctx.Globals().Set(js.JSBridgeRequest,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_request: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			// Tracing
			span := k.tracer.StartSpan("command:"+topic, context.Background())
			resp, err := invoker.Invoke(context.Background(), topic, payload)
			span.End(err)
			if err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			return qctx.NewString(string(resp))
		}))

	// __go_brainkit_request_async(topic, payloadJSON) → Promise<resultJSON> (ASYNC)
	qctx.Globals().Set(js.JSBridgeRequestAsync,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_request_async: expected 2 args"})
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				k.bridge.Go(func(goCtx context.Context) {
					span := k.tracer.StartSpan("command:"+topic, goCtx)
					resp, err := invoker.Invoke(goCtx, topic, payload)
					span.End(err)
					if err != nil {
						if goCtx.Err() != nil {
							return
						}
						// Extract BrainkitError code if available
						var bkErr sdkerrors.BrainkitError
						errCode := "INTERNAL_ERROR"
						errDetailsJSON := "{}"
						if errors.As(err, &bkErr) {
							errCode = bkErr.Code()
							if d := bkErr.Details(); d != nil {
								if b, e := json.Marshal(d); e == nil {
									errDetailsJSON = string(b)
								}
							}
						}
						errMsg := fmt.Sprintf("brainkit_request %s: %s", topic, err.Error())
						qctx.Schedule(func(qctx *quickjs.Context) {
							script := fmt.Sprintf(`(typeof BrainkitError === "function") ? new BrainkitError(%q, %q, JSON.parse(%q)) : new Error(%q)`,
								errMsg, errCode, errDetailsJSON, errMsg)
							errVal := qctx.Eval(script)
							if errVal.IsException() {
								errVal = qctx.NewError(fmt.Errorf("%s", errMsg))
							}
							defer errVal.Free()
							reject(errVal)
						})
						return
					}

					qctx.Schedule(func(qctx *quickjs.Context) {
						resolve(qctx.NewString(string(resp)))
					})
				})
			})
		}))
}
