package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// --- sdk.Runtime implementation ---

// PublishRaw sends a message to a topic. Returns correlationID.
func (k *Kernel) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return k.remote.PublishRaw(ctx, topic, payload)
}

// SubscribeRaw subscribes to a topic. Subscription is active before this returns.
func (k *Kernel) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	return k.remote.SubscribeRaw(ctx, topic, handler)
}

// --- sdk.CrossNamespaceRuntime implementation ---

// persistenceError routes a persistence failure through the ErrorHandler and emits a bus event.
// The original operation still succeeds in memory — persistence is best-effort.
func (k *Kernel) persistenceError(ctx context.Context, operation, source string, err error) {
	typedErr := &sdkerrors.PersistenceError{Operation: operation, Source: source, Cause: err}
	types.InvokeErrorHandler(k.config.ErrorHandler, typedErr, types.ErrorContext{
		Operation: operation, Component: "persistence", Source: source,
	})
	payload, _ := json.Marshal(map[string]any{
		"operation": operation,
		"source":    source,
		"error":     err.Error(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
	k.publish(ctx, "kit.persistence.error", payload)
}

// PublishRawTo publishes to a specific Kit's namespace.
func (k *Kernel) PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return k.remote.PublishRawToNamespace(ctx, targetNamespace, topic, payload)
}

// SubscribeRawTo subscribes to a topic in a specific Kit's namespace.
func (k *Kernel) SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(sdk.Message)) (func(), error) {
	return k.remote.SubscribeRawToNamespace(ctx, targetNamespace, topic, handler)
}

// publish is an internal convenience for fire-and-forget event publishing.
func (k *Kernel) publish(ctx context.Context, topic string, payload json.RawMessage) error {
	_, err := k.remote.PublishRaw(ctx, topic, payload)
	return err
}

// subscribe is an internal convenience for subscribing with full message.
func (k *Kernel) subscribe(topic string, handler func(sdk.Message)) (func(), error) {
	return k.remote.SubscribeRaw(context.Background(), topic, handler)
}

// callJS invokes a named function in the JS runtime with JSON-serialized arguments.
// The function must be registered on globalThis (e.g., __brainkit.workflow.start).
// Returns the JSON result. Used by bus command handlers to avoid inline JS construction.
func (k *Kernel) callJS(ctx context.Context, fn string, args any) (json.RawMessage, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("callJS %s: marshal args: %w", fn, err)
	}
	code := fmt.Sprintf("return JSON.stringify(await %s(JSON.parse(%q)))", fn, string(argsJSON))
	result, err := k.EvalTS(ctx, "__dispatch__.ts", code)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(result), nil
}

// callJSSync invokes a named function synchronously via bridge.Eval (not EvalTS).
// Used for non-async operations that run on the JS thread directly (e.g., provider cache refresh).
func (k *Kernel) callJSSync(fn string, args any) {
	argsJSON, _ := json.Marshal(args)
	k.bridge.Eval("__dispatch_sync__.js", fmt.Sprintf("%s(JSON.parse(%q))", fn, string(argsJSON)))
}

// ReplyRaw publishes directly to a resolved replyTo topic without namespace prefixing.
// This is the Go equivalent of __go_brainkit_bus_reply in bridges.go.
// Used by sdk.Reply and sdk.SendChunk.
func (k *Kernel) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	if replyTo == "" {
		return nil
	}
	wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
	wmsg.Metadata.Set("correlationId", correlationID)
	if done {
		wmsg.Metadata.Set("done", "true")
	}
	// replyTo is already namespaced+sanitized — publish directly to transport
	return k.transport.Publisher.Publish(replyTo, wmsg)
}

// replyEnvelope publishes a terminal envelope reply. Stamps envelope=true
// metadata so the Caller decodes the payload via sdk.FromEnvelope.
func (k *Kernel) replyEnvelope(replyTo, correlationID string, payload []byte) error {
	if replyTo == "" {
		return nil
	}
	wmsg := message.NewMessage(watermill.NewUUID(), payload)
	wmsg.Metadata.Set("correlationId", correlationID)
	wmsg.Metadata.Set("done", "true")
	wmsg.Metadata.Set("envelope", "true")
	return k.transport.Publisher.Publish(replyTo, wmsg)
}
