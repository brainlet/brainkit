package configfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransportBuildMemoryIsBuiltIn(t *testing.T) {
	cfg, err := (TransportYAML{Type: "memory"}).build()
	if err != nil {
		t.Fatalf("build memory: %v", err)
	}
	if got := cfg.Kind(); got != "memory" {
		t.Fatalf("kind = %q, want memory", got)
	}
}

func TestTransportBuildRequiresRegisteredBackend(t *testing.T) {
	_, err := (TransportYAML{Type: "embedded"}).build()
	if err == nil {
		t.Fatal("expected unregistered transport error")
	}
	if !strings.Contains(err.Error(), "server/configfile/transportbackends") {
		t.Fatalf("error = %q, want configfile transport backend hint", err.Error())
	}
}

func TestLoadRequiresRegisteredKitStoreBackend(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: store-backend
fs_root: ` + tmp + `
transport:
  type: memory
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected unregistered kit store backend error")
	}
	if !strings.Contains(err.Error(), "server/configfile/storebackends") {
		t.Fatalf("error = %q, want configfile store backend hint", err.Error())
	}
}

func TestLoadRequiresRegisteredPackageLoader(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: package-loader
transport:
  type: memory
packages:
  - path: ./pkg
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("expected unregistered package loader error")
	}
	if !strings.Contains(err.Error(), "server/configfile/packageboot") {
		t.Fatalf("error = %q, want configfile packageboot hint", err.Error())
	}
}
