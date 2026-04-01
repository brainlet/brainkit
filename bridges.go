package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	quickjs "github.com/buke/quickjs-go"
	"github.com/google/uuid"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	provreg "github.com/brainlet/brainkit/registry"
	"github.com/brainlet/brainkit/tracing"
	"github.com/brainlet/brainkit/sdk/messages"
	"errors"
)

// registerBridges adds Go bridge functions to the Kernel's QuickJS context.
func (k *Kernel) registerBridges() {
	qctx := k.bridge.Context()
	invoker := newLocalInvoker(k)

	// __go_brainkit_request(topic, payloadJSON) → resultJSON (SYNCHRONOUS)
	qctx.Globals().Set("__go_brainkit_request",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_request: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			// RBAC enforcement on command
			if err := k.checkCommandPermission(k.currentDeploymentSource(), topic); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

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
	qctx.Globals().Set("__go_brainkit_request_async",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_request_async: expected 2 args"})
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			// RBAC enforcement on command
			if err := k.checkCommandPermission(k.currentDeploymentSource(), topic); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				k.bridge.Go(func(goCtx context.Context) {
					span := k.tracer.StartSpan("command:"+topic, goCtx)
					resp, err := invoker.Invoke(goCtx, topic, payload)
					span.End(err)
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
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "brainkit_control: expected 2 args (action, payload)"})
			}
			action := args[0].String()
			payload := json.RawMessage(args[1].String())

			// RBAC enforcement on registration
			source := k.currentDeploymentSource()
			if action == "tools.register" || action == "tools.unregister" {
				if err := k.checkRegistrationPermission(source, "tool"); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
			} else if action == "agents.register" || action == "agents.unregister" {
				if err := k.checkRegistrationPermission(source, "agent"); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
			}

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
					return k.throwBrainkitError(qctx, err)
				}
				fullName, regErr := k.toolsDomain.Register(context.Background(), req.Name, req.Description, req.InputSchema, k.callerID)
				if regErr != nil {
					return k.throwBrainkitError(qctx, regErr)
				}
				resp, _ = json.Marshal(map[string]string{"registered": fullName})
			case "tools.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				if err = k.toolsDomain.Unregister(context.Background(), req.Name); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "agents.register":
				var req AgentInfo
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				if err = k.agentsDomain.Register(context.Background(), req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]string{"registered": req.Name})
			case "agents.unregister":
				var req struct {
					Name string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				if err = k.agentsDomain.Unregister(context.Background(), req.Name); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "registry.register":
				var req struct {
					Category string          `json:"category"`
					Name     string          `json:"name"`
					Config   json.RawMessage `json:"config"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				// Two-pass unmarshal: read type from config, then register
				var typeHolder struct {
					Type string `json:"type"`
				}
				json.Unmarshal(req.Config, &typeHolder)
				switch req.Category {
				case "provider":
					k.providers.RegisterAIProvider(req.Name, provreg.AIProviderRegistration{
						Type: provreg.AIProviderType(typeHolder.Type),
					})
				case "vectorStore":
					k.providers.RegisterVectorStore(req.Name, provreg.VectorStoreRegistration{
						Type: provreg.VectorStoreType(typeHolder.Type),
					})
				case "storage":
					k.providers.RegisterStorage(req.Name, provreg.StorageRegistration{
						Type: provreg.StorageType(typeHolder.Type),
					})
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			case "registry.unregister":
				var req struct {
					Category string `json:"category"`
					Name     string `json:"name"`
				}
				if err = json.Unmarshal(payload, &req); err != nil {
					return k.throwBrainkitError(qctx, err)
				}
				switch req.Category {
				case "provider":
					k.providers.UnregisterAIProvider(req.Name)
				case "vectorStore":
					k.providers.UnregisterVectorStore(req.Name)
				case "storage":
					k.providers.UnregisterStorage(req.Name)
				}
				resp, _ = json.Marshal(map[string]bool{"ok": true})
			default:
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "action", Message: "unknown control action: " + action})
			}
			return qctx.NewString(string(resp))
		}))

	qctx.Globals().Set("__go_brainkit_bus_send",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.send: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
			payload := json.RawMessage(args[1].String())

			if commandCatalog().HasCommand(topic) {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "topic", Message: topic + " is a command topic; use bridgeRequest for commands"})
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
	qctx.Globals().Set("__go_brainkit_bus_publish",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.publish: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
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

			ctx := messaging.WithPublishMeta(context.Background(), correlationID, replyTo)
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
	qctx.Globals().Set("__go_brainkit_bus_emit",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.emit: expected 2 args (topic, payload)"})
			}
			topic := args[0].String()
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
	// Publishes a response to a specific replyTo topic with correlationId and done flag.
	// Used by msg.reply() (done=true) and msg.send() (done=false) in JS.
	qctx.Globals().Set("__go_brainkit_bus_reply",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.reply: expected 4 args"})
			}
			replyTo := args[0].String()
			payload := args[1].String()
			correlationID := args[2].String()
			done := args[3].ToBool()

			if replyTo == "" {
				return qctx.NewUndefined()
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
			return qctx.NewUndefined()
		}))

	qctx.Globals().Set("__go_brainkit_subscribe",
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
				// Parse payload as raw JSON; fall back to string
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
					defer k.exitHandler()
					defer handlerSpan.End(nil)
					// Set source for RBAC inside the scheduled callback (JS thread).
					// Must be here, not in subscriber goroutine, to avoid races.
					k.setCurrentSource(subscriberSource)
					defer k.setCurrentSource("")

					script := fmt.Sprintf(`(function(){ var fn = globalThis.__bus_subs[%q]; if (typeof fn === "function") { return fn(JSON.parse(%s)); } })()`, subID, quoted)
					val := qctx.Eval(script)
					if val == nil {
						return
					}

					// Sync exception
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
						// Async rejection
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

	qctx.Globals().Set("__go_brainkit_unsubscribe",
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

	// __go_console_log_tagged(source, level, message) — per-Compartment tagged logging
	qctx.Globals().Set("__go_console_log_tagged",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 3 {
				return qctx.NewUndefined()
			}
			source := args[0].String()
			level := args[1].String()
			message := args[2].String()
			k.emitLog(source, level, message)
			return qctx.NewUndefined()
		}))

	// __go_registry_resolve(category, name) → configJSON or ""
	qctx.Globals().Set("__go_registry_resolve",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.NewString("")
			}
			category := args[0].String()
			name := args[1].String()

			var configJSON []byte
			switch category {
			case "provider":
				if reg, ok := k.providers.GetAIProvider(name); ok {
					configJSON, _ = json.Marshal(map[string]any{
						"type":   string(reg.Type),
						"name":   name,
						"config": redactCredentials(reg.Config),
					})
				}
			case "vectorStore":
				if reg, ok := k.providers.GetVectorStore(name); ok {
					configJSON, _ = json.Marshal(map[string]any{
						"type":   string(reg.Type),
						"name":   name,
						"config": redactCredentials(reg.Config),
					})
				}
			case "storage":
				if reg, ok := k.providers.GetStorage(name); ok {
					configJSON, _ = json.Marshal(map[string]any{
						"type":   string(reg.Type),
						"name":   name,
						"config": redactCredentials(reg.Config),
					})
				}
			}
			if configJSON == nil {
				return qctx.NewString("")
			}
			return qctx.NewString(string(configJSON))
		}))

	// __go_registry_has(category, name) → "true" or "false"
	qctx.Globals().Set("__go_registry_has",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 2 {
				return qctx.NewString("false")
			}
			category := args[0].String()
			name := args[1].String()
			var found bool
			switch category {
			case "provider":
				found = k.providers.HasAIProvider(name)
			case "vectorStore":
				found = k.providers.HasVectorStore(name)
			case "storage":
				found = k.providers.HasStorage(name)
			}
			if found {
				return qctx.NewString("true")
			}
			return qctx.NewString("false")
		}))

	// __go_registry_list(category) → JSON array
	qctx.Globals().Set("__go_registry_list",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.NewString("[]")
			}
			category := args[0].String()
			var result any
			switch category {
			case "provider":
				result = k.providers.ListAIProviders()
			case "vectorStore":
				result = k.providers.ListVectorStores()
			case "storage":
				result = k.providers.ListStorages()
			default:
				result = []any{}
			}
			b, _ := json.Marshal(result)
			return qctx.NewString(string(b))
		}))

	// __go_brainkit_await_approval(approvalTopic, payloadJSON, timeoutMs) → Promise<responseJSON>
	// Publishes an approval request to approvalTopic with auto-generated replyTo.
	// Subscribes to replyTo and waits for a response with context.WithTimeout.
	// Returns the response payload JSON. On timeout, returns {"approved":false,"reason":"timeout"}.
	// All bus lifecycle (publish, subscribe, wait, cleanup) happens in Go — no JS closures,
	// no setTimeout, no GC risk during the wait.
	qctx.Globals().Set("__go_brainkit_await_approval",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 3 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "await_approval: expected 3 args"})
			}
			approvalTopic := args[0].String()
			payload := json.RawMessage(args[1].String())
			timeoutMs := args[2].ToInt64()
			if timeoutMs <= 0 {
				timeoutMs = 30000
			}

			return qctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
				k.bridge.Go(func(goCtx context.Context) {
					timeout := time.Duration(timeoutMs) * time.Millisecond
					waitCtx, waitCancel := context.WithTimeout(goCtx, timeout)
					defer waitCancel()

					// Generate replyTo
					correlationID := uuid.NewString()
					replyTo := approvalTopic + ".reply." + correlationID

					// Subscribe BEFORE publishing (avoid race)
					replyCh := make(chan messages.Message, 1)
					unsub, subErr := k.remote.SubscribeRaw(waitCtx, replyTo, func(msg messages.Message) {
						select {
						case replyCh <- msg:
						default:
						}
					})
					if subErr != nil {
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("await_approval: subscribe: %w", subErr))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}
					defer unsub()

					// Publish approval request with replyTo
					pubCtx := messaging.WithPublishMeta(waitCtx, correlationID, replyTo)
					if _, pubErr := k.remote.PublishRaw(pubCtx, approvalTopic, payload); pubErr != nil {
						qctx.Schedule(func(qctx *quickjs.Context) {
							errVal := qctx.NewError(fmt.Errorf("await_approval: publish: %w", pubErr))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}

					// Wait for response or timeout
					select {
					case msg := <-replyCh:
						responseJSON := string(msg.Payload)
						qctx.Schedule(func(qctx *quickjs.Context) {
							resolve(qctx.NewString(responseJSON))
						})
					case <-waitCtx.Done():
						// Timeout — return timeout indicator so JS can auto-decline
						timeoutJSON := `{"approved":false,"reason":"timeout"}`
						qctx.Schedule(func(qctx *quickjs.Context) {
							resolve(qctx.NewString(timeoutJSON))
						})
					}
				})
			})
		}))

	// __go_brainkit_bus_schedule(expression, topic, payloadJSON, source) → scheduleID
	qctx.Globals().Set("__go_brainkit_bus_schedule",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 4 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "args", Message: "bus.schedule: expected 4 args"})
			}
			expression := args[0].String()
			topic := args[1].String()
			payload := json.RawMessage(args[2].String())
			source := args[3].String()

			id, err := k.Schedule(context.Background(), ScheduleConfig{
				Expression: expression,
				Topic:      topic,
				Payload:    payload,
				Source:     source,
			})
			if err != nil {
				return k.throwBrainkitError(qctx, err)
			}
			return qctx.NewString(id)
		}))

	// __go_brainkit_bus_unschedule(scheduleID)
	qctx.Globals().Set("__go_brainkit_bus_unschedule",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "scheduleId", Message: "bus.unschedule: expected 1 arg"})
			}
			k.Unschedule(context.Background(), args[0].String())
			return qctx.NewUndefined()
		}))

	// __go_brainkit_secret_get(name) → value or "" (not found)
	// Throws on: no args, store nil, store error. Returns "" only for "not found".
	qctx.Globals().Set("__go_brainkit_secret_get",
		qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return k.throwBrainkitError(qctx, &sdkerrors.ValidationError{Field: "name", Message: "is required"})
			}
			name := args[0].String()

			// RBAC enforcement — must have secrets.get permission
			if err := k.checkCommandPermission(k.currentDeploymentSource(), "secrets.get"); err != nil {
				return k.throwBrainkitError(qctx, err)
			}

			if k.secretStore == nil {
				return k.throwBrainkitError(qctx, &sdkerrors.NotConfiguredError{Feature: "secrets"})
			}
			val, err := k.secretStore.Get(context.Background(), name)
			if err != nil {
				return k.throwBrainkitError(qctx, &sdkerrors.BridgeError{Function: "secret_get", Cause: err})
			}
			if val == "" {
				return qctx.NewString("") // legitimate "not found"
			}
			// Audit: emit secrets.accessed event
			source := k.currentDeploymentSource()
			if source == "" {
				source = k.callerID
			}
			k.emitSecretEvent(context.Background(), messages.SecretsAccessedEvent{
				Name:      name,
				Accessor:  source,
				Timestamp: time.Now().Format(time.RFC3339),
			})
			return qctx.NewString(val)
		}))

	// Set context globals
	qctx.Globals().Set("__brainkit_sandbox_id", qctx.NewString(k.agents.ID()))
	qctx.Globals().Set("__brainkit_sandbox_namespace", qctx.NewString(k.namespace))
	qctx.Globals().Set("__brainkit_sandbox_callerID", qctx.NewString(k.callerID))
}

// throwBrainkitError constructs a structured JS error with .code and .details properties.
// If err implements BrainkitError, code and details are populated from it.
// Otherwise, code defaults to "INTERNAL_ERROR" with the error message.
func (k *Kernel) throwBrainkitError(qctx *quickjs.Context, err error) *quickjs.Value {
	var bkErr sdkerrors.BrainkitError
	code := "INTERNAL_ERROR"
	detailsJSON := "{}"
	msg := err.Error()

	if errors.As(err, &bkErr) {
		code = bkErr.Code()
		if d := bkErr.Details(); d != nil {
			if b, e := json.Marshal(d); e == nil {
				detailsJSON = string(b)
			}
		}
	}

	script := fmt.Sprintf(`(function() {
		var e = new Error(%q);
		e.code = %q;
		e.details = JSON.parse(%q);
		return e;
	})()`, msg, code, detailsJSON)

	errVal := qctx.Eval(script)
	if errVal.IsException() {
		// Fallback if JS construction fails
		return qctx.ThrowError(err)
	}
	return qctx.Throw(errVal)
}

// redactCredentials strips sensitive fields (API keys, tokens, passwords, secrets)
// from a config struct before returning it to JS. Marshal → strip → unmarshal.
func redactCredentials(config any) any {
	raw, err := json.Marshal(config)
	if err != nil {
		return config
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return config
	}
	sensitiveKeys := map[string]bool{
		"APIKey": true, "apiKey": true, "api_key": true,
		"AuthToken": true, "authToken": true, "auth_token": true,
		"AccessKey": true, "accessKey": true, "access_key": true,
		"SecretKey": true, "secretKey": true, "secret_key": true,
		"Password": true, "password": true,
		"Token": true, "token": true,
		"AdminKey": true, "adminKey": true,
	}
	for k := range m {
		if sensitiveKeys[k] {
			if s, ok := m[k].(string); ok && len(s) > 0 {
				m[k] = "****"
			}
		}
	}
	return m
}
