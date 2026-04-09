package brainkit

// TransportConfig configures the bus transport. Create with EmbeddedNATS(),
// NATS(), AMQP(), Redis(), or Memory().
type TransportConfig struct {
	typ      string
	natsURL  string
	natsName string
	amqpURL  string
	redisURL string
}

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
