//go:build stress

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestWASMStress_ManyModules(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	const numModules = 10

	for i := 0; i < numModules; i++ {
		name := fmt.Sprintf("module-%d", i)
		source := fmt.Sprintf(`export function run(): i32 { return %d; }`, i)

		_, err := kit.EvalTS(ctx, fmt.Sprintf("compile%d.ts", i), fmt.Sprintf(`
			await wasm.compile(%s, { name: %q, runtime: "stub" });
		`, "`"+source+"`", name))
		if err != nil {
			t.Fatalf("compile %d: %v", i, err)
		}
	}

	modules, err := kit.ListWASMModules()
	if err != nil {
		t.Fatalf("ListWASMModules: %v", err)
	}
	if len(modules) != numModules {
		t.Fatalf("expected %d modules, got %d", numModules, len(modules))
	}

	for i := 0; i < numModules; i++ {
		name := fmt.Sprintf("module-%d", i)
		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), fmt.Sprintf(`
			var r = await wasm.run(%q);
			return JSON.stringify(r);
		`, name))
		if err != nil {
			t.Fatalf("run %s: %v", name, err)
		}
		var rr struct {
			ExitCode int `json:"exitCode"`
		}
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != i {
			t.Errorf("module-%d: exitCode=%d, want %d", i, rr.ExitCode, i)
		}
	}
}

func TestWASMStress_ModuleOverwrite(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		source := fmt.Sprintf(`export function run(): i32 { return %d; }`, i+100)
		_, err := kit.EvalTS(ctx, fmt.Sprintf("compile%d.ts", i), fmt.Sprintf(`
			await wasm.compile(%s, { name: "versioned", runtime: "stub" });
		`, "`"+source+"`"))
		if err != nil {
			t.Fatalf("compile v%d: %v", i, err)
		}
	}

	modules, _ := kit.ListWASMModules()
	count := 0
	for _, m := range modules {
		if m.Name == "versioned" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 'versioned' module, got %d", count)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		var r = await wasm.run("versioned");
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	var rr struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal([]byte(result), &rr)
	if rr.ExitCode != 104 {
		t.Errorf("exitCode=%d, want 104", rr.ExitCode)
	}
}

func TestWASMStress_CompileErrors(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		source string
	}{
		{"syntax_error", "export function run(: i32 { return 0; }"},
		{"type_error", "export function run(): i32 { return \"not a number\"; }"},
		{"missing_import", "import { missing } from 'nonexistent';"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := kit.EvalTS(ctx, tc.name+".ts", fmt.Sprintf(`
				try {
					await wasm.compile(%s, { name: %q, runtime: "stub" });
					return JSON.stringify({ ok: true });
				} catch(e) {
					return JSON.stringify({ ok: false, error: e.message || String(e) });
				}
			`, "`"+tc.source+"`", tc.name))
			if err != nil {
				t.Fatalf("EvalTS: %v", err)
			}
			var r struct {
				OK    bool   `json:"ok"`
				Error string `json:"error"`
			}
			json.Unmarshal([]byte(result), &r)
			if r.OK {
				t.Errorf("expected compile to fail for %q, but it succeeded", tc.name)
			} else {
				t.Logf("%s error: %s", tc.name, r.Error)
			}
		})
	}
}

func TestWASMStress_RunNonExistent(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	result, err := kit.EvalTS(ctx, "run.ts", `
		try {
			var r = await wasm.run("does-not-exist");
			return JSON.stringify({ ok: true });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	var r struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	json.Unmarshal([]byte(result), &r)
	if r.OK {
		t.Error("expected run to fail for non-existent module")
	}
	if !strings.Contains(r.Error, "not found") {
		t.Errorf("expected 'not found' error, got: %s", r.Error)
	}
}

func TestWASMStress_RemoveAndRecompile(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		source := fmt.Sprintf(`export function run(): i32 { return %d; }`, i)
		_, err := kit.EvalTS(ctx, fmt.Sprintf("compile%d.ts", i), fmt.Sprintf(`
			await wasm.compile(%s, { name: "recycled", runtime: "stub" });
		`, "`"+source+"`"))
		if err != nil {
			t.Fatalf("compile %d: %v", i, err)
		}

		result, err := kit.EvalTS(ctx, fmt.Sprintf("run%d.ts", i), `
			var r = await wasm.run("recycled");
			return JSON.stringify(r);
		`)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		var rr struct {
			ExitCode int `json:"exitCode"`
		}
		json.Unmarshal([]byte(result), &rr)
		if rr.ExitCode != i {
			t.Errorf("cycle %d: exitCode=%d, want %d", i, rr.ExitCode, i)
		}

		err = kit.RemoveWASMModule("recycled")
		if err != nil {
			t.Fatalf("remove %d: %v", i, err)
		}

		info, _ := kit.GetWASMModule("recycled")
		if info != nil {
			t.Fatalf("module should be removed after cycle %d", i)
		}
	}
}

func TestWASMStress_NoRunExport(t *testing.T) {
	kit := newTestKitNoKey(t)
	ctx := context.Background()

	source := `
export function compute(): i32 {
  return 42;
}
`
	_, err := kit.EvalTS(ctx, "compile.ts", `
		await wasm.compile(`+"`"+source+"`"+`, { name: "no-run", runtime: "stub" });
	`)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	result, err := kit.EvalTS(ctx, "run.ts", `
		try {
			var r = await wasm.run("no-run");
			return JSON.stringify({ ok: true, exitCode: r.exitCode });
		} catch(e) {
			return JSON.stringify({ ok: false, error: e.message || String(e) });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}
	t.Logf("no-run result: %s", result)
}
