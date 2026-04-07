package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	quickjs "github.com/buke/quickjs-go"
	"github.com/google/uuid"
	js "github.com/brainlet/brainkit/internal/contract"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/sdk/messages"
)

// registerBusBridges adds bus_send, bus_publish, bus_emit, bus_reply, subscribe, unsubscribe bridges.
func (k *Kernel) registerBusBridges(qctx *quickjs.Context) {
	qctx.Globals().Set(js.JSBridgeBusSend,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.send: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			if topic == "" {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())

			if commandCatalog().HasCommand(topic) {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: topic + " is a command topic; use bridgeRequest for commands"})
			}

			// RBAC enforcement — parity with bus_emit (fixes RBAC bypass via direct bridge call)
			if err := k.checkBusPermission(k.currentDeploymentSource(), topic, "emit"); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			if err := eventCatalog().Validate(topic, payload); err != nil {
				return k.throwBrainkitError(qctx, err)
			}
			if err := k.publish(context.Background(), topic, payload); err != nil {
				return k.throwBrainkitError(qctx, err)
			}
			return qctx.NewUndefined()
		}))

	// __go_brainkit_bus_publish(topic, payloadJSON) → JSON string {replyTo, correlationId}
	// Publishes a message with auto-generated replyTo, returns routing info to JS.
	qctx.Globals().Set(js.JSBridgeBusPublish,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.publish: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			if topic == "" {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())

			// RBAC enforcement
			if err := k.checkBusPermission(k.currentDeploymentSource(), topic, "publish"); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			// Bus rate limiting — per-role token bucket
			if len(k.busRateLimiters) > 0 {
				source := k.currentDeploymentSource()
				if source != "" && k.rbac != nil {
					role := k.rbac.RoleForSource(source)
					if limiter, ok := k.busRateLimiters[role.Name]; ok {
						if !limiter.Allow() {
							return k.throwBrainkitError(qctx, &sdkerrors.RateLimitedError{Role: role.Name, Limit: float64(limiter.Limit())})
						}
					}
				}
			}

			// Tracing
			span := k.tracer.StartSpan("bus.publish:"+topic, context.Background())

			correlationID := uuid.NewString()
			replyTo := topic + ".reply." + correlationID

			ctx := transport.WithPublishMeta(context.Background(), correlationID, replyTo)
			_, err := k.remote.PublishRaw(ctx, topic, payload)
			span.End(err)
			if err != nil {
				return k.throwBrainkitError(qctx, err)
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
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.emit: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			if topic == "" {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "is required"})
			}
			payload := json.RawMessage(args[1].String())

			// Block command topics — same check as bus_send (fixes bug #8)
			if commandCatalog().HasCommand(topic) {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: topic + " is a command topic; use bridgeRequest for commands"})
			}

			// RBAC enforcement
			if err := k.checkBusPermission(k.currentDeploymentSource(), topic, "emit"); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			if err := k.publish(context.Background(), topic, payload); err != nil {
				return k.throwBrainkitError(qctx, err)
			}
			return qctx.NewUndefined()
		}))

	// __go_brainkit_bus_reply(replyTo, payloadJSON, correlationId, done) → void
	qctx.Globals().Set(js.JSBridgeBusReply,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.reply: expected 4 args"})
			}
			replyTo := args[0].String()
			payload := args[1].String()
			correlationID := args[2].String()
			done := args[3].ToBool()
			replyToken := ""
			if len(args) >= 5 {
				replyToken = args[4].String()
			}

			if replyTo == "" {
				return qctx.NewUndefined()
			}

			// Validate reply token when RBAC is active
			if err := k.validateReplyToken(correlationID, replyTo, k.currentDeploymentSource(), replyToken); err != nil {
				k.emitReplyDenied(replyTo, correlationID, k.currentDeploymentSource(), "invalid reply token")
				return k.throwBrainkitError(qctx, err)
			}

			wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
			wmsg.Metadata.Set("correlationId", correlationID)
			if done {
				wmsg.Metadata.Set("done", "true")
			}

			// replyTo is already namespaced+sanitized by the publisher
			if err := k.transport.Publisher.Publish(replyTo, wmsg); err != nil {
				return k.throwBrainkitError(qctx, &sdkerrors.TransportError{Operation: "bus.reply", Cause: err})
			}

			// Stream heartbeat management — start on first stream message, stop on done
			if done {
				k.streamTracker.StopHeartbeat(replyTo)
			} else if strings.Contains(payload, `"type"`) {
				// Only start heartbeat for stream protocol messages (have "type" field).
				// msg.stream.*() calls always include type, raw msg.send() calls don't.
				k.streamTracker.StartHeartbeat(replyTo, correlationID)
			}

			return qctx.NewUndefined()
		}))

	qctx.Globals().Set(js.JSBridgeSubscribe,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: "subscribe: expected topic pattern"})
			}
			topic := args[0].String()

			// Capture deployment source at subscribe time for RBAC during callbacks
			subscriberSource := k.currentDeploymentSource()

			// RBAC enforcement
			if err := k.checkBusPermission(subscriberSource, topic, "subscribe"); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			subID := uuid.NewString()
			cancel, err := k.subscribe(topic, func(msg messages.Message) {
				// Reject if draining (graceful shutdown)
				if !k.enterHandler() {
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
				handlerSpan := k.tracer.StartSpan("handler:"+topic, spanCtx)
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

				// Generate reply token — ONLY own-mailbox subscribers get tokens.
				if msg.Metadata != nil && msg.Metadata["replyTo"] != "" &&
					subscriberSource != "" && rbac.IsOwnMailbox(subscriberSource, topic) {
					replyToken := k.generateReplyToken(
						msg.Metadata["correlationId"],
						msg.Metadata["replyTo"],
						subscriberSource,
					)
					if replyToken != "" {
						msgObj["replyToken"] = replyToken
					}
				}

				msgJSON, _ := json.Marshal(msgObj)
				quoted := strconv.Quote(string(msgJSON))

				qctx.Schedule(func(qctx *quickjs.Context) {
					defer k.exitHandler()
					defer handlerSpan.End(nil)
					k.setCurrentSource(subscriberSource)
					defer k.setCurrentSource("")

					script := fmt.Sprintf(`(function(){ var fn = globalThis.`+js.JSBusSubs+`[%q]; if (typeof fn === "function") { return fn(JSON.parse(%s)); } })()`, subID, quoted)
					val := qctx.Eval(script)
					if val == nil {
						return
					}

					if val.IsException() {
						handlerErr := qctx.Exception()
						val.Free()
						k.handleHandlerFailure(msg, topic, handlerErr)
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
							k.handleHandlerFailure(msg, topic, handlerErr)
							return
						}
						awaited.Free()
					} else {
						val.Free()
					}
				})
			})
			if err != nil {
				return k.throwBrainkitError(qctx, err)
			}
			k.mu.Lock()
			k.bridgeSubs[subID] = cancel
			k.mu.Unlock()
			return qctx.NewString(subID)
		}))

	qctx.Globals().Set(js.JSBridgeUnsubscribe,
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "subscriptionId", Message: "unsubscribe: expected subscription ID"})
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
}
