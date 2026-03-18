package brainkit

import (
	"context"
	"testing"
)

func TestWASMLifecycle_NamedCompile(t *testing.T) {
	kit := newTestKitNoKey(t)

	result, err := kit.EvalTS(context.Background(), "compile.ts", `
		var mod = await wasm.compile(
			'export function run(): i32 { return 42; }',
			{ name: "calculator" }
		);
		return JSON.stringify({ name: mod.name, size: mod.size, exports: mod.exports });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	t.Logf("Compile result: %s", result)

	info, err := kit.GetWASMModule("calculator")
	if err != nil {
		t.Fatalf("GetWASMModule: %v", err)
	}
	if info == nil {
		t.Fatal("module 'calculator' not found")
	}
	if info.Name != "calculator" {
		t.Errorf("name = %q, want calculator", info.Name)
	}
	if info.Size == 0 {
		t.Error("size should be > 0")
	}
	if info.SourceHash == "" {
		t.Error("source hash should not be empty")
	}
	t.Logf("Module: name=%s size=%d exports=%v hash=%s", info.Name, info.Size, info.Exports, info.SourceHash[:8])
}

func TestWASMLifecycle_AnonymousCompile(t *testing.T) {
	kit := newTestKitNoKey(t)

	_, err := kit.EvalTS(context.Background(), "compile.ts", `
		var mod = await wasm.compile('export function run(): i32 { return 1; }');
		return JSON.stringify(mod.name);
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	modules, _ := kit.ListWASMModules()
	if len(modules) == 0 {
		t.Fatal("expected at least 1 module")
	}
	t.Logf("anonymous module name: %s", modules[0].Name)
}

func TestWASMLifecycle_List(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Compile each module separately so we can see which one fails
	result1, err := kit.EvalTS(context.Background(), "compile1.ts", `
		try {
			var m = await wasm.compile('export function run(): i32 { return 1; }', { name: "alpha" });
			return JSON.stringify({ ok: true, name: m.name, size: m.size });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("compile alpha EvalTS: %v", err)
	}
	t.Logf("alpha: %s", result1)

	result2, err := kit.EvalTS(context.Background(), "compile2.ts", `
		try {
			var m = await wasm.compile('export function run(): i32 { return 2; }', { name: "beta" });
			return JSON.stringify({ ok: true, name: m.name, size: m.size, exports: m.exports });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e), stack: e.stack || "" });
		}
	`)
	if err != nil {
		t.Fatalf("compile beta EvalTS: %v", err)
	}
	t.Logf("beta: %s", result2)

	modules, err := kit.ListWASMModules()
	if err != nil {
		t.Fatalf("ListWASMModules: %v", err)
	}

	names := map[string]bool{}
	for _, m := range modules {
		names[m.Name] = true
		t.Logf("found module: %s (size=%d)", m.Name, m.Size)
	}
	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d (names: %v)", len(modules), names)
	}
	if !names["alpha"] {
		t.Error("missing module 'alpha'")
	}
	if !names["beta"] {
		t.Error("missing module 'beta'")
	}
}

func TestWASMLifecycle_RunByName(t *testing.T) {
	kit := newTestKitNoKey(t)

	_, err := kit.EvalTS(context.Background(), "compile.ts", `
		await wasm.compile('export function run(): i32 { return 42; }', { name: "answer" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(context.Background(), "run.ts", `
		var result = await wasm.run("answer");
		return JSON.stringify(result);
	`)
	if err != nil {
		t.Fatalf("run by name: %v", err)
	}
	t.Logf("Run result: %s", result)
}

func TestWASMLifecycle_Remove(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.EvalTS(context.Background(), "compile.ts", `
		await wasm.compile('export function run(): i32 { return 1; }', { name: "temp" });
	`)

	info, _ := kit.GetWASMModule("temp")
	if info == nil {
		t.Fatal("module should exist")
	}

	err := kit.RemoveWASMModule("temp")
	if err != nil {
		t.Fatalf("RemoveWASMModule: %v", err)
	}

	info2, _ := kit.GetWASMModule("temp")
	if info2 != nil {
		t.Error("module should be removed")
	}
}

func TestWASMLifecycle_Idempotent(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.EvalTS(context.Background(), "v1.ts", `
		await wasm.compile('export function run(): i32 { return 1; }', { name: "versioned" });
	`)
	kit.EvalTS(context.Background(), "v2.ts", `
		await wasm.compile('export function run(): i32 { return 2; }', { name: "versioned" });
	`)

	modules, _ := kit.ListWASMModules()
	count := 0
	for _, m := range modules {
		if m.Name == "versioned" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 'versioned' module (idempotent), got %d", count)
	}
}

func TestWASMLifecycle_JSListAndExists(t *testing.T) {
	kit := newTestKitNoKey(t)

	result, err := kit.EvalTS(context.Background(), "test.ts", `
		await wasm.compile('export function run(): i32 { return 1; }', { name: "checker" });

		var exists = wasm.exists("checker");
		var notExists = wasm.exists("nonexistent");
		var list = wasm.list();
		var info = wasm.get("checker");

		return JSON.stringify({
			exists: exists,
			notExists: notExists,
			listLen: list.length,
			infoName: info ? info.name : null,
		});
	`)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	t.Logf("JS result: %s", result)
}

func TestWASMLifecycle_ResourceRegistryIntegration(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.EvalTS(context.Background(), "my-modules.ts", `
		await wasm.compile('export function run(): i32 { return 1; }', { name: "tracked" });
	`)

	resources, err := kit.ListResources("wasm")
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	found := false
	for _, r := range resources {
		if r.Name == "tracked" {
			found = true
			if r.Source != "my-modules.ts" {
				t.Errorf("source = %q, want my-modules.ts", r.Source)
			}
		}
	}
	if !found {
		t.Errorf("wasm module 'tracked' not in resource registry: %+v", resources)
	}
}
