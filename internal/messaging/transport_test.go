package messaging_test

import (
	"testing"

	"github.com/brainlet/brainkit/internal/messaging"
)

func TestNewTransport_Memory(t *testing.T) {
	pub, sub, err := messaging.NewTransport(messaging.TransportConfig{Type: "memory"})
	if err != nil {
		t.Fatalf("memory transport: %v", err)
	}
	if pub == nil || sub == nil {
		t.Fatal("pub or sub is nil")
	}
}

func TestNewTransport_Default(t *testing.T) {
	pub, sub, err := messaging.NewTransport(messaging.TransportConfig{})
	if err != nil {
		t.Fatalf("default transport: %v", err)
	}
	if pub == nil || sub == nil {
		t.Fatal("pub or sub is nil")
	}
}

func TestNewTransport_UnknownType(t *testing.T) {
	_, _, err := messaging.NewTransport(messaging.TransportConfig{Type: "invalid"})
	if err == nil {
		t.Fatal("expected error for unknown transport type")
	}
}

func TestNewTransport_UnsupportedLegacyTypes(t *testing.T) {
	for _, tp := range []string{"legacy-one", "legacy-two"} {
		_, _, err := messaging.NewTransport(messaging.TransportConfig{Type: tp})
		if err == nil {
			t.Fatalf("expected error for unsupported transport %q", tp)
		}
	}
}
