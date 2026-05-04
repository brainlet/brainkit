package transport

import (
	"fmt"
	"strconv"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
)

// MaxDepth is the maximum cascade depth before cycle detection triggers.
const MaxDepth = 16

// DepthMiddleware rejects messages that exceed the cascade depth limit (cycle detection).
func DepthMiddleware(h HandlerFunc) HandlerFunc {
	return func(msg *Message) ([]*Message, error) {
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
func CallerIDMiddleware(defaultCallerID string) Middleware {
	return func(h HandlerFunc) HandlerFunc {
		return func(msg *Message) ([]*Message, error) {
			if msg.Metadata.Get("callerId") == "" {
				msg.Metadata.Set("callerId", defaultCallerID)
			}
			return h(msg)
		}
	}
}

// MetricsMiddleware tracks message processing time and counts.
func MetricsMiddleware(m *Metrics) Middleware {
	return func(h HandlerFunc) HandlerFunc {
		return func(msg *Message) ([]*Message, error) {
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
func MaxConcurrencyMiddleware(n int) Middleware {
	if n <= 0 {
		return func(h HandlerFunc) HandlerFunc { return h }
	}
	sem := make(chan struct{}, n)
	return func(h HandlerFunc) HandlerFunc {
		return func(msg *Message) ([]*Message, error) {
			sem <- struct{}{}
			defer func() { <-sem }()
			return h(msg)
		}
	}
}

// Metrics tracks message processing statistics (thread-safe).
type Metrics struct {
	mu                  syncx.Mutex
	published           map[string]int
	handled             map[string]int
	errors              map[string]int
	handleDurationTotal map[string]time.Duration
	handleDurationMax   map[string]time.Duration
	handleDurationCount map[string]int
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		published:           make(map[string]int),
		handled:             make(map[string]int),
		errors:              make(map[string]int),
		handleDurationTotal: make(map[string]time.Duration),
		handleDurationMax:   make(map[string]time.Duration),
		handleDurationCount: make(map[string]int),
	}
}

// Record records a message handling event.
func (m *Metrics) Record(topic string, duration time.Duration, err error) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handled[topic]++
	m.handleDurationCount[topic]++
	m.handleDurationTotal[topic] += duration
	if duration > m.handleDurationMax[topic] {
		m.handleDurationMax[topic] = duration
	}
	if err != nil {
		m.errors[topic]++
	}
}

// Published records a message publish event.
func (m *Metrics) Published(topic string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.published[topic]++
	m.mu.Unlock()
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	if m == nil {
		return MetricsSnapshot{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	snap := MetricsSnapshot{
		Published:           make(map[string]int, len(m.published)),
		Handled:             make(map[string]int, len(m.handled)),
		Errors:              make(map[string]int, len(m.errors)),
		HandleDurationTotal: make(map[string]time.Duration, len(m.handleDurationTotal)),
		HandleDurationMax:   make(map[string]time.Duration, len(m.handleDurationMax)),
		HandleDurationCount: make(map[string]int, len(m.handleDurationCount)),
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
	for k, v := range m.handleDurationTotal {
		snap.HandleDurationTotal[k] = v
	}
	for k, v := range m.handleDurationMax {
		snap.HandleDurationMax[k] = v
	}
	for k, v := range m.handleDurationCount {
		snap.HandleDurationCount[k] = v
	}
	return snap
}
