package messaging_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/brainlet/brainkit/internal/messaging"
)

// testMsg is a simple message type for testing.
type testMsg struct {
	Value string `json:"value"`
}

func (testMsg) BusTopic() string { return "test.topic" }

func TestPublishHandle_RoundTrip(t *testing.T) {
	logger := watermill.NopLogger{}
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		t.Fatalf("new router: %v", err)
	}

	var received testMsg
	var wg sync.WaitGroup
	wg.Add(1)

	messaging.Handle[testMsg](router, pubSub, func(ctx context.Context, env messaging.Envelope[testMsg]) error {
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

	err = messaging.Publish(pubSub, testMsg{Value: "hello"}, "test-caller")
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
	wmsg := message.NewMessage(watermill.NewUUID(), payload)
	wmsg.Metadata.Set("callerId", "test-caller")
	wmsg.Metadata.Set("traceId", "trace-123")

	env, err := messaging.DecodeEnvelope[testMsg](wmsg)
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
	wmsg := message.NewMessage(watermill.NewUUID(), []byte("not json"))
	_, err := messaging.DecodeEnvelope[testMsg](wmsg)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPublish_StampsCallerID(t *testing.T) {
	logger := watermill.NopLogger{}
	pubSub := gochannel.NewGoChannel(gochannel.Config{}, logger)
	router, _ := message.NewRouter(message.RouterConfig{}, logger)

	var gotCallerID string
	var wg sync.WaitGroup
	wg.Add(1)

	messaging.Handle[testMsg](router, pubSub, func(ctx context.Context, env messaging.Envelope[testMsg]) error {
		gotCallerID = env.Raw.Metadata.Get("callerId")
		wg.Done()
		return nil
	})

	go router.Run(context.Background())
	defer router.Close()
	<-router.Running()

	messaging.Publish(pubSub, testMsg{Value: "x"}, "my-kit")
	wg.Wait()

	if gotCallerID != "my-kit" {
		t.Errorf("callerID = %q, want 'my-kit'", gotCallerID)
	}
}
