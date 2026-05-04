package transport_test

import (
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/transport"
)

func TestDepthMiddleware_AllowsNormalDepth(t *testing.T) {
	handler := transport.DepthMiddleware(func(msg *transport.Message) ([]*transport.Message, error) {
		return nil, nil
	})

	msg := transport.NewMessage([]byte("{}"))
	msg.Metadata.Set("depth", "5")

	_, err := handler(msg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDepthMiddleware_RejectsCycle(t *testing.T) {
	handler := transport.DepthMiddleware(func(msg *transport.Message) ([]*transport.Message, error) {
		return nil, nil
	})

	msg := transport.NewMessage([]byte("{}"))
	msg.Metadata.Set("depth", "16")

	_, err := handler(msg)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDepthMiddleware_NoDepthHeader(t *testing.T) {
	handler := transport.DepthMiddleware(func(msg *transport.Message) ([]*transport.Message, error) {
		return nil, nil
	})

	msg := transport.NewMessage([]byte("{}"))
	_, err := handler(msg)
	if err != nil {
		t.Fatalf("expected no error without depth header, got: %v", err)
	}
}

func TestCallerIDMiddleware_StampsDefault(t *testing.T) {
	mw := transport.CallerIDMiddleware("default-kit")
	handler := mw(func(msg *transport.Message) ([]*transport.Message, error) {
		return nil, nil
	})

	msg := transport.NewMessage([]byte("{}"))
	handler(msg)

	if msg.Metadata.Get("callerId") != "default-kit" {
		t.Errorf("callerId = %q, want 'default-kit'", msg.Metadata.Get("callerId"))
	}
}

func TestCallerIDMiddleware_DoesNotOverwrite(t *testing.T) {
	mw := transport.CallerIDMiddleware("default-kit")
	handler := mw(func(msg *transport.Message) ([]*transport.Message, error) {
		return nil, nil
	})

	msg := transport.NewMessage([]byte("{}"))
	msg.Metadata.Set("callerId", "original-caller")
	handler(msg)

	if msg.Metadata.Get("callerId") != "original-caller" {
		t.Errorf("callerId = %q, want 'original-caller'", msg.Metadata.Get("callerId"))
	}
}

func TestMetrics_SnapshotIsIsolated(t *testing.T) {
	m := transport.NewMetrics()
	m.Published("test.topic")
	m.Published("test.topic")
	m.Record("test.topic", 10*time.Millisecond, nil)
	m.Record("test.topic", 25*time.Millisecond, assertErr{})

	snap := m.Snapshot()
	if snap.Published["test.topic"] != 2 {
		t.Errorf("published = %d, want 2", snap.Published["test.topic"])
	}
	if snap.Handled["test.topic"] != 2 {
		t.Errorf("handled = %d, want 2", snap.Handled["test.topic"])
	}
	if snap.Errors["test.topic"] != 1 {
		t.Errorf("errors = %d, want 1", snap.Errors["test.topic"])
	}
	if snap.HandleDurationCount["test.topic"] != 2 {
		t.Errorf("handle duration count = %d, want 2", snap.HandleDurationCount["test.topic"])
	}
	if snap.HandleDurationTotal["test.topic"] != 35*time.Millisecond {
		t.Errorf("handle duration total = %s, want 35ms", snap.HandleDurationTotal["test.topic"])
	}
	if snap.HandleDurationMax["test.topic"] != 25*time.Millisecond {
		t.Errorf("handle duration max = %s, want 25ms", snap.HandleDurationMax["test.topic"])
	}

	// Mutating snapshot should not affect original
	snap.Published["test.topic"] = 999
	snap.HandleDurationCount["test.topic"] = 999
	snap2 := m.Snapshot()
	if snap2.Published["test.topic"] != 2 {
		t.Errorf("snapshot mutation leaked: %d", snap2.Published["test.topic"])
	}
	if snap2.HandleDurationCount["test.topic"] != 2 {
		t.Errorf("duration count mutation leaked: %d", snap2.HandleDurationCount["test.topic"])
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "assert" }
