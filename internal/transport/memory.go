package transport

import (
	"context"
	"fmt"
	"sync"
)

const memorySubscriptionBuffer = 1024
const memoryBacklogLimit = 4096

type memoryBroker struct {
	mu      sync.Mutex
	closed  bool
	subs    map[string]map[*memorySubscription]struct{}
	backlog map[string][]*Message
}

func newMemoryBroker() *memoryBroker {
	return &memoryBroker{
		subs:    make(map[string]map[*memorySubscription]struct{}),
		backlog: make(map[string][]*Message),
	}
}

func (b *memoryBroker) Publish(topic string, messages ...*Message) error {
	if topic == "" {
		return fmt.Errorf("memory transport: topic is required")
	}
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if err := b.publishOne(topic, msg); err != nil {
			return err
		}
	}
	return nil
}

func (b *memoryBroker) publishOne(topic string, msg *Message) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return fmt.Errorf("memory transport: closed")
	}
	subs := make([]*memorySubscription, 0, len(b.subs[topic]))
	for sub := range b.subs[topic] {
		subs = append(subs, sub)
	}
	if len(subs) == 0 {
		queue := append(b.backlog[topic], msg.clone())
		if len(queue) > memoryBacklogLimit {
			queue = queue[len(queue)-memoryBacklogLimit:]
		}
		b.backlog[topic] = queue
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()

	for _, sub := range subs {
		sub.deliver(msg.clone())
	}
	return nil
}

func (b *memoryBroker) Subscribe(ctx context.Context, topic string) (<-chan *Message, error) {
	if topic == "" {
		return nil, fmt.Errorf("memory transport: topic is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	sub := &memorySubscription{
		ctx: ctx,
	}

	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil, fmt.Errorf("memory transport: closed")
	}
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[*memorySubscription]struct{})
	}
	backlog := b.backlog[topic]
	bufferSize := memorySubscriptionBuffer
	if len(backlog) > bufferSize {
		bufferSize = len(backlog)
	}
	sub.ch = make(chan *Message, bufferSize)
	b.subs[topic][sub] = struct{}{}
	delete(b.backlog, topic)
	b.mu.Unlock()

	for _, msg := range backlog {
		if !sub.deliver(msg.clone()) {
			break
		}
	}

	sub.watchCancel(context.AfterFunc(ctx, func() {
		b.unsubscribe(topic, sub)
	}))

	return sub.ch, nil
}

func (b *memoryBroker) unsubscribe(topic string, sub *memorySubscription) {
	b.mu.Lock()
	if subs := b.subs[topic]; subs != nil {
		delete(subs, sub)
		if len(subs) == 0 {
			delete(b.subs, topic)
		}
	}
	b.mu.Unlock()
	sub.close()
}

func (b *memoryBroker) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	var subs []*memorySubscription
	for _, topicSubs := range b.subs {
		for sub := range topicSubs {
			subs = append(subs, sub)
		}
	}
	b.subs = make(map[string]map[*memorySubscription]struct{})
	b.backlog = make(map[string][]*Message)
	b.mu.Unlock()

	for _, sub := range subs {
		sub.close()
	}
	return nil
}

type memorySubscription struct {
	ctx             context.Context
	ch              chan *Message
	stopCancelWatch func() bool

	mu     sync.Mutex
	closed bool
}

func (s *memorySubscription) watchCancel(stop func() bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		if stop != nil {
			stop()
		}
		return
	}
	s.stopCancelWatch = stop
}

func (s *memorySubscription) deliver(msg *Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	select {
	case s.ch <- msg:
		return true
	case <-s.ctx.Done():
		return false
	}
}

func (s *memorySubscription) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	stopCancelWatch := s.stopCancelWatch
	s.stopCancelWatch = nil
	close(s.ch)
	s.mu.Unlock()
	if stopCancelWatch != nil {
		stopCancelWatch()
	}
}
