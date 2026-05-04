package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
)

func TestCommandsProfileRegistersCommandRuntimeModules(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: commands-profile
fs_root: ` + tmp + `
transport:
  type: memory
modules:
  jsruntime: {}
  agents: {}
  packages: {}
  tools: {}
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := configfile.Load(configPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := map[string]bool{}
	for _, mod := range cfg.Modules {
		got[mod.ID()] = true
	}
	for _, want := range []string{"agents", "jsruntime", "packages", "tools"} {
		if !got[want] {
			t.Fatalf("loaded modules = %#v, missing %s", got, want)
		}
	}
}
