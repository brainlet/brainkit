package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSendAndSubscribe(t *testing.T) {
	b := New()
	defer b.Close()

	received := make(chan Message, 1)
	_, err := b.Subscribe("test.topic", func(msg Message) {
		received <- msg
	})
	if err != nil {
		t.Fatal(err)
	}

	err = b.Send(context.Background(), Message{
		Topic:    "test.topic",
		CallerID: "test.caller",
		Payload:  json.RawMessage(`{"hello":"world"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-received:
		if msg.Topic != "test.topic" {
			t.Errorf("topic = %q, want test.topic", msg.Topic)
		}
		if msg.CallerID != "test.caller" {
			t.Errorf("callerID = %q, want test.caller", msg.CallerID)
		}
		if msg.ID == "" {
			t.Error("expected non-empty message ID")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestSubscribeTopicPrefix(t *testing.T) {
	b := New()
	defer b.Close()

	received := make(chan Message, 3)
	b.Subscribe("test.*", func(msg Message) {
		received <- msg
	})

	b.Send(context.Background(), Message{Topic: "test.one", CallerID: "c"})
	b.Send(context.Background(), Message{Topic: "test.two", CallerID: "c"})
	b.Send(context.Background(), Message{Topic: "other.nope", CallerID: "c"})

	time.Sleep(50 * time.Millisecond)

	if len(received) != 2 {
		t.Errorf("received %d messages, want 2", len(received))
	}
}

func TestRequest(t *testing.T) {
	b := New()
	defer b.Close()

	b.Handle("echo.*", func(ctx context.Context, msg Message) (*Message, error) {
		return &Message{
			Topic:   msg.ReplyTo,
			Payload: msg.Payload,
		}, nil
	})

	resp, err := b.Request(context.Background(), "echo.test", "test.caller",
		json.RawMessage(`{"data":"ping"}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Payload) != `{"data":"ping"}` {
		t.Errorf("payload = %s, want ping", resp.Payload)
	}
}

func TestRequestTimeout(t *testing.T) {
	b := New()
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := b.Request(ctx, "nohandler.topic", "test.caller", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestCycleDetection(t *testing.T) {
	b := New()
	defer b.Close()

	msg := Message{
		Topic:    "test.topic",
		CallerID: "test",
		Depth:    MaxDepth,
	}
	err := b.Send(context.Background(), msg)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestInterceptorRuns(t *testing.T) {
	b := New()
	defer b.Close()

	var intercepted bool
	b.AddInterceptor(&testInterceptor{
		name:     "test",
		pri:      100,
		matchFn:  func(topic string) bool { return true },
		processFn: func(ctx context.Context, msg *Message) error {
			intercepted = true
			return nil
		},
	})

	received := make(chan Message, 1)
	b.Subscribe("test.topic", func(msg Message) { received <- msg })
	b.Send(context.Background(), Message{Topic: "test.topic", CallerID: "c"})

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	if !intercepted {
		t.Error("interceptor was not called")
	}
}

func TestInterceptorReject(t *testing.T) {
	b := New()
	defer b.Close()

	b.AddInterceptor(&testInterceptor{
		name:    "blocker",
		pri:     0,
		matchFn: func(topic string) bool { return true },
		processFn: func(ctx context.Context, msg *Message) error {
			return fmt.Errorf("blocked")
		},
	})

	err := b.Send(context.Background(), Message{Topic: "test.topic", CallerID: "c"})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected blocked error, got %v", err)
	}
}

func TestUnsubscribe(t *testing.T) {
	b := New()
	defer b.Close()

	count := 0
	sub, _ := b.Subscribe("test.*", func(msg Message) { count++ })

	b.Send(context.Background(), Message{Topic: "test.a", CallerID: "c"})
	time.Sleep(20 * time.Millisecond)

	b.Unsubscribe(sub)

	b.Send(context.Background(), Message{Topic: "test.b", CallerID: "c"})
	time.Sleep(20 * time.Millisecond)

	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestCallerIDImmutable(t *testing.T) {
	b := New()
	defer b.Close()

	b.AddInterceptor(&testInterceptor{
		name:    "mutator",
		pri:     100,
		matchFn: func(string) bool { return true },
		processFn: func(ctx context.Context, msg *Message) error {
			msg.CallerID = "hacked"
			msg.Topic = "hacked.topic"
			return nil
		},
	})

	received := make(chan Message, 1)
	b.Subscribe("test.*", func(msg Message) { received <- msg })
	b.Send(context.Background(), Message{Topic: "test.topic", CallerID: "original"})

	select {
	case msg := <-received:
		if msg.CallerID != "original" {
			t.Errorf("CallerID was mutated to %q", msg.CallerID)
		}
		if msg.Topic != "test.topic" {
			t.Errorf("Topic was mutated to %q", msg.Topic)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// --- test helpers ---

type testInterceptor struct {
	name      string
	pri       int
	matchFn   func(string) bool
	processFn func(context.Context, *Message) error
}

func (i *testInterceptor) Name() string                                    { return i.name }
func (i *testInterceptor) Priority() int                                   { return i.pri }
func (i *testInterceptor) Match(topic string) bool                         { return i.matchFn(topic) }
func (i *testInterceptor) Process(ctx context.Context, msg *Message) error { return i.processFn(ctx, msg) }

func TestHandlerTimeout(t *testing.T) {
	b := NewWithTimeout(100 * time.Millisecond) // very short timeout for testing
	defer b.Close()

	// Register a handler that takes too long
	b.Handle("slow.*", func(ctx context.Context, msg Message) (*Message, error) {
		select {
		case <-time.After(5 * time.Second): // way longer than timeout
			return &Message{Payload: json.RawMessage(`{"ok":true}`)}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	reply, err := b.Request(ctx, "slow.op", "test", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	// Should get a timeout error in the response payload
	var result struct {
		Error string `json:"error"`
	}
	json.Unmarshal(reply.Payload, &result)
	if result.Error == "" {
		t.Fatal("expected timeout error in reply")
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("expected 'timeout' in error, got: %s", result.Error)
	}
	t.Logf("timeout error: %s", result.Error)
}

func TestHandlerTimeout_FastHandler(t *testing.T) {
	b := NewWithTimeout(5 * time.Second)
	defer b.Close()

	// Fast handler — should NOT timeout
	b.Handle("fast.*", func(ctx context.Context, msg Message) (*Message, error) {
		return &Message{Payload: json.RawMessage(`{"fast":true}`)}, nil
	})

	ctx := context.Background()
	reply, err := b.Request(ctx, "fast.op", "test", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Request: %v", err)
	}

	var result struct {
		Fast  bool   `json:"fast"`
		Error string `json:"error"`
	}
	json.Unmarshal(reply.Payload, &result)
	if result.Error != "" {
		t.Errorf("expected no error, got: %s", result.Error)
	}
	if !result.Fast {
		t.Error("expected fast=true")
	}
}

func TestHandlerTimeout_Configurable(t *testing.T) {
	// Default timeout
	b1 := New()
	if b1.HandlerTimeout != DefaultHandlerTimeout {
		t.Errorf("default timeout = %v, want %v", b1.HandlerTimeout, DefaultHandlerTimeout)
	}

	// Custom timeout
	b2 := NewWithTimeout(10 * time.Second)
	if b2.HandlerTimeout != 10*time.Second {
		t.Errorf("custom timeout = %v, want 10s", b2.HandlerTimeout)
	}

	// Can change at runtime
	b1.HandlerTimeout = 5 * time.Second
	if b1.HandlerTimeout != 5*time.Second {
		t.Error("runtime change failed")
	}
}
