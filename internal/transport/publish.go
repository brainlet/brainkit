package transport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/sdk"
)

// Publish stamps internal metadata, serializes the typed message, and sends it to the transport.
func Publish[T sdk.BrainkitMessage](pub message.Publisher, msg T, callerID string) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %T: %w", msg, err)
	}
	wmsg := message.NewMessage(watermill.NewUUID(), payload)
	wmsg.Metadata.Set("callerId", callerID)
	return pub.Publish(msg.BusTopic(), wmsg)
}

// Handle registers a typed handler on the message's topic.
// When a message arrives on the topic, it's decoded into Envelope[T] and the handler is called.
// Return nil to Ack (message processed), return error to Nack (retry/DLQ depending on transport).
func Handle[T sdk.BrainkitMessage](
	router *message.Router,
	sub message.Subscriber,
	fn func(context.Context, Envelope[T]) error,
) {
	var zero T
	topic := zero.BusTopic()

	router.AddConsumerHandler(
		topic+"_np_handler",
		topic,
		sub,
		func(wmsg *message.Message) error {
			env, err := DecodeEnvelope[T](wmsg)
			if err != nil {
				return err
			}
			// Stamp subscription topic for middleware metrics
			wmsg.Metadata.Set("_subscription_topic", topic)
			return fn(context.Background(), env)
		},
	)
}
