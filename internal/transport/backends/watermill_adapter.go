package backends

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	core "github.com/brainlet/brainkit/internal/transport"
)

type watermillPublisher struct {
	inner message.Publisher
}

func (p watermillPublisher) Publish(topic string, messages ...*core.Message) error {
	wmsgs := make([]*message.Message, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		wmsgs = append(wmsgs, toWatermillMessage(msg))
	}
	return p.inner.Publish(topic, wmsgs...)
}

type watermillSubscriber struct {
	inner message.Subscriber
}

func (s watermillSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *core.Message, error) {
	in, err := s.inner.Subscribe(ctx, topic)
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
				case out <- fromWatermillMessage(msg):
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

func toWatermillMessage(msg *core.Message) *message.Message {
	wmsg := message.NewMessage(msg.UUID, append([]byte(nil), msg.Payload...))
	wmsg.SetContext(msg.Context())
	for key, value := range msg.Metadata {
		wmsg.Metadata.Set(key, value)
	}
	return wmsg
}

func fromWatermillMessage(msg *message.Message) *core.Message {
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
