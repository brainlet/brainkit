package deploy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// mockDeployer tracks deploy/teardown calls without a real Kernel.
type mockDeployer struct {
	deployed map[string]string // source → code
	tornDown []string
}

func newMockDeployer() *mockDeployer {
	return &mockDeployer{deployed: make(map[string]string)}
}

func (m *mockDeployer) Deploy(_ context.Context, source, code string) error {
	m.deployed[source] = code
	return nil
}

func (m *mockDeployer) Teardown(_ context.Context, source string) error {
	delete(m.deployed, source)
	m.tornDown = append(m.tornDown, source)
	return nil
}

type mockPluginChecker struct {
	running map[string]bool
}

func (m *mockPluginChecker) IsPluginRunning(name string) bool { return m.running[name] }

type mockSecretChecker struct {
	secrets map[string]bool
}

func (m *mockSecretChecker) HasSecret(name string) bool { return m.secrets[name] }

func writeManifest(t *testing.T, dir string, manifest PackageManifest) {
	t.Helper()
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644)
}

func TestDeployPackage_Basic(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "config.ts", `export const CONFIG = { name: "test" };`)
	writeFile(t, dir, "index.ts", `
		import { CONFIG } from "./config";
		console.log(CONFIG.name);
	`)

	writeManifest(t, dir, PackageManifest{
		Name:    "test-pkg",
		Version: "1.0.0",
		Entry:   "index.ts",
	})

	deployer := newMockDeployer()
	pkg, err := DeployPackage(context.Background(), deployer, dir, nil, nil)
	if err != nil {
		t.Fatal("deploy:", err)
	}

	if pkg.Name != "test-pkg" {
		t.Fatalf("expected name 'test-pkg', got %q", pkg.Name)
	}
	if pkg.Source != "test-pkg.ts" {
		t.Fatalf("expected source 'test-pkg.ts', got %q", pkg.Source)
	}

	// Verify bundled code was passed to deployer
	code, ok := deployer.deployed["test-pkg.ts"]
	if !ok {
		t.Fatal("source not deployed")
	}
	if code == "" {
		t.Fatal("deployed code is empty")
	}
}

func TestDeployPackage_IndexTsConvention(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "index.ts", `console.log("hello");`)
	// No entry in manifest — should find index.ts by convention
	writeManifest(t, dir, PackageManifest{
		Name:    "convention",
		Version: "1.0.0",
	})

	deployer := newMockDeployer()
	pkg, err := DeployPackage(context.Background(), deployer, dir, nil, nil)
	if err != nil {
		t.Fatal("deploy:", err)
	}

	if pkg.Source != "convention.ts" {
		t.Fatalf("expected source 'convention.ts', got %q", pkg.Source)
	}
	if _, ok := deployer.deployed["convention.ts"]; !ok {
		t.Fatal("source not deployed")
	}
}

func TestDeployPackage_MissingEntry(t *testing.T) {
	dir := t.TempDir()

	writeManifest(t, dir, PackageManifest{
		Name:    "broken",
		Version: "1.0.0",
		Entry:   "nonexistent.ts",
	})

	deployer := newMockDeployer()
	_, err := DeployPackage(context.Background(), deployer, dir, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestDeployPackage_DependencyCheckFails(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "index.ts", `console.log("hi");`)
	writeManifest(t, dir, PackageManifest{
		Name:    "needs-plugin",
		Version: "1.0.0",
		Requires: &Requirements{
			Plugins: []string{"brainlet/telegram-gateway@>=1.0.0"},
		},
	})

	plugins := &mockPluginChecker{
		running: map[string]bool{},
	}
	secrets := &mockSecretChecker{secrets: map[string]bool{}}

	deployer := newMockDeployer()
	_, err := DeployPackage(context.Background(), deployer, dir, plugins, secrets)
	if err == nil {
		t.Fatal("expected error for missing plugin dependency")
	}
}

func TestDeployPackage_SecretCheckFails(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "index.ts", `console.log("hi");`)
	writeManifest(t, dir, PackageManifest{
		Name:    "needs-secret",
		Version: "1.0.0",
		Requires: &Requirements{
			Secrets: []string{"MY_TOKEN"},
		},
	})

	plugins := &mockPluginChecker{running: map[string]bool{}}
	secrets := &mockSecretChecker{secrets: map[string]bool{}} // MY_TOKEN not set

	deployer := newMockDeployer()
	_, err := DeployPackage(context.Background(), deployer, dir, plugins, secrets)
	if err == nil {
		t.Fatal("expected error for missing secret dependency")
	}
}

func TestDeployPackage_DepsPass(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "index.ts", `console.log("ok");`)
	writeManifest(t, dir, PackageManifest{
		Name:    "full-deps",
		Version: "1.0.0",
		Requires: &Requirements{
			Plugins: []string{"brainlet/telegram-gateway@>=1.0.0"},
			Secrets: []string{"BOT_TOKEN"},
		},
	})

	plugins := &mockPluginChecker{
		running: map[string]bool{"telegram-gateway": true},
	}
	secrets := &mockSecretChecker{secrets: map[string]bool{"BOT_TOKEN": true}}

	deployer := newMockDeployer()
	pkg, err := DeployPackage(context.Background(), deployer, dir, plugins, secrets)
	if err != nil {
		t.Fatal("deploy with deps:", err)
	}
	if pkg.Name != "full-deps" {
		t.Fatalf("wrong name: %q", pkg.Name)
	}
	if pkg.Source != "full-deps.ts" {
		t.Fatalf("wrong source: %q", pkg.Source)
	}
}

func TestTeardownPackage(t *testing.T) {
	deployer := newMockDeployer()
	deployer.deployed["pkg.ts"] = "code"

	pkg := &Package{
		Name:   "pkg",
		Source: "pkg.ts",
	}

	err := TeardownPackage(context.Background(), deployer, pkg)
	if err != nil {
		t.Fatal("teardown:", err)
	}
	if len(deployer.deployed) != 0 {
		t.Fatalf("expected 0 deployed after teardown, got %d", len(deployer.deployed))
	}
	if len(deployer.tornDown) != 1 {
		t.Fatalf("expected 1 teardown, got %d", len(deployer.tornDown))
	}
	if deployer.tornDown[0] != "pkg.ts" {
		t.Fatalf("expected teardown of 'pkg.ts', got %q", deployer.tornDown[0])
	}
}

func TestDeployFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hello.ts", `console.log("hi");`)

	deployer := newMockDeployer()
	pkg, err := DeployFile(context.Background(), deployer, filepath.Join(dir, "hello.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Name != "hello" {
		t.Fatalf("expected name 'hello', got %q", pkg.Name)
	}
	if pkg.Source != "hello.ts" {
		t.Fatalf("expected source 'hello.ts', got %q", pkg.Source)
	}
	if _, ok := deployer.deployed["hello.ts"]; !ok {
		t.Fatal("not deployed")
	}
}

// --- ResolveEntry tests ---

func TestResolveEntry_ExplicitEntry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "main.ts", `console.log("main");`)

	path, err := ResolveEntry(dir, PackageManifest{Entry: "main.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "main.ts" {
		t.Fatalf("expected main.ts, got %s", path)
	}
}

func TestResolveEntry_IndexTsFallback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `console.log("index");`)
	writeFile(t, dir, "other.ts", `console.log("other");`)

	path, err := ResolveEntry(dir, PackageManifest{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "index.ts" {
		t.Fatalf("expected index.ts, got %s", path)
	}
}

func TestResolveEntry_OnlyTsFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "solo.ts", `console.log("solo");`)

	path, err := ResolveEntry(dir, PackageManifest{})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "solo.ts" {
		t.Fatalf("expected solo.ts, got %s", path)
	}
}

func TestResolveEntry_NoTsFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := ResolveEntry(dir, PackageManifest{})
	if err == nil {
		t.Fatal("expected error for no .ts files")
	}
}

func TestResolveEntry_MultipleTsFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.ts", `console.log("a");`)
	writeFile(t, dir, "b.ts", `console.log("b");`)

	_, err := ResolveEntry(dir, PackageManifest{})
	if err == nil {
		t.Fatal("expected error for multiple .ts files without entry/index.ts")
	}
}
