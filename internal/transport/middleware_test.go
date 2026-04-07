package transport_test

import (
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/transport"
)

func TestDepthMiddleware_AllowsNormalDepth(t *testing.T) {
	handler := messaging.DepthMiddleware(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
	msg.Metadata.Set("depth", "5")

	_, err := handler(msg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDepthMiddleware_RejectsCycle(t *testing.T) {
	handler := messaging.DepthMiddleware(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
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
	handler := messaging.DepthMiddleware(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
	_, err := handler(msg)
	if err != nil {
		t.Fatalf("expected no error without depth header, got: %v", err)
	}
}

func TestCallerIDMiddleware_StampsDefault(t *testing.T) {
	mw := messaging.CallerIDMiddleware("default-kit")
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
	handler(msg)

	if msg.Metadata.Get("callerId") != "default-kit" {
		t.Errorf("callerId = %q, want 'default-kit'", msg.Metadata.Get("callerId"))
	}
}

func TestCallerIDMiddleware_DoesNotOverwrite(t *testing.T) {
	mw := messaging.CallerIDMiddleware("default-kit")
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage(watermill.NewUUID(), []byte("{}"))
	msg.Metadata.Set("callerId", "original-caller")
	handler(msg)

	if msg.Metadata.Get("callerId") != "original-caller" {
		t.Errorf("callerId = %q, want 'original-caller'", msg.Metadata.Get("callerId"))
	}
}

func TestMetrics_SnapshotIsIsolated(t *testing.T) {
	m := messaging.NewMetrics()
	m.Published("test.topic")
	m.Published("test.topic")
	m.Record("test.topic", 0, nil)

	snap := m.Snapshot()
	if snap.Published["test.topic"] != 2 {
		t.Errorf("published = %d, want 2", snap.Published["test.topic"])
	}
	if snap.Handled["test.topic"] != 1 {
		t.Errorf("handled = %d, want 1", snap.Handled["test.topic"])
	}

	// Mutating snapshot should not affect original
	snap.Published["test.topic"] = 999
	snap2 := m.Snapshot()
	if snap2.Published["test.topic"] != 2 {
		t.Errorf("snapshot mutation leaked: %d", snap2.Published["test.topic"])
	}
}
