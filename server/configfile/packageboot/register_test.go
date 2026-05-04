package packageboot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
)

func TestPackageBootRegistersPackageLoader(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: package-loader
transport:
  type: memory
packages:
  - path: ./missing
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := configfile.Load(configPath)
	if err == nil {
		t.Fatal("expected missing package path error")
	}
	if !strings.Contains(err.Error(), "server: load package") {
		t.Fatalf("error = %q, want registered package loader error", err.Error())
	}
	if strings.Contains(err.Error(), "server/configfile/packageboot") {
		t.Fatalf("error = %q, packageboot registration hint should not appear after registration", err.Error())
	}
}
