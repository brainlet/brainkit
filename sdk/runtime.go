package sdk

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk/messages"
)

// Runtime is the unified interface for interacting with a brainkit runtime.
// Kernel, Node, and plugin Client all implement this.
type Runtime interface {
	// PublishRaw sends a message to a topic.
	// Generates a correlationID (UUID), stamps it in the Watermill message metadata
	// as "correlationId", and returns it. If the context already carries a correlationID
	// (via WithCorrelationID), that value is used instead of generating a new one.
	PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (correlationID string, err error)

	// SubscribeRaw subscribes to a topic.
	// The subscription MUST be active and ready to receive messages before this method returns.
	// This is a contract, not an implementation detail — PublishAwait depends on it to avoid
	// race conditions where a publish lands before the subscriber is listening.
	// Handler receives the full Message including payload and metadata (correlationID, callerID).
	// Returns a cancel function to unsubscribe.
	SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (cancel func(), err error)

	// Close shuts down the runtime and releases all resources.
	Close() error
}

// CrossNamespaceRuntime is an optional interface for Runtimes that support cross-Kit operations.
// Kernel and Node implement this. Plugin clients do not (they talk to their host Kit only).
type CrossNamespaceRuntime interface {
	Runtime
	// PublishRawTo publishes to a specific Kit's namespace, bypassing the local namespace.
	PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (correlationID string, err error)
	// SubscribeRawTo subscribes to a topic in a specific Kit's namespace.
	SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(messages.Message)) (cancel func(), err error)
}
