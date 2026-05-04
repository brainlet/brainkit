package esbuild

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/modules/packages"
)

func TestBuilderBuildsDirectoryPackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.ts", `export const value = 42;`)
	writeFile(t, dir, "index.ts", `import { value } from "./config"; output(value);`)
	writeFile(t, dir, "manifest.json", `{"name":"dirpkg","version":"1.2.3","entry":"index.ts"}`)

	built, err := (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{Path: dir}, denyAllPlugins{}, denyAllSecrets{})
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	if built.Name != "dirpkg" || built.Version != "1.2.3" || built.Source != "dirpkg.ts" {
		t.Fatalf("built package = %#v", built)
	}
	if built.Code == "" || strings.Contains(built.Code, `from "./config"`) {
		t.Fatalf("code was not bundled: %q", built.Code)
	}
}

func TestBuilderBuildsSingleFilePackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.ts")
	if err := os.WriteFile(path, []byte(`output("single");`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	built, err := (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{Path: path}, denyAllPlugins{}, denyAllSecrets{})
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	if built.Name != "single" || built.Version != "0.0.0" || built.Source != "single.ts" || built.Code == "" {
		t.Fatalf("built package = %#v", built)
	}
}

func TestBuilderBuildsInlineFileGraph(t *testing.T) {
	manifest, err := json.Marshal(map[string]string{
		"name":  "inline",
		"entry": "index.ts",
	})
	if err != nil {
		t.Fatal(err)
	}

	built, err := (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{
		Manifest: manifest,
		Files: map[string]string{
			"index.ts":  `import { answer } from "./lib"; output(answer);`,
			"lib.ts":    `export const answer = 42;`,
			"unused.ts": `export const unused = true;`,
		},
	}, denyAllPlugins{}, denyAllSecrets{})
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	if built.Name != "inline" || built.Source != "inline.ts" || built.Code == "" {
		t.Fatalf("built package = %#v", built)
	}
	if strings.Contains(built.Code, `from "./lib"`) {
		t.Fatalf("code was not bundled: %q", built.Code)
	}
}

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

type denyAllPlugins struct{}

func (denyAllPlugins) IsPluginRunning(string) bool { return false }

type denyAllSecrets struct{}

func (denyAllSecrets) HasSecret(string) bool { return false }
