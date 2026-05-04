package amqp

import (
	"fmt"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	wmamqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	core "github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/transport/backends/wm"
)

// NewTransport creates an AMQP/RabbitMQ transport.
func NewTransport(cfg core.TransportConfig) (*core.Transport, error) {
	amqpURL := cfg.AMQPURL
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	consumerGroup := cfg.Namespace
	if consumerGroup == "" {
		consumerGroup = "brainkit"
	}
	amqpConfig := wmamqp.NewDurablePubSubConfig(amqpURL, func(topic string) string {
		return consumerGroup + "_" + topic
	})

	logger := watermill.NopLogger{}
	publisher, err := wmamqp.NewPublisher(amqpConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("amqp publisher: %w", err)
	}

	subscriber, err := wmamqp.NewSubscriber(amqpConfig, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("amqp subscriber: %w", err)
	}

	fanOutID := consumerGroup + "_fo_" + watermill.NewShortUUID()
	fanOutConfig := wmamqp.NewDurablePubSubConfig(amqpURL, func(topic string) string {
		return fanOutID + "_" + topic
	})
	fanOutSub, err := wmamqp.NewSubscriber(fanOutConfig, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriber.Close()
		return nil, fmt.Errorf("amqp fan-out subscriber: %w", err)
	}

	return core.NewManagedTransport(
		"amqp",
		wm.Publisher{Inner: publisher},
		wm.Subscriber{Inner: subscriber},
		wm.Subscriber{Inner: fanOutSub},
		func(topic string) string {
			r := strings.NewReplacer("/", "-", "@", "-", " ", "-")
			return r.Replace(topic)
		},
		publisher.Close,
		core.OnceCloser(subscriber.Close),
		core.OnceCloser(fanOutSub.Close),
	), nil
}
