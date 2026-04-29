package module

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk"
)

// CommandSpec binds a logical command topic to a raw JSON handler.
type CommandSpec struct {
	Name   string
	Topic  string
	Handle func(context.Context, json.RawMessage) (json.RawMessage, error)
}

// Command builds a CommandSpec from a typed Brainkit message handler.
func Command[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, Req) (*Resp, error)) CommandSpec {
	var zero Req
	topic := zero.BusTopic()
	return CommandSpec{
		Name:  topic,
		Topic: topic,
		Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
			var req Req
			if len(payload) > 0 {
				if err := json.Unmarshal(payload, &req); err != nil {
					return nil, fmt.Errorf("decode %s: %w", topic, err)
				}
			}
			resp, err := handler(ctx, req)
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return json.RawMessage("null"), nil
			}
			return json.Marshal(resp)
		},
	}
}

// CommandHost mounts command handlers.
type CommandHost interface {
	Handle(CommandSpec) (Handle, error)
	Has(topic string) bool
}
