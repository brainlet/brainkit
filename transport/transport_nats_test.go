//go:build integration

package transport_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	brainkit "github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/bus"
	transportpkg "github.com/brainlet/brainkit/transport"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startNATSContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "nats:latest",
		ExposedPorts: []string{"4222/tcp"},
		WaitingFor:   wait.ForListeningPort("4222/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start NATS container: %v", err)
	}
	t.Cleanup(func() { container.Terminate(ctx) })

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "4222")

	return fmt.Sprintf("nats://%s:%s", host, port.Port())
}

func TestNATSTransport_PublishSubscribe(t *testing.T) {
	url := startNATSContainer(t)

	transport, err := transportpkg.NewNATSTransport(url)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()

	received := make(chan bus.Message, 1)

	transport.Subscribe(bus.SubscriberInfo{
		ID:      bus.SubscriptionID("sub-1"),
		Pattern: "test.topic",
		Handler: func(msg bus.Message) {
			received <- msg
		},
	})

	transport.Publish(bus.Message{
		Topic:    "test.topic",
		CallerID: "test",
		Payload:  json.RawMessage(`{"hello":"world"}`),
	})

	select {
	case msg := <-received:
		if msg.Topic != "test.topic" {
			t.Errorf("topic = %q", msg.Topic)
		}
		var payload map[string]string
		json.Unmarshal(msg.Payload, &payload)
		if payload["hello"] != "world" {
			t.Errorf("payload = %v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestNATSTransport_WildcardSubscription(t *testing.T) {
	url := startNATSContainer(t)

	transport, err := transportpkg.NewNATSTransport(url)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()

	received := make(chan bus.Message, 10)

	transport.Subscribe(bus.SubscriberInfo{
		ID:      bus.SubscriptionID("sub-wildcard"),
		Pattern: "events.*",
		Handler: func(msg bus.Message) {
			received <- msg
		},
	})

	transport.Publish(bus.Message{Topic: "events.order", Payload: json.RawMessage(`{"type":"order"}`)})
	transport.Publish(bus.Message{Topic: "events.payment", Payload: json.RawMessage(`{"type":"payment"}`)})
	transport.Publish(bus.Message{Topic: "other.topic", Payload: json.RawMessage(`{"type":"other"}`)})

	// Wait for 2 matching messages
	for i := 0; i < 2; i++ {
		select {
		case <-received:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout: only received %d/2 wildcard messages", i)
		}
	}

	// Verify other.topic was NOT received
	select {
	case <-received:
		t.Error("received unexpected 3rd message")
	case <-time.After(200 * time.Millisecond):
		// expected
	}
}

func TestNATSTransport_QueueGroup(t *testing.T) {
	url := startNATSContainer(t)

	transport, err := transportpkg.NewNATSTransport(url)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()

	counts := [3]int{}
	var mu sync.Mutex

	for i := 0; i < 3; i++ {
		idx := i
		transport.Subscribe(bus.SubscriberInfo{
			ID:      bus.SubscriptionID(fmt.Sprintf("worker-%d", i)),
			Pattern: "work.queue",
			Group:   "processors",
			Handler: func(msg bus.Message) {
				mu.Lock()
				counts[idx]++
				mu.Unlock()
			},
		})
	}

	for i := 0; i < 30; i++ {
		transport.Publish(bus.Message{
			Topic:   "work.queue",
			Payload: json.RawMessage(fmt.Sprintf(`{"idx":%d}`, i)),
		})
	}

	// Poll until all 30 processed
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		total := counts[0] + counts[1] + counts[2]
		mu.Unlock()
		if total >= 30 {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timeout: %d/30 processed", counts[0]+counts[1]+counts[2])
			mu.Unlock()
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	total := counts[0] + counts[1] + counts[2]
	mu.Unlock()

	if total != 30 {
		t.Errorf("expected 30, got %d", total)
	}
	t.Logf("distribution: %d, %d, %d", counts[0], counts[1], counts[2])
}

func TestNATSTransport_AddressedMessages(t *testing.T) {
	url := startNATSContainer(t)

	transport, err := transportpkg.NewNATSTransport(url)
	if err != nil {
		t.Fatal(err)
	}
	defer transport.Close()

	received := make(chan bus.Message, 1)

	transport.Subscribe(bus.SubscriberInfo{
		ID:      bus.SubscriptionID("sub-addressed"),
		Pattern: "tools.call",
		Address: "kit:staging",
		Handler: func(msg bus.Message) {
			received <- msg
		},
	})

	transport.Publish(bus.Message{
		Topic:   "tools.call",
		Address: "kit:staging",
		Payload: json.RawMessage(`{"name":"test"}`),
	})

	select {
	case msg := <-received:
		if msg.Topic != "tools.call" {
			t.Errorf("topic = %q", msg.Topic)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestNATSTransport_TwoKitsViaNATS(t *testing.T) {
	url := startNATSContainer(t)

	kitA, err := brainkit.New(brainkit.Config{
		Name:      "nats-kit-a",
		Namespace: "a",
		Transport: "nats",
		NATS:      brainkit.NATSConfig{URL: url},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitA.Close()

	kitB, err := brainkit.New(brainkit.Config{
		Name:      "nats-kit-b",
		Namespace: "b",
		Transport: "nats",
		NATS:      brainkit.NATSConfig{URL: url},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kitB.Close()

	// Subscribe on Kit A — synchronous, active when On() returns
	received := make(chan bus.Message, 1)
	kitA.Bus.On("cross-nats.test", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	// Send from Kit B
	kitB.Bus.Send(bus.Message{
		Topic:    "cross-nats.test",
		CallerID: "nats-kit-b",
		Payload:  json.RawMessage(`{"from":"kit-b"}`),
	})

	select {
	case msg := <-received:
		var payload map[string]string
		json.Unmarshal(msg.Payload, &payload)
		if payload["from"] != "kit-b" {
			t.Errorf("payload = %v", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cross-NATS message")
	}
}
