package jsruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	quickjs "github.com/buke/quickjs-go"
	"github.com/google/uuid"
)

func callerCfg(targetNamespace string) sdk.CallerConfig {
	return sdk.CallerConfig{TargetNamespace: targetNamespace}
}

// registerBusBridges adds bus_send, bus_publish, bus_emit, bus_reply, subscribe, unsubscribe bridges.
func (r *Runtime) registerBusBridges(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeBusSend,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.send: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			if topic == "" {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())

			if r.host.HasCommand(topic) {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: topic + " is a command topic; use bridgeRequest for commands"})
			}

			if err := r.host.ValidateEvent(topic, payload); err != nil {
				return r.throwBrainkitError(qctx, err)
			}
			if err := r.host.PublishEvent(context.Background(), topic, payload); err != nil {
				return r.throwBrainkitError(qctx, err)
			}
			return qctx.NewUndefined()
		}))

	// __go_brainkit_bus_publish(topic, payloadJSON) → JSON string {replyTo, correlationId}
	// Publishes a message with auto-generated replyTo, returns routing info to JS.
	qctx.Globals().Set(js.JSBridgeBusPublish,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.publish: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			if topic == "" {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())

			// Tracing
			span := r.host.Tracer().StartSpan("bus.publish:"+topic, context.Background())

			correlationID := uuid.NewString()
			replyTo := topic + ".reply." + correlationID

			ctx := transport.WithPublishMeta(context.Background(), correlationID, replyTo)
			_, err := r.host.Remote().PublishRaw(ctx, topic, payload)
			span.End(err)
			if err != nil {
				return r.throwBrainkitError(qctx, err)
			}

			result, _ := json.Marshal(map[string]string{
				"replyTo":       replyTo,
				"correlationId": correlationID,
			})
			return qctx.NewString(string(result))
		}))

	// __go_brainkit_bus_emit(topic, payloadJSON) → void
	// Fire-and-forget publish. No replyTo, no correlationId returned.
	qctx.Globals().Set(js.JSBridgeBusEmit,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.emit: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			if topic == "" {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())

			// Block command topics — same check as bus_send (fixes bug #8)
			if r.host.HasCommand(topic) {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: topic + " is a command topic; use bridgeRequest for commands"})
			}

			if err := r.host.PublishEvent(context.Background(), topic, payload); err != nil {
				return r.throwBrainkitError(qctx, err)
			}
			return qctx.NewUndefined()
		}))

	// __go_brainkit_bus_reply(replyTo, payloadJSON, correlationId, done, envelope?) → void
	qctx.Globals().Set(js.JSBridgeBusReply,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.reply: expected 4+ args"})
			}
			replyTo := args[0].String()
			payload := args[1].String()
			correlationID := args[2].String()
			done := args[3].ToBool()
			envelope := false
			if len(args) >= 5 {
				envelope = args[4].ToBool()
			}
			if replyTo == "" {
				return qctx.NewUndefined()
			}

			// replyTo is already namespaced+sanitized by the publisher
			if err := r.host.ReplyRawWithEnvelope(context.Background(), replyTo, correlationID, json.RawMessage(payload), done, envelope); err != nil {
				return r.throwBrainkitError(qctx, &sdkerrors.TransportError{Operation: "bus.reply", Cause: err})
			}

			// Stream heartbeat management — start on first stream message, stop on done
			if done {
				r.host.StopStreamHeartbeat(replyTo)
			} else if strings.Contains(payload, `"type"`) {
				// Only start heartbeat for stream protocol messages (have "type" field).
				// msg.stream.*() calls always include type, raw msg.send() calls don't.
				r.host.StartStreamHeartbeat(replyTo, correlationID)
			}

			return qctx.NewUndefined()
		}))

	// __go_brainkit_bus_call(topic, payloadJSON, targetNamespace, timeoutMs) → Promise<envelope JSON>
	// Uses the shared-inbox Caller to publish + await a terminal reply.
	// Envelope unwrap happens in JS land (__kit_bus.call throws BrainkitError on ok=false).
	qctx.Globals().Set(js.JSBridgeBusCall,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.call: expected 4 args (topic, payload, targetNamespace, timeoutMs)"})
			}
			topic := args[0].String()
			if topic == "" {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())
			targetNS := args[2].String()
			timeoutMs := args[3].ToInt32()
			if timeoutMs <= 0 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "timeoutMs", Message: "bus.call: timeoutMs is required (> 0)"})
			}

			c := r.host.Caller()
			if c == nil {
				return r.throwBrainkitError(qctx, &sdkerrors.BridgeError{Function: "bus.call", Cause: fmt.Errorf("caller not initialized")})
			}

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				r.bridge.Go(func(goCtx context.Context) {
					callCtx, cancel := context.WithTimeout(goCtx, time.Duration(timeoutMs)*time.Millisecond)
					defer cancel()
					span := r.host.Tracer().StartSpan("bus.call:"+topic, callCtx)
					cfg := callerCfg(targetNS)
					replyPayload, err := c.Call(callCtx, topic, payload, cfg)
					span.End(err)
					if err != nil {
						if goCtx.Err() != nil {
							return
						}
						// Extract typed code for the JS BrainkitError thrown to user
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
						errMsg := err.Error()
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
					// Success: the Caller already unwrapped the envelope and
					// returned raw data bytes. JS side receives the data
					// directly. For callers that want the raw envelope,
					// wrap it back up.
					raw := string(replyPayload)
					if raw == "" {
						raw = "null"
					}
					qctx.Schedule(func(qctx *quickjs.Context) {
						resolve(qctx.NewString(raw))
					})
				})
			})
		}))

	qctx.Globals().Set(js.JSBridgeSubscribe,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "subscribe: expected topic pattern"})
			}
			topic := args[0].String()

			// Capture deployment source at subscribe time for tracing during callbacks
			subscriberSource := r.CurrentSource()

			subID := uuid.NewString()
			cancel, err := r.host.SubscribeEvent(topic, func(msg sdk.Message) {
				// Reject if draining (graceful shutdown)
				if !r.host.EnterHandler() {
					return
				}

				// Tracing — build span context from inbound message metadata
				spanCtx := context.Background()
				if traceID := msg.Metadata["traceId"]; traceID != "" {
					spanCtx = tracing.WithTraceContext(spanCtx, tracing.TraceContext{
						TraceID:  traceID,
						ParentID: msg.Metadata["parentSpanId"],
					})
				}
				if msg.Metadata["traceSampled"] == "false" {
					spanCtx = tracing.WithSampled(spanCtx, false)
				}
				handlerSpan := r.host.Tracer().StartSpan("handler:"+topic, spanCtx)
				handlerSpan.SetSource(subscriberSource)

				// Build full message JSON with metadata for JS handlers
				msgObj := map[string]any{
					"topic": msg.Topic,
				}
				if len(msg.Payload) > 0 && (msg.Payload[0] == '{' || msg.Payload[0] == '[' || msg.Payload[0] == '"') {
					msgObj["payload"] = json.RawMessage(msg.Payload)
				} else {
					msgObj["payload"] = string(msg.Payload)
				}
				if msg.CallerID != "" {
					msgObj["callerId"] = msg.CallerID
				}
				if msg.Metadata != nil {
					if v := msg.Metadata["replyTo"]; v != "" {
						msgObj["replyTo"] = v
					}
					if v := msg.Metadata["correlationId"]; v != "" {
						msgObj["correlationId"] = v
					}
					if v := msg.Metadata["traceId"]; v != "" {
						msgObj["traceId"] = v
					}
				}

				msgJSON, _ := json.Marshal(msgObj)
				quoted := strconv.Quote(string(msgJSON))

				qctx.Schedule(func(qctx *quickjs.Context) {
					defer r.host.ExitHandler()
					defer handlerSpan.End(nil)
					r.SetCurrentSource(subscriberSource)
					defer r.SetCurrentSource("")

					script := fmt.Sprintf(`(function(){ var fn = globalThis.`+js.JSBusSubs+`[%q]; if (typeof fn === "function") { return fn(JSON.parse(%s)); } })()`, subID, quoted)
					val := qctx.Eval(script)
					if val == nil {
						return
					}

					if val.IsException() {
						handlerErr := qctx.Exception()
						val.Free()
						handlerErr = r.enrichHandlerErr(qctx, handlerErr)
						r.host.HandleHandlerFailure(msg, topic, handlerErr)
						return
					}

					if val.IsPromise() {
						awaited := qctx.Await(val)
						if awaited == nil {
							return
						}
						if awaited.IsException() || qctx.HasException() {
							handlerErr := qctx.Exception()
							awaited.Free()
							handlerErr = r.enrichHandlerErr(qctx, handlerErr)
							r.host.HandleHandlerFailure(msg, topic, handlerErr)
							return
						}
						awaited.Free()
					} else {
						val.Free()
					}
				})
			})
			if err != nil {
				return r.throwBrainkitError(qctx, err)
			}
			r.addBridgeSub(subID, cancel)
			return qctx.NewString(subID)
		}))

	qctx.Globals().Set(js.JSBridgeUnsubscribe,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return r.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "subscriptionId", Message: "unsubscribe: expected subscription ID"})
			}
			subID := args[0].String()
			cancel := r.removeBridgeSub(subID)
			if cancel != nil {
				cancel()
			}
			return qctx.NewUndefined()
		}))
}
