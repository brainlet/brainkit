package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
	_ "github.com/brainlet/brainkit/server/configfile/storebackends/sqlite"
)

func TestServerProfileRegistersGatewayAndProbes(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: server-profile
fs_root: ` + tmp + `
transport:
  type: memory
modules:
  gateway:
    listen: ":0"
  probes: {}
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
	for _, want := range []string{"gateway", "probes"} {
		if !got[want] {
			t.Fatalf("loaded modules = %#v, missing %s", got, want)
		}
	}
}
