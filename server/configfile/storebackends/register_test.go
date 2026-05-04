package storebackends

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
)

func TestRegisterAllKitStoreBackends(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: standard-store
fs_root: ` + tmp + `
transport:
  type: memory
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := configfile.Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Store == nil {
		t.Fatal("Store is nil, want SQLite store")
	}
	t.Cleanup(func() { _ = cfg.Store.Close() })
}
