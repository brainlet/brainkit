package transport

import (
	"fmt"
	"strconv"
	"github.com/brainlet/brainkit/internal/syncx"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// MaxDepth is the maximum cascade depth before cycle detection triggers.
const MaxDepth = 16

// DepthMiddleware rejects messages that exceed the cascade depth limit (cycle detection).
func DepthMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		depthStr := msg.Metadata.Get("depth")
		if depthStr != "" {
			depth, _ := strconv.Atoi(depthStr)
			if depth >= MaxDepth {
				return nil, fmt.Errorf("%w (depth %d >= max %d)", ErrCycleDetected, depth, MaxDepth)
			}
		}
		return h(msg)
	}
}

// CallerIDMiddleware stamps a default callerID if not already set.
func CallerIDMiddleware(defaultCallerID string) func(message.HandlerFunc) message.HandlerFunc {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			if msg.Metadata.Get("callerId") == "" {
				msg.Metadata.Set("callerId", defaultCallerID)
			}
			return h(msg)
		}
	}
}

// MetricsMiddleware tracks message processing time and counts.
func MetricsMiddleware(m *Metrics) func(message.HandlerFunc) message.HandlerFunc {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			topic := msg.Metadata.Get("_subscription_topic")
			start := time.Now()
			result, err := h(msg)
			m.Record(topic, time.Since(start), err)
			return result, err
		}
	}
}

// MaxConcurrencyMiddleware limits concurrent handler invocations.
// When the limit is reached, new messages block until a slot is available.
func MaxConcurrencyMiddleware(n int) func(message.HandlerFunc) message.HandlerFunc {
	if n <= 0 {
		return func(h message.HandlerFunc) message.HandlerFunc { return h }
	}
	sem := make(chan struct{}, n)
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			sem <- struct{}{}
			defer func() { <-sem }()
			return h(msg)
		}
	}
}

// Metrics tracks message processing statistics (thread-safe).
type Metrics struct {
	mu        syncx.Mutex
	published map[string]int
	handled   map[string]int
	errors    map[string]int
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		published: make(map[string]int),
		handled:   make(map[string]int),
		errors:    make(map[string]int),
	}
}

// Record records a message handling event.
func (m *Metrics) Record(topic string, duration time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handled[topic]++
	if err != nil {
		m.errors[topic]++
	}
}

// Published records a message publish event.
func (m *Metrics) Published(topic string) {
	m.mu.Lock()
	m.published[topic]++
	m.mu.Unlock()
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := MetricsSnapshot{
		Published: make(map[string]int, len(m.published)),
		Handled:   make(map[string]int, len(m.handled)),
		Errors:    make(map[string]int, len(m.errors)),
	}
	for k, v := range m.published {
		snap.Published[k] = v
	}
	for k, v := range m.handled {
		snap.Handled[k] = v
	}
	for k, v := range m.errors {
		snap.Errors[k] = v
	}
	return snap
}

