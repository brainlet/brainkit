package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

type mockPluginChecker struct {
	running map[string]bool
}

func (m *mockPluginChecker) IsPluginRunning(name string) bool { return m.running[name] }

type mockSecretChecker struct {
	secrets map[string]bool
}

func (m *mockSecretChecker) HasSecret(name string) bool { return m.secrets[name] }

func TestValidateDepsFailsForMissingPlugin(t *testing.T) {
	err := ValidateDeps(PackageManifest{
		Name: "needs-plugin",
		Requires: &Requirements{
			Plugins: []string{"brainlet/telegram-gateway@>=1.0.0"},
		},
	}, &mockPluginChecker{running: map[string]bool{}}, &mockSecretChecker{secrets: map[string]bool{}})
	if err == nil {
		t.Fatal("expected missing plugin dependency error")
	}
}

func TestValidateDepsFailsForMissingSecret(t *testing.T) {
	err := ValidateDeps(PackageManifest{
		Name: "needs-secret",
		Requires: &Requirements{
			Secrets: []string{"BOT_TOKEN"},
		},
	}, &mockPluginChecker{running: map[string]bool{}}, &mockSecretChecker{secrets: map[string]bool{}})
	if err == nil {
		t.Fatal("expected missing secret dependency error")
	}
}

func TestValidateDepsPasses(t *testing.T) {
	err := ValidateDeps(PackageManifest{
		Name: "full-deps",
		Requires: &Requirements{
			Plugins: []string{"brainlet/telegram-gateway@>=1.0.0"},
			Secrets: []string{"BOT_TOKEN"},
		},
	}, &mockPluginChecker{running: map[string]bool{"telegram-gateway": true}}, &mockSecretChecker{secrets: map[string]bool{"BOT_TOKEN": true}})
	if err != nil {
		t.Fatalf("ValidateDeps: %v", err)
	}
}

func TestResolveEntryExplicitEntry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.ts", `output("main");`)

	path, err := ResolveEntry(dir, PackageManifest{Entry: "main.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "main.ts" {
		t.Fatalf("entry = %s, want main.ts", path)
	}
}

func TestResolveEntryIndexFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `output("index");`)
	writeFile(t, dir, "other.ts", `output("other");`)

	path, err := ResolveEntry(dir, PackageManifest{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "index.ts" {
		t.Fatalf("entry = %s, want index.ts", path)
	}
}

func TestResolveEntryOnlyTSFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "solo.ts", `output("solo");`)

	path, err := ResolveEntry(dir, PackageManifest{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "solo.ts" {
		t.Fatalf("entry = %s, want solo.ts", path)
	}
}

func TestResolveEntryRejectsNoTSFiles(t *testing.T) {
	_, err := ResolveEntry(t.TempDir(), PackageManifest{})
	if err == nil {
		t.Fatal("expected no .ts files error")
	}
}

func TestResolveEntryRejectsMultipleTSFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.ts", `output("a");`)
	writeFile(t, dir, "b.ts", `output("b");`)

	_, err := ResolveEntry(dir, PackageManifest{})
	if err == nil {
		t.Fatal("expected multiple .ts files error")
	}
}
