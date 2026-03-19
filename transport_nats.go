package brainkit

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/brainlet/brainkit/bus"
	"github.com/nats-io/nats.go"
)

// NATSTransport implements bus.Transport using NATS as the message broker.
// Bus topics map directly to NATS subjects.
// Worker groups map to NATS queue subscriptions.
// Address routing uses subject prefixes.
type NATSTransport struct {
	conn *nats.Conn
	mu   sync.RWMutex
	subs map[bus.SubscriptionID]*nats.Subscription
}

// NewNATSTransport connects to a NATS server and returns a transport.
func NewNATSTransport(url string, opts ...nats.Option) (*NATSTransport, error) {
	defaultOpts := []nats.Option{
		nats.MaxReconnects(-1),
		nats.ReconnectWait(nats.DefaultReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("[nats] disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[nats] reconnected to %s", nc.ConnectedUrl())
		}),
	}
	allOpts := append(defaultOpts, opts...)

	nc, err := nats.Connect(url, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("nats transport: connect to %s: %w", url, err)
	}

	return &NATSTransport{
		conn: nc,
		subs: make(map[bus.SubscriptionID]*nats.Subscription),
	}, nil
}

func (t *NATSTransport) Publish(msg bus.Message) error {
	subject := topicToNATSSubject(msg.Topic, msg.Address)
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("nats: marshal message: %w", err)
	}
	return t.conn.Publish(subject, data)
}

func (t *NATSTransport) Forward(msg bus.Message, target string) error {
	subject := addressToNATSPrefix(target) + "." + msg.Topic
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("nats: marshal message: %w", err)
	}
	return t.conn.Publish(subject, data)
}

func (t *NATSTransport) Subscribe(info bus.SubscriberInfo) error {
	subject := patternToNATSSubject(info.Pattern, info.Address)

	handler := func(m *nats.Msg) {
		var msg bus.Message
		if err := json.Unmarshal(m.Data, &msg); err != nil {
			log.Printf("[nats] unmarshal error on %s: %v", m.Subject, err)
			return
		}
		info.Handler(msg)
	}

	var sub *nats.Subscription
	var err error

	if info.Group != "" {
		sub, err = t.conn.QueueSubscribe(subject, info.Group, handler)
	} else {
		sub, err = t.conn.Subscribe(subject, handler)
	}
	if err != nil {
		return fmt.Errorf("nats: subscribe to %s: %w", subject, err)
	}

	t.mu.Lock()
	t.subs[info.ID] = sub
	t.mu.Unlock()

	return nil
}

func (t *NATSTransport) Unsubscribe(id bus.SubscriptionID) error {
	t.mu.Lock()
	sub, ok := t.subs[id]
	if ok {
		delete(t.subs, id)
	}
	t.mu.Unlock()

	if ok {
		return sub.Unsubscribe()
	}
	return nil
}

func (t *NATSTransport) Metrics() bus.TransportMetrics {
	return bus.TransportMetrics{
		Topics:  make(map[string]bus.TopicMetrics),
		Workers: make(map[string]bus.WorkerGroupMetrics),
	}
}

func (t *NATSTransport) SubscriberCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.subs)
}

func (t *NATSTransport) Close() error {
	t.mu.Lock()
	for id, sub := range t.subs {
		sub.Unsubscribe()
		delete(t.subs, id)
	}
	t.mu.Unlock()

	t.conn.Drain()
	t.conn.Close()
	return nil
}

// topicToNATSSubject converts a bus topic to a NATS subject.
// "tools.call" with address "" -> "tools.call"
// "tools.call" with address "kit:staging" -> "kit.staging.tools.call"
func topicToNATSSubject(topic, address string) string {
	if address == "" {
		return topic
	}
	return addressToNATSPrefix(address) + "." + topic
}

// patternToNATSSubject converts a bus topic pattern to a NATS subject.
// "tools.*" -> "tools.*" (same), "events.**" -> "events.>"
func patternToNATSSubject(pattern, address string) string {
	subject := strings.ReplaceAll(pattern, "**", ">")
	if address != "" {
		return addressToNATSPrefix(address) + "." + subject
	}
	return subject
}

// addressToNATSPrefix converts "kit:staging" -> "kit.staging"
func addressToNATSPrefix(address string) string {
	return strings.ReplaceAll(address, ":", ".")
}
