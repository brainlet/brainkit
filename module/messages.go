package module

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk"
)

// MessageHost publishes low-level events/protocol messages and owns scoped bus
// subscriptions for modules. Normal module request/reply should use
// CapabilityRequestCaller plus generated CallXxxWithCaller helpers; raw reply
// topics are reserved for protocol bridges and handlers responding to an
// inbound message.
type MessageHost interface {
	PublishRaw(context.Context, string, json.RawMessage) (string, error)
	SubscribeRaw(context.Context, string, func(sdk.Message)) (Handle, error)
	ReplyRaw(context.Context, string, string, json.RawMessage, bool) error
}
