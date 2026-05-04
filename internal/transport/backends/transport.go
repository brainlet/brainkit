package backends

import (
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"

	wmamqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	wmnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	wmredis "github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	core "github.com/brainlet/brainkit/internal/transport"
	"github.com/nats-io/nats.go"

	"github.com/redis/go-redis/v9"
)

// NewTransportSet creates a fully managed transport bundle.
func NewTransportSet(cfg core.TransportConfig) (*core.Transport, error) {
	logger := watermill.NopLogger{}

	var (
		t    *core.Transport
		err  error
		kind string
	)
	switch cfg.Type {
	case "memory":
		t, err = core.NewTransportSet(cfg)
		kind = "memory"

	case "", "embedded":
		t, err = newEmbeddedNATSTransport(cfg, logger)
		kind = "embedded"

	case "nats":
		t, err = newNATSTransport(cfg, logger)
		kind = "nats"

	case "amqp":
		t, err = newAMQPTransport(cfg, logger)
		kind = "amqp"

	case "redis":
		t, err = newRedisTransport(cfg, logger)
		kind = "redis"

	default:
		return nil, fmt.Errorf("unknown transport type: %q (supported: memory, embedded, nats, amqp, redis)", cfg.Type)
	}
	if err != nil {
		return nil, err
	}
	t.Kind = kind
	return t, nil
}

// ---------------------------------------------------------------------------
// NATS JetStream
// ---------------------------------------------------------------------------

func newNATSTransport(cfg core.TransportConfig, logger watermill.LoggerAdapter) (*core.Transport, error) {
	url := cfg.NATSURL
	if url == "" {
		url = "nats://127.0.0.1:4222"
	}

	natsSubjectCalc := func(queueGroupPrefix, topic string) *wmnats.SubjectDetail {
		safeTopic := strings.ReplaceAll(topic, ".", "-")
		qg := ""
		if queueGroupPrefix != "" {
			qg = queueGroupPrefix
		}
		return &wmnats.SubjectDetail{Primary: safeTopic, QueueGroup: qg}
	}

	// Consumer group = namespace. Replicas with the same namespace compete.
	consumerGroup := cfg.Namespace
	if consumerGroup == "" {
		consumerGroup = "brainkit"
	}
	consumerGroup = sanitizeDurable(consumerGroup)

	publisher, err := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               url,
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("nats publisher: %w", err)
	}

	// `nats.DeliverNew()` is critical: watermill-nats deletes the
	// JetStream consumer on Subscriber.Close (sub.Unsubscribe in
	// pkg/nats/subscriber.go). On the next process start the fresh
	// consumer would inherit the default `DeliverPolicy: DeliverAll`
	// and replay every persisted message in the stream — including
	// every previous send that already ran. DeliverNew anchors the
	// consumer to the head of the stream at subscribe time, so
	// re-attaching after restart only sees messages published from
	// that point forward.
	//
	// This pairs with the LimitsPolicy retention watermill provisions
	// streams with: the messages stay on disk for inspection /
	// debugging, but no live subscriber sees them as fresh deliveries.
	deliverNew := []nats.SubOpt{nats.DeliverNew()}

	// Command subscriber — consumer group for competing consumers
	subscriber, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               url,
		QueueGroupPrefix:  consumerGroup,
		SubscribersCount:  1,
		CloseTimeout:      15 * time.Second,
		AckWaitTimeout:    30 * time.Second,
		SubscribeTimeout:  30 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision:    true,
			DurablePrefix:    consumerGroup,
			TrackMsgId:       true,
			SubscribeOptions: deliverNew,
		},
	}, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("nats subscriber: %w", err)
	}

	// Fan-out subscriber — unique durable per instance, no queue group.
	// Every instance with this subscriber receives ALL messages (broadcast).
	fanOutID := consumerGroup + "-fo-" + watermill.NewShortUUID()
	fanOutSub, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               url,
		QueueGroupPrefix:  "", // no queue group = fan-out
		SubscribersCount:  1,
		CloseTimeout:      15 * time.Second,
		AckWaitTimeout:    30 * time.Second,
		SubscribeTimeout:  30 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision:    true,
			DurablePrefix:    fanOutID,
			TrackMsgId:       true,
			SubscribeOptions: deliverNew,
		},
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriber.Close()
		return nil, fmt.Errorf("nats fan-out subscriber: %w", err)
	}

	return core.NewManagedTransport(
		"nats",
		watermillPublisher{inner: publisher},
		watermillSubscriber{inner: subscriber},
		watermillSubscriber{inner: fanOutSub},
		func(topic string) string {
			r := strings.NewReplacer(".", "-", "/", "-", "@", "-", " ", "-")
			return r.Replace(topic)
		},
		publisher.Close,
		core.OnceCloser(subscriber.Close),
		core.OnceCloser(fanOutSub.Close),
	), nil
}

// ---------------------------------------------------------------------------
// AMQP (RabbitMQ)
// ---------------------------------------------------------------------------

func newAMQPTransport(cfg core.TransportConfig, logger watermill.LoggerAdapter) (*core.Transport, error) {
	amqpURL := cfg.AMQPURL
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	// Command subscriber — shared queue name for competing consumers
	consumerGroup := cfg.Namespace
	if consumerGroup == "" {
		consumerGroup = "brainkit"
	}
	amqpConfig := wmamqp.NewDurablePubSubConfig(amqpURL, func(topic string) string {
		return consumerGroup + "_" + topic
	})

	publisher, err := wmamqp.NewPublisher(amqpConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("amqp publisher: %w", err)
	}

	subscriber, err := wmamqp.NewSubscriber(amqpConfig, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("amqp subscriber: %w", err)
	}

	// Fan-out subscriber — unique queue per instance
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
		watermillPublisher{inner: publisher},
		watermillSubscriber{inner: subscriber},
		watermillSubscriber{inner: fanOutSub},
		func(topic string) string {
			r := strings.NewReplacer("/", "-", "@", "-", " ", "-")
			return r.Replace(topic)
		},
		publisher.Close,
		core.OnceCloser(subscriber.Close),
		core.OnceCloser(fanOutSub.Close),
	), nil
}

// ---------------------------------------------------------------------------
// Redis Streams
// ---------------------------------------------------------------------------

func newRedisTransport(cfg core.TransportConfig, logger watermill.LoggerAdapter) (*core.Transport, error) {
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
	consumerGroup = sanitizeDurable(consumerGroup)

	// Command subscriber — consumer group for competing consumers
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

	// Fan-out subscriber — unique consumer group per instance
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
		watermillPublisher{inner: publisher},
		watermillSubscriber{inner: subscriber},
		watermillSubscriber{inner: fanOutSub},
		nil,
		core.OnceCloser(publisher.Close),
		core.OnceCloser(subscriber.Close),
		core.OnceCloser(fanOutSub.Close),
	), nil
}

// ---------------------------------------------------------------------------
// Embedded NATS (in-process server + Watermill NATS adapter)
// ---------------------------------------------------------------------------

func newEmbeddedNATSTransport(cfg core.TransportConfig, logger watermill.LoggerAdapter) (*core.Transport, error) {
	embedded, err := NewEmbeddedNATS(EmbeddedNATSConfig{
		StoreDir: cfg.NATSStoreDir,
	})
	if err != nil {
		return nil, fmt.Errorf("embedded nats: %w", err)
	}

	// Reuse the standard NATS transport factory with the embedded server's URL
	cfg.NATSURL = embedded.ClientURL()
	cfg.Type = "nats"

	transport, err := newNATSTransport(cfg, logger)
	if err != nil {
		embedded.Shutdown()
		return nil, err
	}

	transport.Kind = "embedded"
	transport.AppendClose(func() error {
		embedded.Shutdown()
		return nil
	})

	return transport, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// NewTransport preserves the old pub/sub factory signature for tests and helpers.
func NewTransport(cfg core.TransportConfig) (core.Publisher, core.Subscriber, error) {
	transport, err := NewTransportSet(cfg)
	if err != nil {
		return nil, nil, err
	}
	return transport.Publisher, transport.Subscriber, nil
}

// NamespacedTopic derives the concrete subject from a logical topic.
func NamespacedTopic(namespace, logicalTopic string) string {
	namespace = strings.TrimSpace(namespace)
	logicalTopic = strings.TrimSpace(logicalTopic)
	if namespace == "" {
		return logicalTopic
	}
	if logicalTopic == "" {
		return namespace
	}
	return namespace + "." + logicalTopic
}

func sanitizeDurable(value string) string {
	replacer := strings.NewReplacer(".", "_", "/", "_", "@", "_", "-", "_", " ", "_", ":", "_")
	return replacer.Replace(value)
}
