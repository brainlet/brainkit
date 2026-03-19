package bus

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultHandlerTimeout is the default Ask timeout.
	DefaultHandlerTimeout = 30 * time.Second
	// DefaultJobTimeout is the default cascade timeout.
	DefaultJobTimeout = 5 * time.Minute
	// DefaultJobRetention is how long completed jobs stay in memory.
	DefaultJobRetention = 5 * time.Minute
	// ProtocolVersion is the bus protocol version.
	ProtocolVersion = "v1"
)

// Bus is the async-first message router.
type Bus struct {
	transport    Transport
	interceptors []interceptorEntry
	jobs         *jobTracker

	handlerTimeout time.Duration

	mu     sync.RWMutex
	closed bool
	names  map[string]bool // registered Kit names (for collision detection)

	// Ask reply tracking
	replyMu       sync.Mutex
	replyOneshots  map[string]*replyEntry // replyTo topic → entry
}

type replyEntry struct {
	callback func(Message)
	once     sync.Once
	timer    *time.Timer
	subID    SubscriptionID
}

// NewBus creates a Bus backed by the given transport.
func NewBus(transport Transport, opts ...BusOption) *Bus {
	cfg := &busConfig{
		handlerTimeout: DefaultHandlerTimeout,
		jobTimeout:     DefaultJobTimeout,
		jobRetention:   DefaultJobRetention,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	b := &Bus{
		transport:      transport,
		handlerTimeout: cfg.handlerTimeout,
		jobs:           newJobTracker(cfg.jobTimeout, cfg.jobRetention),
		names:          make(map[string]bool),
		replyOneshots:  make(map[string]*replyEntry),
	}
	return b
}

// RegisterName registers a Kit name on the bus. Returns error if already taken.
func (b *Bus) RegisterName(name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.names[name] {
		return fmt.Errorf("bus: Kit name %q already registered", name)
	}
	b.names[name] = true
	return nil
}

// UnregisterName removes a Kit name from the bus.
func (b *Bus) UnregisterName(name string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.names, name)
}

// Send broadcasts a message (fire-and-forget).
func (b *Bus) Send(msg Message) error {
	if msg.Depth >= MaxDepth {
		return fmt.Errorf("bus: cycle detected (depth %d >= max %d)", msg.Depth, MaxDepth)
	}
	b.stamp(&msg)

	if err := b.runInterceptors(&msg); err != nil {
		return fmt.Errorf("bus: interceptor rejected: %w", err)
	}

	// Track in job
	b.jobs.getOrCreate(msg.TraceID)
	b.jobs.incrementMessages(msg.TraceID)

	// Address resolution: route non-local messages via Forward
	if msg.Address != "" && !isLocalAddress(msg.Address) {
		target := extractHost(msg.Address)
		return b.transport.Forward(msg, target)
	}

	return b.transport.Publish(msg)
}

// Ask sends a message and registers a callback for the reply.
// The callback is guaranteed to be called exactly once (reply or timeout).
// Returns a cancel function that aborts the pending ask.
func (b *Bus) Ask(msg Message, callback func(Message)) (cancel func()) {
	b.stamp(&msg)

	replyTopic := "_reply." + uuid.NewString()
	msg.ReplyTo = replyTopic

	entry := &replyEntry{
		callback: callback,
	}

	// Subscribe to the reply topic (one-shot)
	subID := b.On(replyTopic, func(reply Message, _ ReplyFunc) {
		b.replyMu.Lock()
		e, ok := b.replyOneshots[replyTopic]
		if ok {
			delete(b.replyOneshots, replyTopic)
		}
		b.replyMu.Unlock()

		if ok && e != nil {
			if e.timer != nil {
				e.timer.Stop()
			}
			e.once.Do(func() {
				b.jobs.decrementPending(reply.TraceID)
				callback(reply)
			})
			b.Off(e.subID)
		}
	})
	entry.subID = subID

	// Timeout timer
	entry.timer = time.AfterFunc(b.handlerTimeout, func() {
		b.replyMu.Lock()
		e, ok := b.replyOneshots[replyTopic]
		if ok {
			delete(b.replyOneshots, replyTopic)
		}
		b.replyMu.Unlock()

		if ok && e != nil {
			e.once.Do(func() {
				errPayload, _ := json.Marshal(map[string]string{
					"error": fmt.Sprintf("timeout after %v for topic %q", b.handlerTimeout, msg.Topic),
				})
				b.jobs.decrementPending(msg.TraceID)
				callback(Message{
					Version:  ProtocolVersion,
					Topic:    replyTopic,
					CallerID: "bus",
					Payload:  errPayload,
					TraceID:  msg.TraceID,
					ParentID: msg.ID,
				})
			})
			b.Off(e.subID)
		}
	})

	b.replyMu.Lock()
	b.replyOneshots[replyTopic] = entry
	b.replyMu.Unlock()

	// Track pending Ask in job
	b.jobs.getOrCreate(msg.TraceID)
	b.jobs.incrementPending(msg.TraceID)
	b.jobs.incrementMessages(msg.TraceID)

	// Run interceptors + publish
	if err := b.runInterceptors(&msg); err != nil {
		// Interceptor rejected — fire callback with error immediately
		entry.timer.Stop()
		b.replyMu.Lock()
		delete(b.replyOneshots, replyTopic)
		b.replyMu.Unlock()
		entry.once.Do(func() {
			errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
			b.jobs.decrementPending(msg.TraceID)
			callback(Message{
				Version: ProtocolVersion, Topic: replyTopic, CallerID: "bus",
				Payload: errPayload, TraceID: msg.TraceID,
			})
		})
		b.Off(subID)
		return func() {}
	}

	// Address resolution: route non-local messages via Forward (matches Bus.Send)
	if msg.Address != "" && !isLocalAddress(msg.Address) {
		target := extractHost(msg.Address)
		if err := b.transport.Forward(msg, target); err != nil {
			entry.timer.Stop()
			b.replyMu.Lock()
			delete(b.replyOneshots, replyTopic)
			b.replyMu.Unlock()
			entry.once.Do(func() {
				errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
				b.jobs.decrementPending(msg.TraceID)
				callback(Message{
					Version: ProtocolVersion, Topic: replyTopic, CallerID: "bus",
					Payload: errPayload, TraceID: msg.TraceID,
				})
			})
			b.Off(subID)
			return func() {}
		}
	} else {
		b.transport.Publish(msg)
	}

	return func() {
		// Cancel: stop timer, remove oneshot, unsubscribe
		b.replyMu.Lock()
		e, ok := b.replyOneshots[replyTopic]
		if ok {
			delete(b.replyOneshots, replyTopic)
		}
		b.replyMu.Unlock()

		if ok && e != nil {
			e.timer.Stop()
			b.Off(e.subID)
		}
	}
}

// On subscribes to messages matching a topic pattern.
func (b *Bus) On(pattern string, handler func(Message, ReplyFunc), opts ...SubscribeOption) SubscriptionID {
	cfg := &subscribeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	id := SubscriptionID(uuid.NewString())

	b.transport.Subscribe(SubscriberInfo{
		ID:      id,
		Pattern: pattern,
		Group:   cfg.group,
		Address: cfg.address,
		Handler: func(msg Message) {
			// Build reply function
			replied := false
			reply := func(payload json.RawMessage) {
				if replied {
					return // one-shot
				}
				replied = true
				if msg.ReplyTo == "" {
					return // was a Send, not Ask
				}
				b.Send(Message{
					Topic:    msg.ReplyTo,
					CallerID: "bus",
					Payload:  payload,
					TraceID:  msg.TraceID,
					ParentID: msg.ID,
					Depth:    msg.Depth + 1,
				})
			}
			handler(msg, reply)
		},
	})

	return id
}

// Off removes a subscription.
func (b *Bus) Off(id SubscriptionID) {
	b.transport.Unsubscribe(id)
}

// AddInterceptor registers an interceptor sorted by priority.
func (b *Bus) AddInterceptor(i Interceptor) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.interceptors = append(b.interceptors, interceptorEntry{
		interceptor: i,
		priority:    i.Priority(),
	})
	sort.Slice(b.interceptors, func(a, c int) bool {
		return b.interceptors[a].priority < b.interceptors[c].priority
	})
}

// Jobs returns all tracked jobs.
func (b *Bus) Jobs() []Job {
	return b.jobs.list()
}

// Job returns a specific job by trace ID.
func (b *Bus) Job(traceID string) *Job {
	return b.jobs.get(traceID)
}

// SetJobTimeout sets the default timeout for job cascades at runtime.
func (b *Bus) SetJobTimeout(timeout time.Duration) {
	b.jobs.mu.Lock()
	b.jobs.timeout = timeout
	b.jobs.mu.Unlock()
}

// Metrics returns a snapshot of bus metrics.
func (b *Bus) Metrics() BusMetrics {
	tm := b.transport.Metrics()
	jobs := b.jobs.list()
	active := 0
	for _, j := range jobs {
		if j.Status == "running" {
			active++
		}
	}

	return BusMetrics{
		Transport:   tm,
		ActiveJobs:  active,
		TotalJobs:   len(jobs),
		Subscribers: b.transport.SubscriberCount(),
	}
}

// Close shuts down the bus.
func (b *Bus) Close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()

	b.jobs.close()

	// Cancel all pending reply entries
	b.replyMu.Lock()
	for topic, entry := range b.replyOneshots {
		entry.timer.Stop()
		delete(b.replyOneshots, topic)
	}
	b.replyMu.Unlock()

	b.transport.Close()
}

// --- internal helpers ---

func (b *Bus) stamp(msg *Message) {
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.TraceID == "" {
		msg.TraceID = msg.ID
	}
	msg.Version = ProtocolVersion
}

func (b *Bus) runInterceptors(msg *Message) error {
	b.mu.RLock()
	interceptors := make([]interceptorEntry, len(b.interceptors))
	copy(interceptors, b.interceptors)
	b.mu.RUnlock()
	return runInterceptors(interceptors, msg)
}

// isLocalAddress returns true if the address targets the local bus.
func isLocalAddress(addr string) bool {
	if addr == "" {
		return true
	}
	if strings.HasPrefix(addr, "kit:") || strings.HasPrefix(addr, "host:") {
		return false
	}
	return true
}

// extractHost returns the first routing segment from an address.
func extractHost(addr string) string {
	if idx := strings.Index(addr, "/"); idx != -1 {
		return addr[:idx]
	}
	return addr
}
