package esbuild

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/packages"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
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

func TestBuilderBuildsInlineFileGraphWithJSAndJSON(t *testing.T) {
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
			"index.ts":    `import { helper } from "./helper.js"; import config from "./config.json"; output(helper + ":" + config.label);`,
			"helper.js":   `export const helper = "helper-ok";`,
			"config.json": `{"label":"json-ok"}`,
		},
	}, denyAllPlugins{}, denyAllSecrets{})
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	if !strings.Contains(built.Code, "helper-ok") || !strings.Contains(built.Code, "json-ok") {
		t.Fatalf("expected JS and JSON relative imports in built package:\n%s", built.Code)
	}
}

func TestBuilderWrapsUnsupportedBareImportDiagnostic(t *testing.T) {
	manifest, err := json.Marshal(map[string]string{
		"name":  "inline",
		"entry": "index.ts",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{
		Manifest: manifest,
		Files: map[string]string{
			"index.ts": `import { v4 as uuid } from "uuid"; output(uuid());`,
		},
	}, denyAllPlugins{}, denyAllSecrets{})
	if err == nil {
		t.Fatal("expected unsupported bare import error")
	}

	var deployErr *sdkerrors.DeployError
	if !errors.As(err, &deployErr) {
		t.Fatalf("error = %T %[1]v, want DeployError wrapper", err)
	}
	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError cause", err)
	}
	if resolverErr.Specifier != "uuid" ||
		resolverErr.Importer != "index.ts" ||
		resolverErr.Source != "inline" ||
		resolverErr.Profile != resolverProfileSourceRelative {
		t.Fatalf("resolver error = %#v", resolverErr)
	}
	if !strings.Contains(err.Error(), "source-relative") ||
		!strings.Contains(err.Error(), "allowed bare imports: kit, ai, agent, compiler") {
		t.Fatalf("diagnostic is not actionable: %v", err)
	}
}

func TestBuilderSingleFileUnsupportedBareImportDiagnosticIncludesSource(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.ts")
	if err := os.WriteFile(path, []byte(`import { v4 as uuid } from "uuid"; output(uuid());`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{Path: path}, denyAllPlugins{}, denyAllSecrets{})
	if err == nil {
		t.Fatal("expected unsupported bare import error")
	}

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError", err)
	}
	if resolverErr.Source != "single" {
		t.Fatalf("source = %q, want single", resolverErr.Source)
	}
}

func TestBuilderRejectsUnknownResolver(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `output("ok");`)
	writeFile(t, dir, "manifest.json", `{"name":"unknown-resolver","entry":"index.ts","resolver":"mystery"}`)

	_, err := (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{Path: dir}, denyAllPlugins{}, denyAllSecrets{})
	if err == nil {
		t.Fatal("expected unknown resolver error")
	}

	var validationErr *sdkerrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T %[1]v, want ValidationError", err)
	}
	if validationErr.Field != "manifest.resolver" || !strings.Contains(validationErr.Message, "unsupported resolver") {
		t.Fatalf("validation error = %#v", validationErr)
	}
}

func TestBuilderInlineRejectsNPMPreviewResolver(t *testing.T) {
	manifest, err := json.Marshal(map[string]string{
		"name":     "inline",
		"entry":    "index.ts",
		"resolver": resolverProfileNPMPreview,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{
		Manifest: manifest,
		Files: map[string]string{
			"index.ts": `output("ok");`,
		},
	}, denyAllPlugins{}, denyAllSecrets{})
	if err == nil {
		t.Fatal("expected inline npm-preview resolver error")
	}

	var validationErr *sdkerrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T %[1]v, want ValidationError", err)
	}
	if validationErr.Field != "manifest.resolver" || !strings.Contains(validationErr.Message, "filesystem package path") {
		t.Fatalf("validation error = %#v", validationErr)
	}
}

func TestBuilderPathResolverOverrideRejectsSingleFileNPMPreview(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.ts")
	if err := os.WriteFile(path, []byte(`output("ok");`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	override, err := json.Marshal(map[string]string{"resolver": resolverProfileNPMPreview})
	if err != nil {
		t.Fatal(err)
	}
	_, err = (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{
		Path:     path,
		Manifest: override,
	}, denyAllPlugins{}, denyAllSecrets{})
	if err == nil {
		t.Fatal("expected single-file npm-preview resolver error")
	}

	var validationErr *sdkerrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T %[1]v, want ValidationError", err)
	}
	if !strings.Contains(validationErr.Message, "filesystem package directory") {
		t.Fatalf("validation error = %#v", validationErr)
	}
}

func TestBuilderNPMPreviewRequiresPackageMetadata(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `output("ok");`)
	writeFile(t, dir, "manifest.json", `{"name":"npm-preview-missing","entry":"index.ts","resolver":"npm-preview"}`)

	_, err := (Builder{}).BuildPackage(context.Background(), packages.BuildRequest{Path: dir}, denyAllPlugins{}, denyAllSecrets{})
	if err == nil {
		t.Fatal("expected npm-preview package metadata error")
	}

	var validationErr *sdkerrors.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error = %T %[1]v, want ValidationError", err)
	}
	if validationErr.Field != "manifest.resolver" || !strings.Contains(validationErr.Message, "package.json") {
		t.Fatalf("validation error = %#v", validationErr)
	}
}

func TestBuilderNPMPreviewLiveInstallsAndBundlesUUID(t *testing.T) {
	if os.Getenv("BRAINKIT_TEST_NPM_PREVIEW") != "1" {
		t.Skip("set BRAINKIT_TEST_NPM_PREVIEW=1 to run live npm-preview install test")
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skipf("pnpm not available: %v", err)
	}

	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name":"brainkit-npm-preview-test","private":true,"type":"module","dependencies":{"uuid":"9.0.1"}}`)
	writeFile(t, dir, "index.ts", `import { v4 as uuid } from "uuid"; output(uuid());`)
	writeFile(t, dir, "manifest.json", `{"name":"npm-preview-live","entry":"index.ts","resolver":"npm-preview"}`)

	lockCmd := exec.Command("pnpm", "install", "--lockfile-only", "--ignore-scripts")
	lockCmd.Dir = dir
	lockCmd.Env = append(os.Environ(), "CI=1")
	if out, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("pnpm lockfile: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	built, err := (Builder{}).BuildPackage(ctx, packages.BuildRequest{Path: dir}, denyAllPlugins{}, denyAllSecrets{})
	if err != nil {
		t.Fatalf("BuildPackage: %v", err)
	}
	if built.Name != "npm-preview-live" || built.Code == "" {
		t.Fatalf("built package = %#v", built)
	}
	if strings.Contains(built.Code, `from "uuid"`) {
		t.Fatalf("uuid import was not bundled:\n%s", built.Code)
	}
}

func TestBuilderNPMPreviewMastraBrowserProviderBoundaries(t *testing.T) {
	if os.Getenv("BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW") != "1" {
		t.Skip("set BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 to run real Mastra browser-provider npm-preview canaries")
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skipf("pnpm not available: %v", err)
	}

	canaries := []mastraBrowserProviderCanary{
		{
			PackageName:       "@mastra/agent-browser",
			Version:           "0.2.2",
			ExportName:        "AgentBrowser",
			ManifestName:      "mastra-agent-browser-canary",
			ExpectedBoundary:  "unsupported-node-api",
			ExpectedSpecifier: "node:vm",
		},
		{
			PackageName:  "@mastra/stagehand",
			Version:      "0.2.2",
			ExportName:   "StagehandBrowser",
			ManifestName: "mastra-stagehand-canary",
			ExpectBuild:  true,
		},
		{
			PackageName:       "@mastra/browser-viewer",
			Version:           "0.1.1",
			ExportName:        "BrowserViewer",
			ManifestName:      "mastra-browser-viewer-canary",
			ExpectedBoundary:  "unsupported-node-api",
			ExpectedSpecifier: "http2",
		},
	}
	for _, canary := range canaries {
		t.Run(canary.ManifestName, func(t *testing.T) {
			runMastraBrowserProviderBoundaryCanary(t, canary)
		})
	}
}

type mastraBrowserProviderCanary struct {
	PackageName  string
	Version      string
	ExportName   string
	ManifestName string
	ExpectBuild  bool

	ExpectedBoundary  string
	ExpectedSpecifier string
}

func runMastraBrowserProviderBoundaryCanary(t *testing.T, canary mastraBrowserProviderCanary) {
	t.Helper()

	dir := t.TempDir()
	pkgJSON, err := json.Marshal(map[string]any{
		"name":    "brainkit-" + strings.ReplaceAll(strings.TrimPrefix(canary.ManifestName, "mastra-"), "/", "-"),
		"private": true,
		"type":    "module",
		"dependencies": map[string]string{
			canary.PackageName: canary.Version,
			"@mastra/core":     "1.33.0",
			"zod":              "3.25.76",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "package.json", string(pkgJSON))
	writeFile(t, dir, "index.ts", fmt.Sprintf(`
		import { %[1]s } from "%[2]s";
		output({ importedType: typeof %[1]s });
	`, canary.ExportName, canary.PackageName))
	writeFile(t, dir, "manifest.json", fmt.Sprintf(`{"name":%q,"entry":"index.ts","resolver":"npm-preview"}`, canary.ManifestName))

	lockCmd := exec.Command("pnpm", "install", "--lockfile-only", "--ignore-scripts")
	lockCmd.Dir = dir
	lockCmd.Env = append(os.Environ(), "CI=1")
	if out, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("pnpm lockfile: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	built, err := (Builder{}).BuildPackage(ctx, packages.BuildRequest{Path: dir}, denyAllPlugins{}, denyAllSecrets{})
	if canary.ExpectBuild {
		if err != nil {
			t.Fatalf("expected %s to bundle under npm-preview; got %T %[2]v", canary.PackageName, err)
		}
		if built.Code == "" {
			t.Fatalf("expected non-empty bundled code for %s", canary.PackageName)
		}
		if strings.Contains(built.Code, fmt.Sprintf(`from "%s"`, canary.PackageName)) {
			t.Fatalf("%s bare import was not bundled:\n%s", canary.PackageName, built.Code)
		}
		t.Logf("current %s package boundary: bundle-ok; runtime browser/provider support still unclaimed", canary.PackageName)
		return
	}
	if err == nil {
		t.Fatalf("expected %s to hit an explicit browser/Node runtime boundary before support is claimed", canary.PackageName)
	}

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError boundary diagnostic", err)
	}
	if resolverErr.Profile != resolverProfileNPMPreview {
		t.Fatalf("profile = %q, want %q", resolverErr.Profile, resolverProfileNPMPreview)
	}
	if resolverErr.BoundaryClass == "" || resolverErr.Reason == "" {
		t.Fatalf("boundary diagnostic missing class/reason: %#v", resolverErr)
	}
	if canary.ExpectedBoundary != "" && resolverErr.BoundaryClass != canary.ExpectedBoundary {
		t.Fatalf("boundary class = %q, want %q", resolverErr.BoundaryClass, canary.ExpectedBoundary)
	}
	if canary.ExpectedSpecifier != "" && resolverErr.Specifier != canary.ExpectedSpecifier {
		t.Fatalf("specifier = %q, want %q", resolverErr.Specifier, canary.ExpectedSpecifier)
	}
	t.Logf("current %s boundary: %s / %s / %s", canary.PackageName, resolverErr.BoundaryClass, resolverErr.Specifier, resolverErr.Reason)
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
