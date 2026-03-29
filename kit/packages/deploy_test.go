package packages

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// mockDeployer tracks deploy/teardown calls without a real Kernel.
type mockDeployer struct {
	deployed  map[string]string // source → code
	tornDown  []string
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
	installed map[string]bool
	running   map[string]bool
}

func (m *mockPluginChecker) IsPluginInstalled(name string) bool { return m.installed[name] }
func (m *mockPluginChecker) IsPluginRunning(name string) bool   { return m.running[name] }

type mockSecretChecker struct {
	secrets map[string]bool
}

func (m *mockSecretChecker) HasSecret(name string) bool { return m.secrets[name] }

func writeManifest(t *testing.T, dir string, manifest PackageManifestV2) {
	t.Helper()
	data, _ := json.Marshal(manifest)
	os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0644)
}

func TestDeployPackage_Basic(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "config.ts", `export const CONFIG = { name: "test" };`)
	writeFile(t, dir, "agents/main.ts", `
		import { CONFIG } from "../config";
		console.log(CONFIG.name);
	`)

	writeManifest(t, dir, PackageManifestV2{
		Name:    "test-pkg",
		Version: "1.0.0",
		Services: map[string]Service{
			"main": {Entry: "agents/main.ts"},
		},
	})

	deployer := newMockDeployer()
	pkg, err := DeployPackage(context.Background(), deployer, dir, nil, nil)
	if err != nil {
		t.Fatal("deploy:", err)
	}

	if pkg.Name != "test-pkg" {
		t.Fatalf("expected name 'test-pkg', got %q", pkg.Name)
	}
	if len(pkg.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(pkg.Services))
	}
	if pkg.Services[0] != "test-pkg/main.ts" {
		t.Fatalf("expected service 'test-pkg/main.ts', got %q", pkg.Services[0])
	}

	// Verify bundled code was passed to deployer
	code, ok := deployer.deployed["test-pkg/main.ts"]
	if !ok {
		t.Fatal("service not deployed")
	}
	if code == "" {
		t.Fatal("deployed code is empty")
	}
}

func TestDeployPackage_MultipleServices(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "svc-a.ts", `console.log("a");`)
	writeFile(t, dir, "svc-b.ts", `console.log("b");`)

	writeManifest(t, dir, PackageManifestV2{
		Name:    "multi",
		Version: "2.0.0",
		Services: map[string]Service{
			"alpha": {Entry: "svc-a.ts"},
			"beta":  {Entry: "svc-b.ts"},
		},
	})

	deployer := newMockDeployer()
	pkg, err := DeployPackage(context.Background(), deployer, dir, nil, nil)
	if err != nil {
		t.Fatal("deploy:", err)
	}

	if len(pkg.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(pkg.Services))
	}
	if len(deployer.deployed) != 2 {
		t.Fatalf("expected 2 deployed, got %d", len(deployer.deployed))
	}
}

func TestDeployPackage_MissingEntry(t *testing.T) {
	dir := t.TempDir()

	writeManifest(t, dir, PackageManifestV2{
		Name:    "broken",
		Version: "1.0.0",
		Services: map[string]Service{
			"main": {Entry: "nonexistent.ts"},
		},
	})

	deployer := newMockDeployer()
	_, err := DeployPackage(context.Background(), deployer, dir, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing entry")
	}
}

func TestDeployPackage_DependencyCheckFails(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "main.ts", `console.log("hi");`)
	writeManifest(t, dir, PackageManifestV2{
		Name:    "needs-plugin",
		Version: "1.0.0",
		Services: map[string]Service{
			"main": {Entry: "main.ts"},
		},
		Requires: &Requirements{
			Plugins: []string{"brainlet/telegram-gateway@>=1.0.0"},
		},
	})

	plugins := &mockPluginChecker{
		installed: map[string]bool{},
		running:   map[string]bool{},
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

	writeFile(t, dir, "main.ts", `console.log("hi");`)
	writeManifest(t, dir, PackageManifestV2{
		Name:    "needs-secret",
		Version: "1.0.0",
		Services: map[string]Service{
			"main": {Entry: "main.ts"},
		},
		Requires: &Requirements{
			Secrets: []string{"MY_TOKEN"},
		},
	})

	plugins := &mockPluginChecker{installed: map[string]bool{}, running: map[string]bool{}}
	secrets := &mockSecretChecker{secrets: map[string]bool{}} // MY_TOKEN not set

	deployer := newMockDeployer()
	_, err := DeployPackage(context.Background(), deployer, dir, plugins, secrets)
	if err == nil {
		t.Fatal("expected error for missing secret dependency")
	}
}

func TestDeployPackage_DepsPass(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "main.ts", `console.log("ok");`)
	writeManifest(t, dir, PackageManifestV2{
		Name:    "full-deps",
		Version: "1.0.0",
		Services: map[string]Service{
			"main": {Entry: "main.ts"},
		},
		Requires: &Requirements{
			Plugins: []string{"brainlet/telegram-gateway@>=1.0.0"},
			Secrets: []string{"BOT_TOKEN"},
		},
	})

	plugins := &mockPluginChecker{
		installed: map[string]bool{"telegram-gateway": true},
		running:   map[string]bool{"telegram-gateway": true},
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
}

func TestTeardownPackage(t *testing.T) {
	deployer := newMockDeployer()
	deployer.deployed["pkg/svc-a.ts"] = "code-a"
	deployer.deployed["pkg/svc-b.ts"] = "code-b"

	pkg := &Package{
		Name:     "pkg",
		Services: []string{"pkg/svc-a.ts", "pkg/svc-b.ts"},
	}

	err := TeardownPackage(context.Background(), deployer, pkg)
	if err != nil {
		t.Fatal("teardown:", err)
	}
	if len(deployer.deployed) != 0 {
		t.Fatalf("expected 0 deployed after teardown, got %d", len(deployer.deployed))
	}
	if len(deployer.tornDown) != 2 {
		t.Fatalf("expected 2 teardowns, got %d", len(deployer.tornDown))
	}
}
