package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Message + TopicMatches
// ---------------------------------------------------------------------------

func TestMessageJSON(t *testing.T) {
	msg := Message{
		Version:  "v1",
		Topic:    "tools.call",
		Address:  "kit:staging/agent:coder-1",
		CallerID: "kit:prod",
		ID:       "msg-1",
		TraceID:  "trace-1",
		ParentID: "msg-0",
		Depth:    2,
		ReplyTo:  "_reply.abc",
		Payload:  json.RawMessage(`{"name":"echo"}`),
		Metadata: map[string]string{"x-request-id": "123"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Version != "v1" {
		t.Errorf("Version = %q, want v1", decoded.Version)
	}
	if decoded.Address != "kit:staging/agent:coder-1" {
		t.Errorf("Address = %q", decoded.Address)
	}
}

func TestTopicMatches(t *testing.T) {
	tests := []struct {
		pattern, topic string
		want           bool
	}{
		{"test.topic", "test.topic", true},
		{"test.*", "test.foo", true},
		{"test.*", "test.foo.bar", true},
		{"test.*", "other.nope", false},
		{"test.topic", "test.other", false},
	}
	for _, tt := range tests {
		got := TopicMatches(tt.pattern, tt.topic)
		if got != tt.want {
			t.Errorf("TopicMatches(%q, %q) = %v, want %v", tt.pattern, tt.topic, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Transport
// ---------------------------------------------------------------------------

func TestInProcessTransport_BroadcastDelivery(t *testing.T) {
	tr := NewInProcessTransport()
	defer tr.Close()

	received := make(chan Message, 2)

	tr.Subscribe(SubscriberInfo{
		ID:      "sub-1",
		Pattern: "test.*",
		Handler: func(msg Message) { received <- msg },
	})
	tr.Subscribe(SubscriberInfo{
		ID:      "sub-2",
		Pattern: "test.*",
		Handler: func(msg Message) { received <- msg },
	})

	tr.Publish(Message{Topic: "test.hello", CallerID: "c"})

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	count := 0
	for count < 2 {
		select {
		case <-received:
			count++
		case <-timer.C:
			t.Fatalf("got %d messages, want 2", count)
		}
	}
}

func TestInProcessTransport_WorkerGroup(t *testing.T) {
	tr := NewInProcessTransport()
	defer tr.Close()

	received := make(chan string, 100)

	for i := 0; i < 3; i++ {
		id := SubscriptionID(fmt.Sprintf("worker-%d", i))
		name := string(id)
		tr.Subscribe(SubscriberInfo{
			ID:      id,
			Pattern: "job.*",
			Group:   "workers",
			Handler: func(msg Message) { received <- name },
		})
	}

	for i := 0; i < 30; i++ {
		tr.Publish(Message{Topic: "job.process", CallerID: "c"})
	}

	// Drain 30 messages with timeout
	counts := map[string]int{}
	for i := 0; i < 30; i++ {
		select {
		case name := <-received:
			counts[name]++
		case <-time.After(2 * time.Second):
			t.Fatalf("only received %d messages, want 30", i)
		}
	}

	for name, c := range counts {
		if c == 0 {
			t.Errorf("worker %s got 0 messages", name)
		}
		t.Logf("worker %s: %d messages", name, c)
	}
}

func TestInProcessTransport_Unsubscribe(t *testing.T) {
	tr := NewInProcessTransport()
	defer tr.Close()

	var count atomic.Int32
	tr.Subscribe(SubscriberInfo{
		ID:      "sub-1",
		Pattern: "test.*",
		Handler: func(msg Message) { count.Add(1) },
	})

	tr.Publish(Message{Topic: "test.a", CallerID: "c"})
	time.Sleep(50 * time.Millisecond)

	tr.Unsubscribe("sub-1")

	tr.Publish(Message{Topic: "test.b", CallerID: "c"})
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("count = %d, want 1", count.Load())
	}
}

// ---------------------------------------------------------------------------
// Job Tracker
// ---------------------------------------------------------------------------

func TestJobTracker_CreateAndComplete(t *testing.T) {
	jt := newJobTracker(5*time.Second, time.Minute)
	defer jt.close()

	job := jt.getOrCreate("trace-1")
	if job.TraceID != "trace-1" {
		t.Errorf("TraceID = %q", job.TraceID)
	}
	if job.Status != "running" {
		t.Errorf("Status = %q, want running", job.Status)
	}

	jt.incrementPending("trace-1")
	jt.incrementPending("trace-1")

	job = jt.get("trace-1")
	if job.Pending != 2 {
		t.Errorf("Pending = %d, want 2", job.Pending)
	}

	jt.decrementPending("trace-1")
	jt.decrementPending("trace-1")

	job = jt.get("trace-1")
	if job.Status != "completed" {
		t.Errorf("Status = %q, want completed", job.Status)
	}
}

func TestJobTracker_Timeout(t *testing.T) {
	jt := newJobTracker(50*time.Millisecond, time.Minute)
	defer jt.close()

	jt.getOrCreate("trace-timeout")
	jt.incrementPending("trace-timeout")

	time.Sleep(200 * time.Millisecond)
	// Manually trigger eviction (ticker is 10s, too slow for test)
	jt.evict()

	job := jt.get("trace-timeout")
	if job == nil {
		t.Fatal("job should still exist")
	}
	if job.Status != "timeout" {
		t.Errorf("Status = %q, want timeout", job.Status)
	}
}

// ---------------------------------------------------------------------------
// Bus Core: Send / Ask / On / Off
// ---------------------------------------------------------------------------

func TestSendBroadcast(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	received := make(chan Message, 1)
	b.On("test.topic", func(msg Message, reply ReplyFunc) {
		received <- msg
	})

	b.Send(Message{Topic: "test.topic", CallerID: "test"})

	select {
	case msg := <-received:
		if msg.Topic != "test.topic" {
			t.Errorf("topic = %q", msg.Topic)
		}
		if msg.ID == "" {
			t.Error("expected auto-generated ID")
		}
		if msg.TraceID == "" {
			t.Error("expected auto-generated TraceID")
		}
		if msg.Version != "v1" {
			t.Errorf("version = %q", msg.Version)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestSendTopicPrefix(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	received := make(chan Message, 3)
	b.On("test.*", func(msg Message, reply ReplyFunc) {
		received <- msg
	})

	b.Send(Message{Topic: "test.one", CallerID: "c"})
	b.Send(Message{Topic: "test.two", CallerID: "c"})
	b.Send(Message{Topic: "other.nope", CallerID: "c"})

	time.Sleep(100 * time.Millisecond)

	if len(received) != 2 {
		t.Errorf("received %d messages, want 2", len(received))
	}
}

func TestAskReply(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	b.On("echo.*", func(msg Message, reply ReplyFunc) {
		reply(msg.Payload)
	})

	done := make(chan json.RawMessage, 1)
	b.Ask(Message{
		Topic:    "echo.test",
		CallerID: "test",
		Payload:  json.RawMessage(`{"ping":true}`),
	}, func(reply Message) {
		done <- reply.Payload
	})

	select {
	case payload := <-done:
		if string(payload) != `{"ping":true}` {
			t.Errorf("payload = %s", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestAskTimeout(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr, WithHandlerTimeout(50*time.Millisecond))
	defer b.Close()

	done := make(chan Message, 1)
	b.Ask(Message{Topic: "no.handler", CallerID: "test"}, func(reply Message) {
		done <- reply
	})

	select {
	case reply := <-done:
		var result struct {
			Error string `json:"error"`
		}
		json.Unmarshal(reply.Payload, &result)
		if result.Error == "" {
			t.Error("expected timeout error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("callback was never called")
	}
}

func TestAskCancel(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr, WithHandlerTimeout(5*time.Second))
	defer b.Close()

	called := false
	cancel := b.Ask(Message{Topic: "no.handler", CallerID: "test"}, func(reply Message) {
		called = true
	})

	cancel()
	time.Sleep(100 * time.Millisecond)

	if called {
		t.Error("callback should NOT have been called after cancel")
	}
}

func TestAskCascade(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	// Handler B: echo reply
	b.On("service.b", func(msg Message, reply ReplyFunc) {
		reply(json.RawMessage(`{"from":"B"}`))
	})

	// Handler A: calls B, then replies with combined result
	b.On("service.a", func(msg Message, reply ReplyFunc) {
		b.Ask(Message{
			Topic:    "service.b",
			CallerID: "handler-a",
		}, func(bReply Message) {
			reply(json.RawMessage(fmt.Sprintf(`{"from":"A","b":%s}`, bReply.Payload)))
		})
	})

	done := make(chan json.RawMessage, 1)
	b.Ask(Message{
		Topic:    "service.a",
		CallerID: "client",
	}, func(reply Message) {
		done <- reply.Payload
	})

	select {
	case payload := <-done:
		var result map[string]any
		json.Unmarshal(payload, &result)
		if result["from"] != "A" {
			t.Errorf("expected from=A, got %v", result["from"])
		}
		if result["b"] == nil {
			t.Error("expected nested B result")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cascade timeout")
	}
}

func TestAskInterceptorReject(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	b.AddInterceptor(&testInterceptor{
		name:      "blocker",
		pri:       0,
		matchFn:   func(topic string) bool { return topic == "blocked.topic" },
		processFn: func(msg *Message) error { return fmt.Errorf("unauthorized") },
	})

	done := make(chan Message, 1)
	b.Ask(Message{Topic: "blocked.topic", CallerID: "test"}, func(reply Message) {
		done <- reply
	})

	select {
	case reply := <-done:
		var result struct {
			Error string `json:"error"`
		}
		json.Unmarshal(reply.Payload, &result)
		if result.Error == "" || !strings.Contains(result.Error, "unauthorized") {
			t.Errorf("expected unauthorized error, got: %s", result.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("callback was never called")
	}
}

func TestOnWorkerGroup(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	received := make(chan string, 30)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("worker-%d", i)
		b.On("job.*", func(msg Message, reply ReplyFunc) {
			received <- name
		}, AsWorker("processors"))
	}

	for i := 0; i < 30; i++ {
		b.Send(Message{Topic: "job.do", CallerID: "test"})
	}

	time.Sleep(200 * time.Millisecond)
	if len(received) != 30 {
		t.Fatalf("got %d, want 30", len(received))
	}
}

func TestOff(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	var count atomic.Int32
	id := b.On("test.*", func(msg Message, reply ReplyFunc) { count.Add(1) })

	b.Send(Message{Topic: "test.a", CallerID: "c"})
	time.Sleep(50 * time.Millisecond)

	b.Off(id)

	b.Send(Message{Topic: "test.b", CallerID: "c"})
	time.Sleep(50 * time.Millisecond)

	if count.Load() != 1 {
		t.Errorf("count = %d, want 1", count.Load())
	}
}

func TestCycleDetection(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	err := b.Send(Message{Topic: "test", CallerID: "c", Depth: MaxDepth})
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestNameCollision(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	if err := b.RegisterName("prod"); err != nil {
		t.Fatal(err)
	}
	if err := b.RegisterName("prod"); err == nil {
		t.Error("expected collision error for duplicate name")
	}
	b.UnregisterName("prod")
	if err := b.RegisterName("prod"); err != nil {
		t.Fatalf("should succeed after unregister: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Interceptors
// ---------------------------------------------------------------------------

func TestInterceptorRuns(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	intercepted := false
	b.AddInterceptor(&testInterceptor{
		name:    "test",
		pri:     100,
		matchFn: func(string) bool { return true },
		processFn: func(msg *Message) error {
			intercepted = true
			return nil
		},
	})

	received := make(chan Message, 1)
	b.On("test.topic", func(msg Message, reply ReplyFunc) { received <- msg })
	b.Send(Message{Topic: "test.topic", CallerID: "c"})

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
	if !intercepted {
		t.Error("interceptor not called")
	}
}

func TestInterceptorReject(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	b.AddInterceptor(&testInterceptor{
		name:      "blocker",
		pri:       0,
		matchFn:   func(string) bool { return true },
		processFn: func(msg *Message) error { return fmt.Errorf("blocked") },
	})

	err := b.Send(Message{Topic: "test", CallerID: "c"})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected blocked error, got %v", err)
	}
}

func TestCallerIDImmutable(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	b.AddInterceptor(&testInterceptor{
		name:    "mutator",
		pri:     100,
		matchFn: func(string) bool { return true },
		processFn: func(msg *Message) error {
			msg.CallerID = "hacked"
			msg.Topic = "hacked"
			msg.Address = "hacked"
			return nil
		},
	})

	received := make(chan Message, 1)
	b.On("test.*", func(msg Message, reply ReplyFunc) { received <- msg })
	b.Send(Message{Topic: "test.topic", CallerID: "original", Address: "agent:x"})

	select {
	case msg := <-received:
		if msg.CallerID != "original" {
			t.Errorf("CallerID mutated to %q", msg.CallerID)
		}
		if msg.Topic != "test.topic" {
			t.Errorf("Topic mutated to %q", msg.Topic)
		}
		if msg.Address != "agent:x" {
			t.Errorf("Address mutated to %q", msg.Address)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

// ---------------------------------------------------------------------------
// AskSync
// ---------------------------------------------------------------------------

func TestAskSync(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr)
	defer b.Close()

	b.On("echo.*", func(msg Message, reply ReplyFunc) {
		reply(msg.Payload)
	})

	ctx := context.Background()
	reply, err := AskSync(b, ctx, Message{
		Topic:    "echo.test",
		CallerID: "test",
		Payload:  json.RawMessage(`{"sync":"yes"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(reply.Payload) != `{"sync":"yes"}` {
		t.Errorf("payload = %s", reply.Payload)
	}
}

func TestAskSync_ContextCancel(t *testing.T) {
	tr := NewInProcessTransport()
	b := NewBus(tr, WithHandlerTimeout(5*time.Second))
	defer b.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := AskSync(b, ctx, Message{Topic: "no.handler", CallerID: "test"})
	if err == nil {
		t.Fatal("expected context error")
	}
}

// ---------------------------------------------------------------------------
// Test Helpers
// ---------------------------------------------------------------------------

type testInterceptor struct {
	name      string
	pri       int
	matchFn   func(string) bool
	processFn func(*Message) error
}

func (i *testInterceptor) Name() string           { return i.name }
func (i *testInterceptor) Priority() int           { return i.pri }
func (i *testInterceptor) Match(topic string) bool { return i.matchFn(topic) }
func (i *testInterceptor) Process(msg *Message) error {
	return i.processFn(msg)
}
