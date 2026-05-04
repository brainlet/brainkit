package wm

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	core "github.com/brainlet/brainkit/internal/transport"
)

// Publisher adapts a Watermill publisher to Brainkit's transport publisher.
type Publisher struct {
	Inner message.Publisher
}

func (p Publisher) Publish(topic string, messages ...*core.Message) error {
	wmsgs := make([]*message.Message, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		wmsgs = append(wmsgs, ToWatermillMessage(msg))
	}
	return p.Inner.Publish(topic, wmsgs...)
}

// Subscriber adapts a Watermill subscriber to Brainkit's transport subscriber.
type Subscriber struct {
	Inner message.Subscriber
}

func (s Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *core.Message, error) {
	in, err := s.Inner.Subscribe(ctx, topic)
	if err != nil {
		return nil, err
	}
	out := make(chan *core.Message)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- FromWatermillMessage(msg):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func ToWatermillMessage(msg *core.Message) *message.Message {
	wmsg := message.NewMessage(msg.UUID, append([]byte(nil), msg.Payload...))
	wmsg.SetContext(msg.Context())
	for key, value := range msg.Metadata {
		wmsg.Metadata.Set(key, value)
	}
	return wmsg
}

func FromWatermillMessage(msg *message.Message) *core.Message {
	metadata := make(map[string]string, len(msg.Metadata))
	for key, value := range msg.Metadata {
		metadata[key] = value
	}
	return core.NewReceivedMessage(
		msg.UUID,
		append([]byte(nil), msg.Payload...),
		metadata,
		msg.Context(),
		msg.Ack,
		msg.Nack,
	)
}
