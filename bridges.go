package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/bus"
	quickjs "github.com/buke/quickjs-go"
)

// registerBridges adds Go bridge functions to the Kit's QuickJS context.
func (k *Kit) registerBridges() {
	qctx := k.bridge.Context()

	// __go_brainkit_request(topic, payloadJSON) → resultJSON (SYNCHRONOUS)
	// Used for quick operations: tools.resolve, small lookups.
	// Blocks the QuickJS thread until the bus response arrives.
	qctx.Globals().Set("__go_brainkit_request",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_request: expected 2 args (topic, payload)"))
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			goCtx := context.Background()
			resp, err := k.Bus.Request(goCtx, topic, k.callerID, payload)
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("brainkit_request %s: %w", topic, err))
			}

			return qctx.NewString(string(resp.Payload))
		}))

	// __go_brainkit_request_async(topic, payloadJSON) → Promise<resultJSON> (ASYNC)
	// Used for I/O operations: tools.call (may hit plugin gRPC), bus.request.
	// Frees the QuickJS thread during the bus call — other JS can run.
	qctx.Globals().Set("__go_brainkit_request_async",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_request_async: expected 2 args"))
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				k.bridge.Go(func(goCtx context.Context) {
					resp, err := k.Bus.Request(goCtx, topic, k.callerID, payload)
					if err != nil {
						if goCtx.Err() != nil {
							return // bridge closing
						}
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("brainkit_request %s: %w", topic, err))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}

					qctx.Schedule(func(qctx *quickjs.Context) {
						resolve(qctx.NewString(string(resp.Payload)))
					})
				})
			})
		}))

	// __go_brainkit_subscribe(topic) → subscriptionID (STRING)
	// Registers a bus subscription. When messages arrive matching the topic pattern,
	// the JS callback stored at globalThis.__bus_subs[subId] is invoked via Schedule.
	qctx.Globals().Set("__go_brainkit_subscribe",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("brainkit_subscribe: expected topic pattern"))
			}
			pattern := args[0].String()

			// Use a pointer so the callback can reference the ID after Subscribe returns
			var subIDStr string

			subID, err := k.Bus.Subscribe(pattern, func(msg bus.Message) {
				payloadJSON, _ := json.Marshal(map[string]any{
					"topic":    msg.Topic,
					"callerID": msg.CallerID,
					"payload":  json.RawMessage(msg.Payload),
					"traceID":  msg.TraceID,
				})
				payloadStr := string(payloadJSON)
				sid := subIDStr // capture for closure
				qctx.Schedule(func(qctx *quickjs.Context) {
					qctx.Eval(fmt.Sprintf(
						`(function() { var fn = globalThis.__bus_subs && globalThis.__bus_subs[%q]; if (fn) fn(%s); })()`,
						sid, payloadStr,
					))
				})
			})
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("brainkit_subscribe: %w", err))
			}
			subIDStr = string(subID)

			return qctx.NewString(subIDStr)
		}))

	// __go_brainkit_unsubscribe(subscriptionID)
	qctx.Globals().Set("__go_brainkit_unsubscribe",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.NewUndefined()
			}
			subID := bus.SubscriptionID(args[0].String())
			k.Bus.Unsubscribe(subID)
			return qctx.NewUndefined()
		}))

	// Set context globals
	qctx.Globals().Set("__brainkit_sandbox_id", qctx.NewString(k.agents.ID()))
	qctx.Globals().Set("__brainkit_sandbox_namespace", qctx.NewString(k.namespace))
	qctx.Globals().Set("__brainkit_sandbox_callerID", qctx.NewString(k.callerID))
}
