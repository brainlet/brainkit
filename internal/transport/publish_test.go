package transport_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
)

// testMsg is a simple message type for testing.
type testMsg struct {
	Value string `json:"value"`
}

func (testMsg) BusTopic() string { return "test.topic" }

func TestPublishHandle_RoundTrip(t *testing.T) {
	transportSet, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory"})
	if err != nil {
		t.Fatalf("new transport: %v", err)
	}
	router, err := transport.NewRouter("test", transport.NewMetrics(), 0)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	var received testMsg
	var wg sync.WaitGroup
	wg.Add(1)

	transport.Handle[testMsg](router, transportSet.Subscriber, func(ctx context.Context, env transport.Envelope[testMsg]) error {
		received = env.Value
		wg.Done()
		return nil
	})

	go func() {
		if err := router.Run(context.Background()); err != nil {
			t.Logf("router run: %v", err)
		}
	}()
	defer router.Close()
	<-router.Running()

	err = transport.Publish(transportSet.Publisher, testMsg{Value: "hello"}, "test-caller")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	wg.Wait()
	if received.Value != "hello" {
		t.Errorf("expected 'hello', got %q", received.Value)
	}
}

func TestDecodeEnvelope(t *testing.T) {
	payload, _ := json.Marshal(testMsg{Value: "meta-test"})
	wmsg := transport.NewMessage(payload)
	wmsg.Metadata.Set("callerId", "test-caller")
	wmsg.Metadata.Set("traceId", "trace-123")

	env, err := transport.DecodeEnvelope[testMsg](wmsg)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Value.Value != "meta-test" {
		t.Errorf("value: %q", env.Value.Value)
	}
	if env.Raw.Metadata.Get("callerId") != "test-caller" {
		t.Errorf("callerId: %q", env.Raw.Metadata.Get("callerId"))
	}
	if env.Raw.Metadata.Get("traceId") != "trace-123" {
		t.Errorf("traceId: %q", env.Raw.Metadata.Get("traceId"))
	}
}

func TestDecodeEnvelope_InvalidJSON(t *testing.T) {
	wmsg := transport.NewMessage([]byte("not json"))
	_, err := transport.DecodeEnvelope[testMsg](wmsg)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPublish_StampsCallerID(t *testing.T) {
	transportSet, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory"})
	if err != nil {
		t.Fatalf("new transport: %v", err)
	}
	router, _ := transport.NewRouter("test", transport.NewMetrics(), 0)

	var gotCallerID string
	var wg sync.WaitGroup
	wg.Add(1)

	transport.Handle[testMsg](router, transportSet.Subscriber, func(ctx context.Context, env transport.Envelope[testMsg]) error {
		gotCallerID = env.Raw.Metadata.Get("callerId")
		wg.Done()
		return nil
	})

	go router.Run(context.Background())
	defer router.Close()
	<-router.Running()

	transport.Publish(transportSet.Publisher, testMsg{Value: "x"}, "my-kit")
	wg.Wait()

	if gotCallerID != "my-kit" {
		t.Errorf("callerID = %q, want 'my-kit'", gotCallerID)
	}
}

func TestMemoryTransportBuffersUntilSubscription(t *testing.T) {
	transportSet, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory"})
	if err != nil {
		t.Fatalf("new transport: %v", err)
	}
	defer transportSet.Close()

	msg := transport.NewMessage([]byte(`{"value":"late"}`))
	msg.Metadata.Set("correlationId", "late-correlation")
	if err := transportSet.Publisher.Publish("late.topic", msg); err != nil {
		t.Fatalf("publish: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ch, err := transportSet.Subscriber.Subscribe(ctx, "late.topic")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	select {
	case got := <-ch:
		if string(got.Payload) != `{"value":"late"}` {
			t.Fatalf("payload = %s", got.Payload)
		}
		if got.Metadata.Get("correlationId") != "late-correlation" {
			t.Fatalf("correlationId = %q", got.Metadata.Get("correlationId"))
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for buffered message")
	}
}

func TestRemoteClientPublishMetricsUseLogicalTopics(t *testing.T) {
	transportSet, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory", Namespace: "metrics"})
	if err != nil {
		t.Fatalf("new transport: %v", err)
	}
	defer transportSet.Close()

	metrics := transport.NewMetrics()
	remote := transport.NewRemoteClientWithTransport("metrics", "metrics-caller", transportSet)
	remote.SetMetrics(metrics)

	ctx := context.Background()
	if _, err := remote.PublishRaw(ctx, "metrics.publish", json.RawMessage(`{"ok":true}`)); err != nil {
		t.Fatalf("publish raw: %v", err)
	}
	if _, err := remote.PublishRawWithMeta(ctx, "metrics.meta", json.RawMessage(`{"ok":true}`), map[string]string{"retryCount": "1"}); err != nil {
		t.Fatalf("publish raw with meta: %v", err)
	}
	if _, err := remote.PublishRawToNamespace(ctx, "peer", "metrics.cross", json.RawMessage(`{"ok":true}`)); err != nil {
		t.Fatalf("publish raw to namespace: %v", err)
	}
	if err := remote.PublishRawGlobal(ctx, "_brainkit.global.metrics", json.RawMessage(`{"ok":true}`)); err != nil {
		t.Fatalf("publish raw global: %v", err)
	}

	snap := metrics.Snapshot()
	for _, topic := range []string{"metrics.publish", "metrics.meta", "metrics.cross", "_brainkit.global.metrics"} {
		if snap.Published[topic] != 1 {
			t.Fatalf("published[%q] = %d, want 1; snapshot=%v", topic, snap.Published[topic], snap.Published)
		}
	}
	for topic := range snap.Published {
		if topic == "metrics.metrics.publish" || topic == "peer.metrics.cross" {
			t.Fatalf("published metrics should use logical topics, got wire topic %q", topic)
		}
	}
}

func TestRemoteClientSubscriptionHandleWaitsForHandlerExit(t *testing.T) {
	transportSet, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory", Namespace: "raw-sub"})
	if err != nil {
		t.Fatalf("new transport: %v", err)
	}
	defer transportSet.Close()

	remote := transport.NewRemoteClientWithTransport("raw-sub", "raw-sub-caller", transportSet)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	handle, err := remote.SubscribeRawHandle(context.Background(), "events.blocking", func(sdk.Message) {
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
	})
	if err != nil {
		t.Fatalf("SubscribeRawHandle: %v", err)
	}
	if got := remote.ActiveSubscriptions(); got != 1 {
		t.Fatalf("ActiveSubscriptions = %d, want 1", got)
	}

	if _, err := remote.PublishRaw(context.Background(), "events.blocking", json.RawMessage(`{"ok":true}`)); err != nil {
		t.Fatalf("PublishRaw: %v", err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not receive message")
	}

	stopCtx, cancelStop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = handle.CloseContext(stopCtx)
	cancelStop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline", err)
	}
	if got := remote.ActiveSubscriptions(); got != 1 {
		t.Fatalf("ActiveSubscriptions after close timeout = %d, want retained live subscription", got)
	}

	close(release)
	retryCtx, cancelRetry := context.WithTimeout(context.Background(), time.Second)
	defer cancelRetry()
	if err := handle.CloseContext(retryCtx); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	if got := remote.ActiveSubscriptions(); got != 0 {
		t.Fatalf("ActiveSubscriptions after retry = %d, want 0", got)
	}
}
