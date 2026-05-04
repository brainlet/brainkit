package brainkit

import (
	"fmt"
	"path/filepath"
	"sync"
)

// TransportConfig configures the bus transport. Create with EmbeddedNATS(),
// NATS(), AMQP(), Redis(), or Memory(). Non-memory backends require importing
// a backend package such as transports/embeddednats, transports/nats,
// transports/amqp, transports/redis, or the aggregate transports package.
type TransportConfig struct {
	typ      string
	natsURL  string
	natsName string
	amqpURL  string
	redisURL string
}

// TransportBuildContext is passed to registered transport backend builders.
type TransportBuildContext struct {
	Config       TransportConfig
	Namespace    string
	FSRoot       string
	NATSStoreDir string
}

// TransportBuilder builds a concrete internal transport. External backend
// packages register builders so importing brainkit alone does not compile every
// network backend.
type TransportBuilder func(TransportBuildContext) (any, error)

var transportBuilders = struct {
	sync.RWMutex
	m map[string]TransportBuilder
}{m: map[string]TransportBuilder{}}

// RegisterTransportBuilder registers a non-memory transport backend.
func RegisterTransportBuilder(kind string, builder TransportBuilder) {
	transportBuilders.Lock()
	defer transportBuilders.Unlock()
	if builder == nil {
		delete(transportBuilders.m, kind)
		return
	}
	transportBuilders.m[kind] = builder
}

func buildConfiguredTransport(cfg TransportConfig, namespace, fsRoot string) (any, error) {
	kind := cfg.typ
	if kind == "" || kind == "memory" {
		return nil, nil
	}

	transportBuilders.RLock()
	builder := transportBuilders.m[kind]
	transportBuilders.RUnlock()
	if builder == nil {
		return nil, fmt.Errorf("brainkit: transport %q requires importing a backend package such as github.com/brainlet/brainkit/transports/%s or the aggregate github.com/brainlet/brainkit/transports", kind, transportImportHint(kind))
	}

	natsStoreDir := ""
	if kind == "embedded" && fsRoot != "" {
		natsStoreDir = filepath.Join(fsRoot, "nats-data")
	}
	return builder(TransportBuildContext{
		Config:       cfg,
		Namespace:    namespace,
		FSRoot:       fsRoot,
		NATSStoreDir: natsStoreDir,
	})
}

// Kind returns the normalized transport kind.
func (c TransportConfig) Kind() string { return c.typ }

// NATSURL returns the configured NATS URL.
func (c TransportConfig) NATSURL() string { return c.natsURL }

// NATSName returns the configured NATS durable name.
func (c TransportConfig) NATSName() string { return c.natsName }

// AMQPURL returns the configured AMQP URL.
func (c TransportConfig) AMQPURL() string { return c.amqpURL }

// RedisURL returns the configured Redis URL.
func (c TransportConfig) RedisURL() string { return c.redisURL }

// TransportOption configures a transport constructor.
type TransportOption func(*TransportConfig)

// EmbeddedNATS creates a zero-config in-process NATS server with JetStream.
// This is the default when no transport is configured.
func EmbeddedNATS(opts ...TransportOption) TransportConfig {
	c := TransportConfig{typ: "embedded"}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// NATS connects to an external NATS server.
func NATS(url string, opts ...TransportOption) TransportConfig {
	c := TransportConfig{typ: "nats", natsURL: url}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// WithNATSName sets the durable consumer prefix for NATS JetStream.
func WithNATSName(name string) TransportOption {
	return func(c *TransportConfig) { c.natsName = name }
}

// AMQP connects to a RabbitMQ server.
func AMQP(url string) TransportConfig {
	return TransportConfig{typ: "amqp", amqpURL: url}
}

// Redis connects to a Redis Streams server.
func Redis(url string) TransportConfig {
	return TransportConfig{typ: "redis", redisURL: url}
}

// Memory creates an in-process GoChannel transport.
// Fast and synchronous — use for tests that don't need real pub/sub.
func Memory() TransportConfig {
	return TransportConfig{typ: "memory"}
}

func transportImportHint(kind string) string {
	switch kind {
	case "embedded":
		return "embeddednats"
	case "nats", "amqp", "redis":
		return kind
	default:
		return "embeddednats"
	}
}
