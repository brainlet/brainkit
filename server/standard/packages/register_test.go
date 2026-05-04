package packages

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
)

func TestPackagesProfileRegistersRuntimeAndPackagesModules(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: packages-profile
fs_root: ` + tmp + `
transport:
  type: memory
modules:
  jsruntime: {}
  eval: {}
  packages: {}
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
	for _, want := range []string{"jsruntime", "eval", "packages"} {
		if !got[want] {
			t.Fatalf("loaded modules = %#v, missing %s", got, want)
		}
	}
}
