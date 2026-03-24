package messaging

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"

	wmamqp "github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	wmnats "github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	wmredis "github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	wmsql "github.com/ThreeDotsLabs/watermill-sql/v4/pkg/sql"
	wmsqlite "github.com/ThreeDotsLabs/watermill-sqlite/wmsqlitemodernc"

	_ "github.com/lib/pq"          // Postgres driver
	_ "modernc.org/sqlite"          // SQLite driver (already in go.mod)
	"github.com/redis/go-redis/v9"
)

// TransportConfig configures the transport backend.
type TransportConfig struct {
	Type     string // "memory" (default), "nats", "amqp", "redis", "sql-postgres", "sql-sqlite"

	// NATS
	NATSURL  string
	NATSName string

	// AMQP (RabbitMQ)
	AMQPURL string // e.g. "amqp://guest:guest@localhost:5672/"

	// Redis Streams
	RedisURL string // e.g. "redis://localhost:6379/0"

	// SQL
	PostgresURL string // e.g. "postgres://user:pass@localhost:5432/brainkit?sslmode=disable"
	SQLitePath  string // e.g. "/tmp/brainkit-bus.db" or ":memory:"
}

// Transport bundles the concrete publisher/subscriber pair plus a shared closer.
type Transport struct {
	Publisher  message.Publisher
	Subscriber message.Subscriber
	closeFns   []func() error

	// TopicSanitizer transforms logical topic names into transport-safe names.
	// Applied automatically by RemoteClient and Host.
	TopicSanitizer func(string) string
}

// SanitizeTopic applies the transport's topic sanitizer if set.
func (t *Transport) SanitizeTopic(topic string) string {
	if t == nil || t.TopicSanitizer == nil {
		return topic
	}
	return t.TopicSanitizer(topic)
}

// Close shuts down all transport resources.
func (t *Transport) Close() error {
	if t == nil {
		return nil
	}
	var firstErr error
	for _, closeFn := range t.closeFns {
		if closeFn == nil {
			continue
		}
		if err := closeFn(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NewTransportSet creates a fully managed transport bundle.
func NewTransportSet(cfg TransportConfig) (*Transport, error) {
	logger := watermill.NopLogger{}

	switch cfg.Type {
	case "", "memory":
		pubSub := gochannel.NewGoChannel(gochannel.Config{
			Persistent: true,
		}, logger)
		return &Transport{
			Publisher:  pubSub,
			Subscriber: pubSub,
			closeFns:   []func() error{pubSub.Close},
		}, nil

	case "nats":
		return newNATSTransport(cfg, logger)

	case "amqp":
		return newAMQPTransport(cfg, logger)

	case "redis":
		return newRedisTransport(cfg, logger)

	case "sql-postgres":
		return newPostgresTransport(cfg, logger)

	case "sql-sqlite":
		return newSQLiteTransport(cfg, logger)

	default:
		return nil, fmt.Errorf("unknown transport type: %q (supported: memory, nats, amqp, redis, sql-postgres, sql-sqlite)", cfg.Type)
	}
}

// ---------------------------------------------------------------------------
// NATS JetStream
// ---------------------------------------------------------------------------

func newNATSTransport(cfg TransportConfig, logger watermill.LoggerAdapter) (*Transport, error) {
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

	durablePrefix := cfg.NATSName
	if durablePrefix == "" {
		durablePrefix = "brainkit"
	}
	durablePrefix = sanitizeDurable(durablePrefix)

	publisher, err := wmnats.NewPublisher(wmnats.PublisherConfig{
		URL:               url,
		Marshaler:         wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream:         wmnats.JetStreamConfig{AutoProvision: true, TrackMsgId: true},
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("nats publisher: %w", err)
	}

	subscriber, err := wmnats.NewSubscriber(wmnats.SubscriberConfig{
		URL:               url,
		QueueGroupPrefix:  durablePrefix,
		SubscribersCount:  1,
		CloseTimeout:      15 * time.Second,
		AckWaitTimeout:    30 * time.Second,
		SubscribeTimeout:  30 * time.Second,
		Unmarshaler:       wmnats.JSONMarshaler{},
		SubjectCalculator: natsSubjectCalc,
		JetStream: wmnats.JetStreamConfig{
			AutoProvision: true,
			DurablePrefix: durablePrefix,
			TrackMsgId:    true,
		},
	}, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("nats subscriber: %w", err)
	}

	return &Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
		closeFns:   []func() error{publisher.Close, subscriber.Close},
		TopicSanitizer: func(topic string) string {
			r := strings.NewReplacer(".", "-", "/", "-", "@", "-", " ", "-")
			return r.Replace(topic)
		},
	}, nil
}

// ---------------------------------------------------------------------------
// AMQP (RabbitMQ)
// ---------------------------------------------------------------------------

func newAMQPTransport(cfg TransportConfig, logger watermill.LoggerAdapter) (*Transport, error) {
	amqpURL := cfg.AMQPURL
	if amqpURL == "" {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	amqpConfig := wmamqp.NewDurablePubSubConfig(amqpURL, wmamqp.GenerateQueueNameTopicName)

	publisher, err := wmamqp.NewPublisher(amqpConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("amqp publisher: %w", err)
	}

	subscriber, err := wmamqp.NewSubscriber(amqpConfig, logger)
	if err != nil {
		_ = publisher.Close()
		return nil, fmt.Errorf("amqp subscriber: %w", err)
	}

	return &Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
		closeFns:   []func() error{publisher.Close, subscriber.Close},
		// AMQP: dots are native routing key delimiters — preserve them.
		// Sanitize slashes, @, spaces which are invalid in exchange names.
		TopicSanitizer: func(topic string) string {
			r := strings.NewReplacer("/", "-", "@", "-", " ", "-")
			return r.Replace(topic)
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Redis Streams
// ---------------------------------------------------------------------------

func newRedisTransport(cfg TransportConfig, logger watermill.LoggerAdapter) (*Transport, error) {
	redisURL := cfg.RedisURL
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("redis url: %w", err)
	}
	rdb := redis.NewClient(redisOpts)

	publisher, err := wmredis.NewPublisher(wmredis.PublisherConfig{
		Client:     rdb,
		Marshaller: wmredis.DefaultMarshallerUnmarshaller{},
	}, logger)
	if err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis publisher: %w", err)
	}

	consumerGroup := "brainkit"
	if cfg.NATSName != "" {
		consumerGroup = sanitizeDurable(cfg.NATSName)
	}

	subscriber, err := wmredis.NewSubscriber(wmredis.SubscriberConfig{
		Client:        rdb,
		Unmarshaller:  wmredis.DefaultMarshallerUnmarshaller{},
		ConsumerGroup: consumerGroup,
	}, logger)
	if err != nil {
		_ = publisher.Close()
		rdb.Close()
		return nil, fmt.Errorf("redis subscriber: %w", err)
	}

	return &Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
		closeFns:   []func() error{publisher.Close, subscriber.Close, rdb.Close},
		// Redis keys accept any binary string — no sanitization needed
	}, nil
}

// ---------------------------------------------------------------------------
// PostgreSQL
// ---------------------------------------------------------------------------

func newPostgresTransport(cfg TransportConfig, logger watermill.LoggerAdapter) (*Transport, error) {
	pgURL := cfg.PostgresURL
	if pgURL == "" {
		pgURL = "postgres://localhost:5432/brainkit?sslmode=disable"
	}

	db, err := sql.Open("postgres", pgURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}

	dbWrapped := wmsql.BeginnerFromStdSQL(db)

	publisher, err := wmsql.NewPublisher(dbWrapped, wmsql.PublisherConfig{
		SchemaAdapter:        wmsql.DefaultPostgreSQLSchema{},
		AutoInitializeSchema: true,
	}, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("postgres publisher: %w", err)
	}

	subscriber, err := wmsql.NewSubscriber(dbWrapped, wmsql.SubscriberConfig{
		SchemaAdapter:    wmsql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   wmsql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
		PollInterval:     100 * time.Millisecond,
	}, logger)
	if err != nil {
		_ = publisher.Close()
		db.Close()
		return nil, fmt.Errorf("postgres subscriber: %w", err)
	}

	return &Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
		closeFns:   []func() error{publisher.Close, subscriber.Close, db.Close},
		// SQL: topic becomes table name — must be valid SQL identifier
		TopicSanitizer: func(topic string) string {
			r := strings.NewReplacer(".", "_", "/", "_", "@", "_", " ", "_")
			return r.Replace(topic)
		},
	}, nil
}

// ---------------------------------------------------------------------------
// SQLite (using official watermill-sqlite/wmsqlitemodernc)
// ---------------------------------------------------------------------------

func newSQLiteTransport(cfg TransportConfig, logger watermill.LoggerAdapter) (*Transport, error) {
	dbPath := cfg.SQLitePath
	if dbPath == "" {
		dbPath = "file::memory:?cache=shared"
	}
	if dbPath != "file::memory:?cache=shared" && !strings.HasPrefix(dbPath, "file:") {
		dbPath = "file:" + dbPath + "?cache=shared"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sqlite connect: %w", err)
	}

	publisher, err := wmsqlite.NewPublisher(db, wmsqlite.PublisherOptions{
		InitializeSchema: true,
		Logger:           logger,
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite publisher: %w", err)
	}

	subscriber, err := wmsqlite.NewSubscriber(db, wmsqlite.SubscriberOptions{
		InitializeSchema: true,
		Logger:           logger,
		PollInterval:     100 * time.Millisecond,
	})
	if err != nil {
		_ = publisher.Close()
		db.Close()
		return nil, fmt.Errorf("sqlite subscriber: %w", err)
	}

	return &Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
		closeFns:   []func() error{publisher.Close, subscriber.Close, db.Close},
		// SQLite watermill adapter handles table naming internally.
		// But our topics may contain dots which become table names — sanitize.
		TopicSanitizer: func(topic string) string {
			r := strings.NewReplacer(".", "_", "/", "_", "@", "_", " ", "_")
			return r.Replace(topic)
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// NewTransport preserves the old pub/sub factory signature for tests and helpers.
func NewTransport(cfg TransportConfig) (message.Publisher, message.Subscriber, error) {
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
