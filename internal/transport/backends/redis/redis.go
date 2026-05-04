package redis

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	wmredis "github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	core "github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/transport/backends/backutil"
	"github.com/brainlet/brainkit/internal/transport/backends/wm"
	"github.com/redis/go-redis/v9"
)

// NewTransport creates a Redis Streams transport.
func NewTransport(cfg core.TransportConfig) (*core.Transport, error) {
	redisURL := cfg.RedisURL
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	newClient := func() *redis.Client {
		opts := *redisOpts
		return redis.NewClient(&opts)
	}

	logger := watermill.NopLogger{}
	publisherClient := newClient()
	publisher, err := wmredis.NewPublisher(wmredis.PublisherConfig{
		Client:     publisherClient,
		Marshaller: wmredis.DefaultMarshallerUnmarshaller{},
	}, logger)
	if err != nil {
		_ = publisherClient.Close()
		return nil, fmt.Errorf("redis publisher: %w", err)
	}

	consumerGroup := cfg.Namespace
	if consumerGroup == "" {
		consumerGroup = "brainkit"
	}
	consumerGroup = backutil.SanitizeDurable(consumerGroup)

	subscriberClient := newClient()
	subscriber, err := wmredis.NewSubscriber(wmredis.SubscriberConfig{
		Client:        subscriberClient,
		Unmarshaller:  wmredis.DefaultMarshallerUnmarshaller{},
		ConsumerGroup: consumerGroup,
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriberClient.Close()
		return nil, fmt.Errorf("redis subscriber: %w", err)
	}

	fanOutGroup := consumerGroup + "-fo-" + watermill.NewShortUUID()
	fanOutClient := newClient()
	fanOutSub, err := wmredis.NewSubscriber(wmredis.SubscriberConfig{
		Client:        fanOutClient,
		Unmarshaller:  wmredis.DefaultMarshallerUnmarshaller{},
		ConsumerGroup: fanOutGroup,
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriber.Close()
		_ = fanOutClient.Close()
		return nil, fmt.Errorf("redis fan-out subscriber: %w", err)
	}

	return core.NewManagedTransport(
		"redis",
		wm.Publisher{Inner: publisher},
		wm.Subscriber{Inner: subscriber},
		wm.Subscriber{Inner: fanOutSub},
		nil,
		core.OnceCloser(publisher.Close),
		core.OnceCloser(subscriber.Close),
		core.OnceCloser(fanOutSub.Close),
	), nil
}
