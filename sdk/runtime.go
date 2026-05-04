package sdk

import (
	"context"
	"encoding/json"
)

// Runtime is the low-level transport interface for interacting with a brainkit
// runtime. Kernel, Node, and plugin Client all implement this.
//
// Normal request/reply code should prefer Call, CallStream, or generated
// CallXxx helpers. Runtime exists for protocol bridges, events, custom
// subscriptions, diagnostics, and tests that intentionally inspect wire
// messages.
type Runtime interface {
	// PublishRaw sends a raw message to a topic.
	//
	// This is a transport primitive, not the normal request/reply path. Use
	// Call or generated CallXxx helpers when a response is expected.
	// Generates a correlationID (UUID), stamps it in the message metadata
	// as "correlationId", and returns it. If the context already carries a correlationID
	// (via WithCorrelationID), that value is used instead of generating a new one.
	PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (correlationID string, err error)

	// SubscribeRaw subscribes to a raw topic.
	//
	// This is for event/protocol surfaces and diagnostics. Normal
	// request/reply callers should not create reply-topic subscriptions; the
	// shared Caller used by Call and CallStream owns that routing.
	// The subscription MUST be active and ready to receive messages before this method returns.
	// This is a contract, not an implementation detail: sdk/protocol.Publish
	// and SubscribeTo depend on it to avoid race conditions where a publish
	// lands before the subscriber is listening.
	// Handler receives the full Message including payload and metadata (correlationID, callerID).
	// Returns a cancel function to unsubscribe.
	SubscribeRaw(ctx context.Context, topic string, handler func(Message)) (cancel func(), err error)

	// Close shuts down the runtime and releases all resources.
	Close() error
}

// CrossNamespaceRuntime is an optional interface for Runtimes that support cross-Kit operations.
// Kernel and Node implement this. Plugin clients do not (they talk to their host Kit only).
type CrossNamespaceRuntime interface {
	Runtime
	// PublishRawTo publishes a raw message to a specific Kit's namespace,
	// bypassing the local namespace. Use Call with WithCallTo for normal
	// cross-namespace request/reply.
	PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (correlationID string, err error)
	// SubscribeRawTo subscribes to a raw topic in a specific Kit's namespace.
	SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(Message)) (cancel func(), err error)
}
