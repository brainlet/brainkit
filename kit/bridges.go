package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	quickjs "github.com/buke/quickjs-go"
	"github.com/google/uuid"
)

// registerBridges adds Go bridge functions to the Kernel's QuickJS context.
func (k *Kernel) registerBridges() {
	qctx := k.bridge.Context()
	invoker := newLocalInvoker(k)

	// __go_brainkit_request(topic, payloadJSON) → resultJSON (SYNCHRONOUS)
	qctx.Globals().Set("__go_brainkit_request",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_request: expected 2 args (topic, payload)"))
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			resp, err := invoker.Invoke(context.Background(), topic, payload)
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("brainkit_request %s: %w", topic, err))
			}

			return qctx.NewString(string(resp))
		}))

	// __go_brainkit_request_async(topic, payloadJSON) → Promise<resultJSON> (ASYNC)
	qctx.Globals().Set("__go_brainkit_request_async",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_request_async: expected 2 args"))
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				k.bridge.Go(func(goCtx context.Context) {
					resp, err := invoker.Invoke(goCtx, topic, payload)
					if err != nil {
						if goCtx.Err() != nil {
							return
						}
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("brainkit_request %s: %w", topic, err))
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

	// __go_brainkit_control handles local-only registration operations
	// (tools.register, tools.unregister, agents.register, agents.unregister).
	// These are JS→Go hooks, not transport-visible commands.
	qctx.Globals().Set("__go_brainkit_control",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_control: expected 2 args (action, payload)"))
			}
			action := args[0].String()
			payload := json.RawMessage(args[1].String())

			var resp json.RawMessage
			var err error
			switch action {
			case "tools.register":
				var req struct {
					Name        string          `json:"name"`
					Description string          `json:"description"`
					InputSchema json.RawMessage `json:"inputSchema"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return qctx.ThrowError(fmt.Errorf("tools.register: %w", err))
				}
				fullName, regErr := k.toolsDomain.Register(context.Background(), req.Name, req.Description, req.InputSchema, k.callerID)
				if regErr != nil {
					return qctx.ThrowError(fmt.Errorf("tools.register: %w", regErr))
				}
				resp, _ = json.Marshal(map[string]string{"registered": fullName})
			case "tools.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return qctx.ThrowError(fmt.Errorf("tools.unregister: %w", err))
				}
				if err = k.toolsDomain.Unregister(context.Background(), req.Name); err != nil {
					return qctx.ThrowError(fmt.Errorf("tools.unregister: %w", err))
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "agents.register":
				var req AgentInfo
				if err = json.Unmarshal(payload, &req); err != nil {
					return qctx.ThrowError(fmt.Errorf("agents.register: %w", err))
				}
				if err = k.agentsDomain.Register(context.Background(), req); err != nil {
					return qctx.ThrowError(fmt.Errorf("agents.register: %w", err))
				}
				resp, _ = json.Marshal(map[string]string{"registered": req.Name})
			case "agents.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return qctx.ThrowError(fmt.Errorf("agents.unregister: %w", err))
				}
				if err = k.agentsDomain.Unregister(context.Background(), req.Name); err != nil {
					return qctx.ThrowError(fmt.Errorf("agents.unregister: %w", err))
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			default:
				return qctx.ThrowError(fmt.Errorf("unknown control action: %s", action))
			}
			return qctx.NewString(string(resp))
		}))

	qctx.Globals().Set("__go_brainkit_bus_send",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.ThrowError(fmt.Errorf("brainkit_bus_send: expected 2 args (topic, payload)"))
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			if commandCatalog().HasCommand(topic) {
				return qctx.ThrowError(fmt.Errorf("brainkit_bus_send: %s is a command topic; bus.publish only sends events", topic))
			}
			if err := eventCatalog().Validate(topic, payload); err != nil {
				return qctx.ThrowError(err)
			}
			if err := k.publish(context.Background(), topic, payload); err != nil {
				return qctx.ThrowError(err)
			}
			return qctx.NewUndefined()
		}))

	qctx.Globals().Set("__go_brainkit_subscribe",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("brainkit_subscribe: expected topic pattern"))
			}
			topic := args[0].String()
			subID := uuid.NewString()
			cancel, err := k.subscribe(topic, func(payload []byte) {
				quoted := strconv.Quote(string(payload))
				qctx.Schedule(func(qctx *quickjs.Context) {
					script := fmt.Sprintf(`(function(){ var fn = globalThis.__bus_subs[%q]; if (typeof fn === "function") { fn(JSON.parse(%s)); } })()`, subID, quoted)
					val := qctx.Eval(script)
					if val != nil {
						val.Free()
					}
				})
			})
			if err != nil {
				return qctx.ThrowError(err)
			}
			k.mu.Lock()
			k.bridgeSubs[subID] = cancel
			k.mu.Unlock()
			return qctx.NewString(subID)
		}))

	qctx.Globals().Set("__go_brainkit_unsubscribe",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("brainkit_unsubscribe: expected subscription ID"))
			}
			subID := args[0].String()
			k.mu.Lock()
			cancel := k.bridgeSubs[subID]
			delete(k.bridgeSubs, subID)
			k.mu.Unlock()
			if cancel != nil {
				cancel()
			}
			return qctx.NewUndefined()
		}))

	// Set context globals
	qctx.Globals().Set("__brainkit_sandbox_id", qctx.NewString(k.agents.ID()))
	qctx.Globals().Set("__brainkit_sandbox_namespace", qctx.NewString(k.namespace))
	qctx.Globals().Set("__brainkit_sandbox_callerID", qctx.NewString(k.callerID))
}
