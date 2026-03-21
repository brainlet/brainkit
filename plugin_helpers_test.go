package brainkit

import (
	"os/exec"
	"testing"
)

func buildTestPlugin(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "/test-plugin"
	cmd := exec.Command("go", "build", "-o", binary, "./testdata/plugin/")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test plugin: %s\n%s", err, out)
	}
	return binary
}
