package standard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/server/configfile"
)

func TestBareStandardDoesNotRegisterFullCatalog(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "brainkit.yaml")
	body := []byte(`namespace: bare-standard
transport:
  type: memory
modules:
  schedules: {}
`)
	if err := os.WriteFile(configPath, body, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := configfile.Load(configPath)
	if err == nil {
		t.Fatalf("Load succeeded; bare server/standard should not register schedules")
	}
	if !strings.Contains(err.Error(), "schedules") {
		t.Fatalf("Load error = %q, want schedules registration failure", err.Error())
	}
}
