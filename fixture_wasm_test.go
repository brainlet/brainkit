//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestFixture_AS_Return42(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/return-42.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.compile", CallerID: kit.callerID, Payload: compilePayload})
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct {
		ModuleID string `json:"moduleId"`
	}
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.run", CallerID: kit.callerID, Payload: runPayload})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 0 {
		t.Errorf("exitCode = %d, want 0", result.ExitCode)
	}
	t.Logf("fixture as/return-42: exitCode=%d", result.ExitCode)
}

func TestFixture_AS_Fibonacci(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/fibonacci.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.compile", CallerID: kit.callerID, Payload: compilePayload})
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct {
		ModuleID string `json:"moduleId"`
	}
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.run", CallerID: kit.callerID, Payload: runPayload})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 0 {
		t.Errorf("exitCode = %d, want 0 (subtest %d failed)", result.ExitCode, result.ExitCode)
	}
	t.Logf("fixture as/fibonacci: exitCode=%d", result.ExitCode)
}

func TestFixture_AS_Arithmetic(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/arithmetic.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.compile", CallerID: kit.callerID, Payload: compilePayload})
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct {
		ModuleID string `json:"moduleId"`
	}
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := bus.AskSync(kit.Bus, ctx, bus.Message{Topic: "wasm.run", CallerID: kit.callerID, Payload: runPayload})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		ExitCode int `json:"exitCode"`
	}
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 0 {
		t.Errorf("exitCode = %d, want 0 (subtest %d failed)", result.ExitCode, result.ExitCode)
	}
	t.Logf("fixture as/arithmetic: exitCode=%d", result.ExitCode)
}

func TestFixture_TS_FullComposition(t *testing.T) {
	kit := newTestKit(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/reverse", ShortName: "reverse",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Reverses a string",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				runes := []rune(args.Text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result, _ := json.Marshal(map[string]string{"result": string(runes)})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/full-composition.js")
	result, err := kit.EvalModule(context.Background(), "full-composition.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Sandbox struct {
			Ns string `json:"ns"`
			Id string `json:"id"`
		} `json:"sandbox"`
		AIText       string `json:"aiText"`
		Reversed     string `json:"reversed"`
		HasLocalTool bool   `json:"hasLocalTool"`
		WasmExitCode int    `json:"wasmExitCode"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Sandbox.Ns != "test" {
		t.Errorf("namespace = %q", out.Sandbox.Ns)
	}
	if out.AIText == "" {
		t.Error("ai.generate returned empty text")
	}
	if out.Reversed != "telniarb" {
		t.Errorf("reversed = %q, want telniarb", out.Reversed)
	}
	if !out.HasLocalTool {
		t.Error("createTool failed")
	}
	if out.WasmExitCode != 99 {
		t.Errorf("wasm exitCode = %d, want 99", out.WasmExitCode)
	}
	t.Logf("fixture full-composition: ai=%q reversed=%q tool=%v wasm=%d",
		out.AIText, out.Reversed, out.HasLocalTool, out.WasmExitCode)
}
