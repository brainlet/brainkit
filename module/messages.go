package module

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk"
)

// MessageHost publishes events and owns scoped bus subscriptions.
type MessageHost interface {
	PublishRaw(context.Context, string, json.RawMessage) (string, error)
	SubscribeRaw(context.Context, string, func(sdk.Message)) (Handle, error)
}
