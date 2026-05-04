package standard

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
)

func TestConfigFileLoadUsesStandardSchedulesFactory(t *testing.T) {
	tmp := t.TempDir()
	schedulePath := filepath.Join(tmp, "schedules.db")
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: schedules-standard
fs_root: ` + tmp + `
transport:
  type: memory
modules:
  schedules:
    path: ` + schedulePath + `
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := configfile.Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Store != nil {
		t.Cleanup(func() { _ = cfg.Store.Close() })
	}
	if len(cfg.Modules) != 1 || cfg.Modules[0].ID() != "schedules" {
		t.Fatalf("modules = %#v, want one schedules module", cfg.Modules)
	}
	if _, err := os.Stat(schedulePath); err != nil {
		t.Fatalf("stat schedule db: %v", err)
	}
	if closer, ok := cfg.Modules[0].(interface {
		CloseContext(context.Context) error
	}); ok {
		if err := closer.CloseContext(context.Background()); err != nil {
			t.Fatalf("close schedules module: %v", err)
		}
	}
}
