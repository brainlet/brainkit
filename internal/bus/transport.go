package bus

import (
	"fmt"
	"sync"
)

// Transport handles message delivery.
type Transport interface {
	// Publish dispatches a message to matching subscribers.
	Publish(msg Message) error

	// Forward sends a message to a remote target.
	// Returns ErrNoRoute if unreachable.
	Forward(msg Message, target string) error

	// Subscribe registers a subscriber.
	Subscribe(info SubscriberInfo) error

	// Unsubscribe removes a subscription.
	Unsubscribe(id SubscriptionID) error

	// Metrics returns transport-level stats.
	Metrics() TransportMetrics

	// SubscriberCount returns the number of active subscribers.
	SubscriberCount() int

	// Close shuts down the transport.
	Close() error
}

// SubscriberInfo describes a subscription for the transport.
type SubscriberInfo struct {
	ID      SubscriptionID
	Pattern string
	Group   string // "" = broadcast, "name" = worker group
	Address string // "" = all, "agent:X" = filter by address
	Handler func(Message)
}

// ErrNoRoute is returned when a message can't be delivered.
var ErrNoRoute = fmt.Errorf("bus: no route to target")

// ---------------------------------------------------------------------------
// InProcessTransport — default local transport
// ---------------------------------------------------------------------------

const defaultQueueSize = 256

// InProcessTransport delivers messages in-process via goroutines and channels.
type InProcessTransport struct {
	mu           sync.RWMutex
	subscribers  map[SubscriptionID]*localSubscriber
	workerGroups map[string]*workerGroup
	closed       bool
}

type localSubscriber struct {
	info SubscriberInfo
	queue chan Message // nil for worker group members (they share the group queue)
	done  chan struct{}
}

type workerGroup struct {
	name    string
	queue   chan Message
	members map[SubscriptionID]*localSubscriber
	done    chan struct{}
}

// NewInProcessTransport creates a local in-process transport.
func NewInProcessTransport() *InProcessTransport {
	return &InProcessTransport{
		subscribers:  make(map[SubscriptionID]*localSubscriber),
		workerGroups: make(map[string]*workerGroup),
	}
}

func (t *InProcessTransport) Subscribe(info SubscriberInfo) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if info.Group != "" {
		// Worker group subscription — share a queue
		wg, ok := t.workerGroups[info.Group]
		if !ok {
			wg = &workerGroup{
				name:    info.Group,
				queue:   make(chan Message, defaultQueueSize),
				members: make(map[SubscriptionID]*localSubscriber),
				done:    make(chan struct{}),
			}
			t.workerGroups[info.Group] = wg
		}
		sub := &localSubscriber{info: info, done: make(chan struct{})}
		wg.members[info.ID] = sub
		t.subscribers[info.ID] = sub

		// Start a worker goroutine pulling from the shared queue
		go func() {
			for {
				select {
				case msg, ok := <-wg.queue:
					if !ok {
						return
					}
					info.Handler(msg)
				case <-sub.done:
					return
				case <-wg.done:
					return
				}
			}
		}()
		return nil
	}

	// Broadcast subscription — own queue + goroutine
	sub := &localSubscriber{
		info:  info,
		queue: make(chan Message, defaultQueueSize),
		done:  make(chan struct{}),
	}
	t.subscribers[info.ID] = sub

	go func() {
		for {
			select {
			case msg, ok := <-sub.queue:
				if !ok {
					return
				}
				go sub.info.Handler(msg)
			case <-sub.done:
				return
			}
		}
	}()

	return nil
}

func (t *InProcessTransport) Unsubscribe(id SubscriptionID) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	sub, ok := t.subscribers[id]
	if !ok {
		return nil
	}

	select {
	case <-sub.done:
	default:
		close(sub.done)
	}
	delete(t.subscribers, id)

	// Remove from worker group if applicable
	if sub.info.Group != "" {
		if wg, ok := t.workerGroups[sub.info.Group]; ok {
			delete(wg.members, id)
			if len(wg.members) == 0 {
				select {
				case <-wg.done:
				default:
					close(wg.done)
				}
				delete(t.workerGroups, sub.info.Group)
			}
		}
	}

	return nil
}

func (t *InProcessTransport) Publish(msg Message) error {
	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return fmt.Errorf("bus: transport closed")
	}

	// Collect matching broadcast subscribers
	var broadcastQueues []chan Message
	matchedGroups := map[string]bool{}

	for _, sub := range t.subscribers {
		if !TopicMatches(sub.info.Pattern, msg.Topic) {
			continue
		}
		// Address filter
		if sub.info.Address != "" && msg.Address != "" && msg.Address != sub.info.Address {
			continue
		}
		if sub.info.Group != "" {
			matchedGroups[sub.info.Group] = true
		} else {
			broadcastQueues = append(broadcastQueues, sub.queue)
		}
	}

	// Collect matching worker group queues
	var groupQueues []chan Message
	for groupName := range matchedGroups {
		if wg, ok := t.workerGroups[groupName]; ok {
			groupQueues = append(groupQueues, wg.queue)
		}
	}
	t.mu.RUnlock()

	// Deliver to broadcast subscribers (non-blocking)
	for _, q := range broadcastQueues {
		select {
		case q <- msg:
		default:
			// queue full — drop with warning (backpressure)
		}
	}

	// Deliver to worker groups (one message per group, workers compete)
	for _, q := range groupQueues {
		select {
		case q <- msg:
		default:
			// queue full
		}
	}

	return nil
}

func (t *InProcessTransport) Forward(msg Message, target string) error {
	return ErrNoRoute // in-process transport has no remote targets
}

func (t *InProcessTransport) Metrics() TransportMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	workers := make(map[string]WorkerGroupMetrics, len(t.workerGroups))
	for name, wg := range t.workerGroups {
		workers[name] = WorkerGroupMetrics{
			Name:    name,
			Members: len(wg.members),
			Pending: len(wg.queue),
		}
	}

	return TransportMetrics{
		Topics:  make(map[string]TopicMetrics),
		Workers: workers,
	}
}

func (t *InProcessTransport) SubscriberCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.subscribers)
}

func (t *InProcessTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true

	for _, sub := range t.subscribers {
		select {
		case <-sub.done:
		default:
			close(sub.done)
		}
	}
	for _, wg := range t.workerGroups {
		select {
		case <-wg.done:
		default:
			close(wg.done)
		}
	}
	return nil
}
