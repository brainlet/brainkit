package transport

import (
	"context"
	"encoding/json"
)

// Presence is the transport surface the discovery module needs: global
// (non-namespaced) publish + fan-out subscribe on a cluster-wide topic
// (`_brainkit.presence`). *RemoteClient satisfies it by structural match.
//
// Lives here — in internal/transport, next to the implementation — so both
// brainkit (for Kit.PresenceTransport()) and modules/discovery can import
// it without a cycle.
type Presence interface {
	PublishRawGlobal(ctx context.Context, topic string, payload json.RawMessage) error
	SubscribeRawFanOutGlobal(ctx context.Context, topic string, handler func(payload json.RawMessage)) (func(), error)
}
