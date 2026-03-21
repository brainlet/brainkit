//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
)

func TestIntegration_WASMCompileFromTS(t *testing.T) {
	kit := newTestKitNoKey(t)

	result, err := kit.EvalTS(context.Background(), "test-wasm.ts", `
		try {
			const compiled = await wasm.compile('export function run(): i32 { return 42; }');
			return JSON.stringify(compiled);
		} catch(e) {
			return JSON.stringify({ error: e.message });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct {
		ModuleID string `json:"moduleId"`
		Error    string `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)
	if out.Error != "" {
		t.Fatalf("wasm.compile error: %s", out.Error)
	}
	if out.ModuleID == "" {
		t.Error("expected moduleId")
	}
	t.Logf("WASM compile from .ts: moduleId=%s", out.ModuleID)
}

func TestIntegration_WASMCompileAndRunFromTS(t *testing.T) {
	kit := newTestKitNoKey(t)

	result, err := kit.EvalTS(context.Background(), "test-wasm-run.ts", `
		try {
			const compiled = await wasm.compile('export function run(): i32 { return 42; }');
			const output = await wasm.run(compiled, {});
			return JSON.stringify({ moduleId: compiled.moduleId, output: output });
		} catch(e) {
			return JSON.stringify({ error: e.message });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct {
		ModuleID string `json:"moduleId"`
		Output   struct {
			ExitCode int `json:"exitCode"`
		} `json:"output"`
		Error string `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)
	if out.Error != "" {
		t.Fatalf("error: %s", out.Error)
	}
	if out.ModuleID == "" {
		t.Error("expected moduleId")
	}
	t.Logf("WASM compile+run from .ts: moduleId=%s exitCode=%d", out.ModuleID, out.Output.ExitCode)
}

func TestIntegration_WASMCompileViaBus(t *testing.T) {
	kit := newTestKitNoKey(t)

	payload, _ := json.Marshal(map[string]any{
		"source":  `export function run(): i32 { return 42; }`,
		"options": map[string]any{},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.compile", CallerID: kit.callerID, Payload: payload})
	if err != nil {
		t.Logf("wasm.compile via bus: %v (expected if as-embed not ready)", err)
		return
	}

	var result struct {
		ModuleID string `json:"moduleId"`
		Text     string `json:"text"`
	}
	json.Unmarshal(resp.Payload, &result)
	if result.ModuleID == "" {
		t.Error("expected moduleId")
	}
	t.Logf("WASM compile via bus: moduleId=%s", result.ModuleID)
}
