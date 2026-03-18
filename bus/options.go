package bus

import "time"

// BusOption configures a Bus.
type BusOption func(*busConfig)

type busConfig struct {
	handlerTimeout time.Duration
	jobTimeout     time.Duration
	jobRetention   time.Duration
}

// WithHandlerTimeout sets the default handler execution timeout.
func WithHandlerTimeout(d time.Duration) BusOption {
	return func(c *busConfig) { c.handlerTimeout = d }
}

// WithJobTimeout sets the default timeout for job cascades.
func WithJobTimeout(d time.Duration) BusOption {
	return func(c *busConfig) { c.jobTimeout = d }
}

// WithJobRetention sets how long completed jobs stay in memory.
func WithJobRetention(d time.Duration) BusOption {
	return func(c *busConfig) { c.jobRetention = d }
}

// SubscribeOption configures a subscription.
type SubscribeOption func(*subscribeConfig)

type subscribeConfig struct {
	group   string // "" = broadcast, "name" = worker group
	address string // "" = all messages, "agent:X" = only addressed to this entity
}

// AsWorker joins a worker group. Only ONE subscriber in the group gets each message.
func AsWorker(group string) SubscribeOption {
	return func(c *subscribeConfig) { c.group = group }
}

// WithAddress filters: subscriber only receives messages addressed to this entity.
func WithAddress(addr string) SubscribeOption {
	return func(c *subscribeConfig) { c.address = addr }
}
