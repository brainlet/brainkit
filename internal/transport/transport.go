package transport

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
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
	Type      string // "memory" (default), "nats", "amqp", "redis", "sql-postgres", "sql-sqlite"
	Namespace string // consumer group name; default: "brainkit". Replicas with same namespace compete.

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
	Publisher        message.Publisher
	Subscriber       message.Subscriber // consumer group = Namespace (competing consumers)
	FanOutSubscriber message.Subscriber // unique group per instance (all replicas receive)
	closeFns         []func() error

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

// onceCloser wraps a Close function with sync.Once to prevent double-close panics.
// Watermill's router.Close() fires handleClose goroutines that call subscriber.Close()
// asynchronously (not tracked by any WaitGroup). When Transport.Close() also calls
// subscriber.Close(), the double-close races on channel close in some backends (SQLite).
// sync.Once ensures exactly one close regardless of call count or concurrency.
func onceCloser(fn func() error) func() error {
	var once sync.Once
	return func() error {
		var err error
		once.Do(func() { err = fn() })
		return err
	}
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
			Publisher:        pubSub,
			Subscriber:       pubSub,
			FanOutSubscriber: pubSub, // GoChannel: all subscribers get all messages by default
			closeFns:         []func() error{pubSub.Close},
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
			AutoProvision: true,
			DurablePrefix: consumerGroup,
			TrackMsgId:    true,
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
			AutoProvision: true,
			DurablePrefix: fanOutID,
			TrackMsgId:    true,
		},
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriber.Close()
		return nil, fmt.Errorf("nats fan-out subscriber: %w", err)
	}

	return &Transport{
		Publisher:        publisher,
		Subscriber:       subscriber,
		FanOutSubscriber: fanOutSub,
		closeFns:         []func() error{publisher.Close, onceCloser(subscriber.Close), onceCloser(fanOutSub.Close)},
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

	return &Transport{
		Publisher:        publisher,
		Subscriber:       subscriber,
		FanOutSubscriber: fanOutSub,
		closeFns:         []func() error{publisher.Close, onceCloser(subscriber.Close), onceCloser(fanOutSub.Close)},
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

	consumerGroup := cfg.Namespace
	if consumerGroup == "" {
		consumerGroup = "brainkit"
	}
	consumerGroup = sanitizeDurable(consumerGroup)

	// Command subscriber — consumer group for competing consumers
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

	// Fan-out subscriber — unique consumer group per instance
	fanOutGroup := consumerGroup + "-fo-" + watermill.NewShortUUID()
	fanOutSub, err := wmredis.NewSubscriber(wmredis.SubscriberConfig{
		Client:        rdb,
		Unmarshaller:  wmredis.DefaultMarshallerUnmarshaller{},
		ConsumerGroup: fanOutGroup,
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriber.Close()
		rdb.Close()
		return nil, fmt.Errorf("redis fan-out subscriber: %w", err)
	}

	return &Transport{
		Publisher:        publisher,
		Subscriber:       subscriber,
		FanOutSubscriber: fanOutSub,
		closeFns:         []func() error{publisher.Close, onceCloser(subscriber.Close), onceCloser(fanOutSub.Close), rdb.Close},
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
		Publisher:        publisher,
		Subscriber:       subscriber,
		FanOutSubscriber: subscriber, // SQL: single-instance, no consumer group distinction
		closeFns:         []func() error{publisher.Close, onceCloser(subscriber.Close), db.Close},
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
		// modernc.org/sqlite supports _pragma DSN param — runs on EVERY new connection.
		// journal_mode=WAL: concurrent readers + writer across processes.
		// busy_timeout=10000: retry up to 10s on lock contention instead of SQLITE_BUSY.
		// synchronous=NORMAL: safe with WAL, reduces fsync overhead.
		dbPath = "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)"
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
		Publisher:        publisher,
		Subscriber:       subscriber,
		FanOutSubscriber: subscriber, // SQLite: single-instance, no consumer group distinction
		closeFns:         []func() error{publisher.Close, onceCloser(subscriber.Close), db.Close},
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
