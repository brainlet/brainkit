package transportbackends

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
)

func TestRegisterAllTransportBackends(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: standard-transport
transport:
  type: nats
  url: nats://example.invalid:4222
  nats_name: standard-test
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := configfile.Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := cfg.Transport.Kind(); got != "nats" {
		t.Fatalf("transport kind = %q, want nats", got)
	}
	if got := cfg.Transport.NATSURL(); got != "nats://example.invalid:4222" {
		t.Fatalf("nats url = %q", got)
	}
	if got := cfg.Transport.NATSName(); got != "standard-test" {
		t.Fatalf("nats name = %q", got)
	}
}
