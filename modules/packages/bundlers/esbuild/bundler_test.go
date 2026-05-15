package esbuild

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

func TestBundleSingleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		const x: number = 42;
		console.log(x);
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if strings.Contains(result, ": number") {
		t.Fatal("TypeScript annotations not stripped")
	}
	if !strings.Contains(result, "42") {
		t.Fatal("expected 42 in output")
	}
}

func TestBundleRelativeImports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.ts", `export const CONFIG = { model: "openai/gpt-4o-mini" };`)
	writeFile(t, dir, "utils/formatter.ts", `
		import { CONFIG } from "../config";
		export function format(text: string): string {
			return "[" + CONFIG.model + "] " + text;
		}
	`)
	writeFile(t, dir, "index.ts", `
		import { CONFIG } from "./config";
		import { format } from "./utils/formatter";
		console.log(format("hello"));
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if !strings.Contains(result, "openai/gpt-4o-mini") {
		t.Fatal("expected CONFIG value in output")
	}
	if !strings.Contains(result, "format") {
		t.Fatal("expected format function in output")
	}
}

func TestBundleRelativeJSAndJSONImports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "helper.js", `export const helper = "helper-ok";`)
	writeFile(t, dir, "config.json", `{"label":"json-ok"}`)
	writeFile(t, dir, "index.ts", `
		import { helper } from "./helper.js";
		import config from "./config.json";
		console.log(helper, config.label);
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if !strings.Contains(result, "helper-ok") || !strings.Contains(result, "json-ok") {
		t.Fatalf("expected JS and JSON relative imports in output:\n%s", result)
	}
	if strings.Contains(result, `from "./helper.js"`) || strings.Contains(result, `from "./config.json"`) {
		t.Fatalf("relative imports were not bundled:\n%s", result)
	}
}

func TestBundleScopeIsolation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.ts", `const helper = "a"; export const A = helper;`)
	writeFile(t, dir, "b.ts", `const helper = "b"; export const B = helper;`)
	writeFile(t, dir, "index.ts", `
		import { A } from "./a";
		import { B } from "./b";
		console.log(A, B);
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if !strings.Contains(result, `"a"`) || !strings.Contains(result, `"b"`) {
		t.Fatalf("expected both values in output:\n%s", result)
	}
}

func TestBundleExternalModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		import { bus, model } from "kit";
		import { generateText as runText } from "ai";
			import { Agent as BrainkitAgent, AgentChannels } from "agent";
			console.log("hello", typeof bus, typeof model, typeof runText, typeof BrainkitAgent, typeof AgentChannels);
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if strings.Contains(result, `from "kit"`) || strings.Contains(result, `from "ai"`) || strings.Contains(result, `from "agent"`) {
		t.Fatalf("external imports should be stripped from runtime artifact:\n%s", result)
	}
	if strings.Contains(result, `require("kit")`) {
		t.Fatal("external 'kit' should not be require'd")
	}
	for _, expected := range []string{"globalThis.bus", "globalThis.generateText", "globalThis.Agent", "globalThis.AgentChannels"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected runtime module stub binding %s in output:\n%s", expected, result)
		}
	}
}

func TestBundleRejectsUnsupportedBareImportWithResolverDiagnostic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		import { v4 as uuid } from "uuid";
		console.log(uuid());
	`)

	_, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{sourcePackage: "bare-fixture"})
	if err == nil {
		t.Fatal("expected unsupported bare import error")
	}

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError", err)
	}
	if resolverErr.Specifier != "uuid" {
		t.Fatalf("specifier = %q, want uuid", resolverErr.Specifier)
	}
	if !strings.Contains(resolverErr.Importer, "index.ts") {
		t.Fatalf("importer = %q, want index.ts", resolverErr.Importer)
	}
	if resolverErr.Source != "bare-fixture" {
		t.Fatalf("source = %q, want bare-fixture", resolverErr.Source)
	}
	if resolverErr.Profile != resolverProfileSourceRelative {
		t.Fatalf("profile = %q, want %q", resolverErr.Profile, resolverProfileSourceRelative)
	}
	if strings.Join(resolverErr.AllowedBareImports, ",") != "kit,ai,agent,compiler" {
		t.Fatalf("allowed bare imports = %#v", resolverErr.AllowedBareImports)
	}
	if !strings.Contains(resolverErr.SuggestedOwner, "npm ecosystem resolver profile") {
		t.Fatalf("suggested owner = %q", resolverErr.SuggestedOwner)
	}
	if !strings.Contains(err.Error(), `unsupported bare import "uuid"`) ||
		!strings.Contains(err.Error(), "source-relative") ||
		!strings.Contains(err.Error(), "allowed bare imports: kit, ai, agent, compiler") {
		t.Fatalf("diagnostic is not actionable: %v", err)
	}
}

func TestBundleNPMPreviewAllowsInstalledBarePackage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "node_modules/fixture-pkg/package.json", `{"name":"fixture-pkg","version":"1.0.0","module":"index.js"}`)
	writeFile(t, dir, "node_modules/fixture-pkg/index.js", `export function value() { return "fixture-ok"; }`)
	writeFile(t, dir, "index.ts", `
		import { value } from "fixture-pkg";
		output(value());
	`)

	result, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "npm-fixture",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if !strings.Contains(result, "fixture-ok") {
		t.Fatalf("expected installed package code in output:\n%s", result)
	}
	if strings.Contains(result, `from "fixture-pkg"`) {
		t.Fatalf("bare package import was not bundled:\n%s", result)
	}
}

func TestBundleNPMPreviewAliasesWSBrowserShim(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "node_modules/ws/package.json", `{
		"name": "ws",
		"version": "1.0.0",
		"exports": {
			".": {
				"browser": "./browser.js",
				"import": "./wrapper.mjs",
				"require": "./index.js"
			}
		},
		"browser": "./browser.js",
		"main": "./index.js"
	}`)
	writeFile(t, dir, "node_modules/ws/browser.js", `module.exports = function(){ throw new Error("ws browser shim selected"); };`)
	writeFile(t, dir, "node_modules/ws/wrapper.mjs", `export default function NodeWS(){}`)
	writeFile(t, dir, "node_modules/ws/index.js", `module.exports = function NodeWS(){}`)
	writeFile(t, dir, "index.ts", `
		import WebSocket, { WebSocket as NamedWebSocket } from "ws";
		output({
			defaultType: typeof WebSocket,
			namedType: typeof NamedWebSocket,
			same: WebSocket === NamedWebSocket,
		});
	`)

	result, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "npm-ws-fixture",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err != nil {
		t.Fatal("bundle:", err)
	}
	if !strings.Contains(result, "globalThis.WebSocket") {
		t.Fatalf("expected npm-preview ws adapter to use Brainkit WebSocket global:\n%s", result)
	}
	if strings.Contains(result, "ws browser shim selected") {
		t.Fatalf("npm-preview selected ws browser shim instead of Brainkit adapter:\n%s", result)
	}
}

func TestBundleNPMPreviewAliasesZodToV4(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "node_modules/zod/package.json", `{
		"name": "zod",
		"version": "3.25.76",
		"exports": {
			".": "./index.js",
			"./v4": "./v4/index.js"
		},
		"module": "./index.js"
	}`)
	writeFile(t, dir, "node_modules/zod/index.js", `export const marker = "zod-v3-entry";`)
	writeFile(t, dir, "node_modules/zod/v4/index.js", `export const marker = "zod-v4-entry";`)
	writeFile(t, dir, "index.ts", `
		import { marker } from "zod";
		output({ marker });
	`)

	result, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "npm-zod-fixture",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err != nil {
		t.Fatal("bundle:", err)
	}
	if !strings.Contains(result, "zod-v4-entry") {
		t.Fatalf("expected exact zod import to resolve to zod/v4:\n%s", result)
	}
	if strings.Contains(result, "zod-v3-entry") {
		t.Fatalf("npm-preview selected zod v3 entry instead of zod/v4:\n%s", result)
	}
}

func TestBundleNPMPreviewBindsBrainkitRuntimeModuleAliases(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "node_modules/uses-ai/package.json", `{"name":"uses-ai","version":"1.0.0","module":"index.js"}`)
	writeFile(t, dir, "node_modules/uses-ai/index.js", `
		import { generateObject as generateObjectFromRuntime } from "ai";
		export function value() { return typeof generateObjectFromRuntime; }
	`)
	writeFile(t, dir, "index.ts", `
		import { value } from "uses-ai";
		output({ value: value() });
	`)

	result, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "npm-runtime-stub-fixture",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err != nil {
		t.Fatal("bundle:", err)
	}
	if !strings.Contains(result, "globalThis.generateObject") {
		t.Fatalf("expected aliased dependency import from ai to bind through Brainkit runtime stub:\n%s", result)
	}
	if strings.Contains(result, `from "ai"`) {
		t.Fatalf("ai bare import was not bundled into runtime stub:\n%s", result)
	}
}

func TestBundleNPMPreviewAllowsJSBridgeOwnedNodeBuiltinStubs(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		import { EventEmitter } from "node:events";
		import { readFileSync } from "fs";
		import { join } from "node:path";
		import { fileURLToPath } from "node:url";
		import { setTimeout as sleep } from "node:timers/promises";
		import { createRequire } from "node:module";
		import process from "node:process";
		output({
			emitter: typeof EventEmitter,
			fs: typeof readFileSync,
			path: join("a", "b"),
			url: typeof fileURLToPath,
			timer: typeof sleep,
			module: typeof createRequire,
			getBuiltin: typeof process.getBuiltinModule,
		});
	`)

	result, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "node-stubs",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err != nil {
		t.Fatal("bundle:", err)
	}

	for _, unexpected := range []string{`from "node:events"`, `from "fs"`, `from "node:path"`, `from "node:url"`, `from "node:timers/promises"`, `from "node:module"`, `from "node:process"`} {
		if strings.Contains(result, unexpected) {
			t.Fatalf("Node builtin import %s was not bundled:\n%s", unexpected, result)
		}
	}
	for _, expected := range []string{"globalThis.EventEmitter", "globalThis.fs", "globalThis.path", "globalThis.node_url", "globalThis.timersPromises", "globalThis.node_module", "globalThis.process"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("expected %s stub reference in output:\n%s", expected, result)
		}
	}
}

func TestBundleNPMPreviewRewritesDynamicImportForSES(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		const load = (specifier: string) => import(specifier);
		const hint = "set globalThis.File to import('node:buffer').File";
		output({ loadType: typeof load, hint });
	`)

	result, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "dynamic-import",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err != nil {
		t.Fatal("bundle:", err)
	}
	if strings.Contains(result, "import(") {
		t.Fatalf("dynamic import expression should be rewritten before SES eval:\n%s", result)
	}
	if !strings.Contains(result, "__require") {
		t.Fatalf("expected esbuild dynamic import lowering through require helper:\n%s", result)
	}
	if !strings.Contains(result, `import\u0028`) {
		t.Fatalf("expected import-token text inside strings to be escaped for SES:\n%s", result)
	}
}

func TestBundleNPMPreviewRejectsNodeBuiltinWithResolverDiagnostic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		import vm from "node:vm";
		output(vm);
	`)

	_, err := bundle(filepath.Join(dir, "index.ts"), bundleOptions{
		sourcePackage:   "node-boundary",
		resolverProfile: resolverProfileNPMPreview,
		packageRoot:     dir,
	})
	if err == nil {
		t.Fatal("expected unsupported Node builtin error")
	}

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError", err)
	}
	if resolverErr.Profile != resolverProfileNPMPreview {
		t.Fatalf("profile = %q, want %q", resolverErr.Profile, resolverProfileNPMPreview)
	}
	if resolverErr.BoundaryClass != "unsupported-node-api" {
		t.Fatalf("boundary class = %q, want unsupported-node-api", resolverErr.BoundaryClass)
	}
	if resolverErr.Reason == "" || !strings.Contains(err.Error(), "Node builtin") {
		t.Fatalf("diagnostic is not actionable: %#v / %v", resolverErr, err)
	}
}

func TestBundleTypeScriptStripping(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		interface Config {
			model: string;
			temperature: number;
		}
		type Result = { text: string };
		const cfg: Config = { model: "gpt-4", temperature: 0.7 };
		function run(c: Config): Result {
			return { text: c.model };
		}
		console.log(run(cfg));
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	if strings.Contains(result, "interface") {
		t.Fatal("interface not stripped")
	}
	if !strings.Contains(result, "gpt-4") {
		t.Fatal("expected runtime value in output")
	}
}

func TestBundleMissingFile(t *testing.T) {
	_, err := Bundle("/nonexistent/path/index.ts")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBundleImportError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		import { X } from "./does-not-exist";
		console.log(X);
	`)

	_, err := Bundle(filepath.Join(dir, "index.ts"))
	if err == nil {
		t.Fatal("expected error for missing import")
	}
}

func TestBundleInMemoryRelativeJSAndJSONImports(t *testing.T) {
	result, err := BundleInMemory(map[string]string{
		"index.ts":    `import { helper } from "./helper.js"; import config from "./config.json"; console.log(helper, config.label);`,
		"helper.js":   `export const helper = "helper-ok";`,
		"config.json": `{"label":"json-ok"}`,
	}, "index.ts")
	if err != nil {
		t.Fatal("bundle:", err)
	}
	if !strings.Contains(result, "helper-ok") || !strings.Contains(result, "json-ok") {
		t.Fatalf("expected JS and JSON relative imports in output:\n%s", result)
	}
}

func TestBundleInMemoryRejectsUnsupportedBareImportWithResolverDiagnostic(t *testing.T) {
	_, err := bundleInMemory(map[string]string{
		"index.ts": `import { v4 as uuid } from "uuid"; console.log(uuid());`,
	}, "index.ts", bundleOptions{sourcePackage: "inline-fixture"})
	if err == nil {
		t.Fatal("expected unsupported bare import error")
	}

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError", err)
	}
	if resolverErr.Specifier != "uuid" ||
		resolverErr.Importer != "index.ts" ||
		resolverErr.Source != "inline-fixture" ||
		resolverErr.Profile != resolverProfileSourceRelative {
		t.Fatalf("resolver error = %#v", resolverErr)
	}
}

func TestBundleInMemoryNPMPreviewRejectsBareImportWithResolverDiagnostic(t *testing.T) {
	_, err := bundleInMemory(map[string]string{
		"index.ts": `import { v4 as uuid } from "uuid"; console.log(uuid());`,
	}, "index.ts", bundleOptions{sourcePackage: "inline-fixture", resolverProfile: resolverProfileNPMPreview})
	if err == nil {
		t.Fatal("expected unsupported npm-preview in-memory error")
	}

	var resolverErr *sdkerrors.PackageResolverError
	if !errors.As(err, &resolverErr) {
		t.Fatalf("error = %T %[1]v, want PackageResolverError", err)
	}
	if resolverErr.Profile != resolverProfileNPMPreview ||
		resolverErr.BoundaryClass != "resolver-profile" ||
		!strings.Contains(resolverErr.Reason, "filesystem package root") {
		t.Fatalf("resolver error = %#v", resolverErr)
	}
}
