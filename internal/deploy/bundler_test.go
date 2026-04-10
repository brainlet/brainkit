package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBundle_SingleFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		const x: number = 42;
		console.log(x);
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	// TypeScript stripped (no ": number")
	if strings.Contains(result, ": number") {
		t.Fatal("TypeScript annotations not stripped")
	}
	// Code preserved
	if !strings.Contains(result, "42") {
		t.Fatal("expected 42 in output")
	}
}

func TestBundle_RelativeImports(t *testing.T) {
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

	// Both modules inlined
	if !strings.Contains(result, "openai/gpt-4o-mini") {
		t.Fatal("expected CONFIG value in output")
	}
	if !strings.Contains(result, "format") {
		t.Fatal("expected format function in output")
	}
}

func TestBundle_ScopeIsolation(t *testing.T) {
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

	// esbuild renames one of the `helper` variables to avoid collision
	// Both values should be present
	if !strings.Contains(result, `"a"`) || !strings.Contains(result, `"b"`) {
		t.Fatalf("expected both values in output:\n%s", result)
	}
}

func TestBundle_ExternalModules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.ts", `
		import { bus, model } from "kit";
		import { generateText } from "ai";
		import { Agent } from "agent";
		console.log("hello");
	`)

	result, err := Bundle(filepath.Join(dir, "index.ts"))
	if err != nil {
		t.Fatal("bundle:", err)
	}

	// External imports should NOT be bundled — they're provided by endowments
	// esbuild with IIFE format drops external imports (they become global refs)
	if strings.Contains(result, `require("kit")`) {
		t.Fatal("external 'kit' should not be require'd")
	}
}

func TestBundle_TypeScriptStripping(t *testing.T) {
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

	// Interfaces and type aliases stripped
	if strings.Contains(result, "interface") {
		t.Fatal("interface not stripped")
	}
	// Runtime code preserved
	if !strings.Contains(result, "gpt-4") {
		t.Fatal("expected runtime value in output")
	}
}

func TestBundle_MissingFile(t *testing.T) {
	_, err := Bundle("/nonexistent/path/index.ts")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBundle_ImportError(t *testing.T) {
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
