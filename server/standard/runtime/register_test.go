package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
)

func TestRuntimeProfileRegistersRuntimeModulesOnly(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: runtime-profile
fs_root: ` + tmp + `
transport:
  type: memory
modules:
  jsruntime: {}
  eval: {}
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
	for _, want := range []string{"jsruntime", "eval"} {
		if !got[want] {
			t.Fatalf("loaded modules = %#v, missing %s", got, want)
		}
	}
	if got["packages"] {
		t.Fatalf("runtime profile must not register packages module: %#v", got)
	}
}
