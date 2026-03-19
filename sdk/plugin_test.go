package sdk

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit/sdk/messages"
)

func TestNew(t *testing.T) {
	p := New("brainlet", "test", "1.0.0")
	if p == nil {
		t.Fatal("New returned nil")
	}
	if p.owner != "brainlet" || p.name != "test" || p.version != "1.0.0" {
		t.Errorf("fields = %s/%s@%s, want brainlet/test@1.0.0", p.owner, p.name, p.version)
	}
}

func TestWithDescription(t *testing.T) {
	p := New("x", "y", "1.0.0", WithDescription("A test plugin"))
	if p.description != "A test plugin" {
		t.Errorf("description = %q", p.description)
	}
}

type testInput struct {
	Value string `json:"value"`
}
type testOutput struct {
	Result string `json:"result"`
}

func TestToolRegistration(t *testing.T) {
	p := New("o", "n", "1.0.0")
	Tool(p, "echo", "Echo tool", func(ctx context.Context, client Client, in testInput) (testOutput, error) {
		return testOutput{Result: in.Value}, nil
	})

	if len(p.tools) != 1 {
		t.Fatalf("tools count = %d, want 1", len(p.tools))
	}
	if p.tools[0].name != "echo" {
		t.Errorf("tool name = %q", p.tools[0].name)
	}
	if p.tools[0].inputSchema == "" {
		t.Error("inputSchema is empty")
	}
}

type testEvent struct {
	Data string `json:"data"`
}

func (testEvent) BusTopic() string { return "test.event" }

func TestOnRegistration(t *testing.T) {
	p := New("o", "n", "1.0.0")
	On[testEvent](p, "test.*", func(ctx context.Context, event testEvent, client Client, reply messages.ReplyFunc) {})

	if len(p.subscriptions) != 1 {
		t.Fatalf("subscriptions count = %d, want 1", len(p.subscriptions))
	}
	if p.subscriptions[0].topic != "test.*" {
		t.Errorf("topic = %q", p.subscriptions[0].topic)
	}
}

func TestEventRegistration(t *testing.T) {
	p := New("o", "n", "1.0.0")
	Event[testEvent](p, "A test event")

	if len(p.events) != 1 {
		t.Fatalf("events count = %d, want 1", len(p.events))
	}
	if p.events[0].name != "test.event" {
		t.Errorf("event name = %q", p.events[0].name)
	}
	if p.events[0].description != "A test event" {
		t.Errorf("event description = %q", p.events[0].description)
	}
}

func TestInterceptRegistration(t *testing.T) {
	p := New("o", "n", "1.0.0")
	Intercept(p, "audit", 200, "tools.*", func(ctx context.Context, msg InterceptMessage) (*InterceptMessage, error) {
		return &msg, nil
	})

	if len(p.interceptors) != 1 {
		t.Fatalf("interceptors count = %d, want 1", len(p.interceptors))
	}
	if p.interceptors[0].name != "audit" {
		t.Errorf("name = %q", p.interceptors[0].name)
	}
	if p.interceptors[0].priority != 200 {
		t.Errorf("priority = %d", p.interceptors[0].priority)
	}
}

func TestBuildManifest(t *testing.T) {
	p := New("brainlet", "cron", "1.0.0", WithDescription("Cron plugin"))

	Tool(p, "create", "Create job", func(ctx context.Context, client Client, in testInput) (testOutput, error) {
		return testOutput{}, nil
	})
	On[testEvent](p, "cron.*", func(ctx context.Context, event testEvent, client Client, reply messages.ReplyFunc) {})
	Event[testEvent](p, "Cron fired")
	Intercept(p, "audit", 100, "tools.*", func(ctx context.Context, msg InterceptMessage) (*InterceptMessage, error) {
		return &msg, nil
	})

	m := p.buildManifest()

	if m.Owner != "brainlet" {
		t.Errorf("owner = %q", m.Owner)
	}
	if m.Name != "cron" {
		t.Errorf("name = %q", m.Name)
	}
	if len(m.Tools) != 1 {
		t.Errorf("tools = %d", len(m.Tools))
	}
	if len(m.Subscriptions) != 1 {
		t.Errorf("subscriptions = %d", len(m.Subscriptions))
	}
	if len(m.Events) != 1 {
		t.Errorf("events = %d", len(m.Events))
	}
	if len(m.Interceptors) != 1 {
		t.Errorf("interceptors = %d", len(m.Interceptors))
	}
}

func TestBuildManifestEmpty(t *testing.T) {
	p := New("x", "y", "1.0.0")
	m := p.buildManifest()

	if m.Owner != "x" || m.Name != "y" {
		t.Errorf("identity wrong")
	}
	if len(m.Tools) != 0 || len(m.Subscriptions) != 0 || len(m.Events) != 0 || len(m.Interceptors) != 0 {
		t.Error("empty plugin should have no registrations")
	}
}

func TestOnStartOnStop(t *testing.T) {
	p := New("o", "n", "1.0.0")

	started := false
	stopped := false

	p.OnStart(func(client Client) error {
		started = true
		return nil
	})
	p.OnStop(func() error {
		stopped = true
		return nil
	})

	if p.onStartFn == nil || p.onStopFn == nil {
		t.Fatal("callbacks not registered")
	}
	p.onStartFn(nil)
	p.onStopFn()

	if !started {
		t.Error("onStart not called")
	}
	if !stopped {
		t.Error("onStop not called")
	}
}
