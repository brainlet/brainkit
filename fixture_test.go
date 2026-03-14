package brainkit

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/registry"
)

// loadFixture reads a test fixture file from testdata/.
func loadFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", path, err)
	}
	return string(data)
}

// ═══════════════════════════════════════════════════════════════
// .ts FIXTURES — real modules developers would write
// ═══════════════════════════════════════════════════════════════

func TestFixture_TS_AgentGenerate(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-generate.js")

	result, err := kit.EvalModule(context.Background(), "agent-generate.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text         string `json:"text"`
		HasUsage     bool   `json:"hasUsage"`
		FinishReason string `json:"finishReason"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(strings.ToUpper(out.Text), "FIXTURE_WORKS") {
		t.Errorf("text = %q", out.Text)
	}
	if !out.HasUsage {
		t.Error("expected usage")
	}
	t.Logf("fixture agent-generate: %q finish=%s", out.Text, out.FinishReason)
}

func TestFixture_TS_AgentStream(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-stream.js")

	result, err := kit.EvalModule(context.Background(), "agent-stream.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text              string `json:"text"`
		Chunks            int    `json:"chunks"`
		HasRealTimeTokens bool   `json:"hasRealTimeTokens"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Text == "" {
		t.Error("expected non-empty text")
	}
	if !out.HasRealTimeTokens {
		t.Error("expected real-time token chunks")
	}
	t.Logf("fixture agent-stream: %d chunks, text=%q", out.Chunks, out.Text)
}

func TestFixture_TS_AgentWithLocalTool(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-with-local-tool.js")

	result, err := kit.EvalModule(context.Background(), "agent-with-local-tool.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		ToolCalls int    `json:"toolCalls"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(out.Text, "42") {
		t.Errorf("expected 42: %q", out.Text)
	}
	t.Logf("fixture agent-with-local-tool: %q toolCalls=%d", out.Text, out.ToolCalls)
}

func TestFixture_TS_AgentWithRegisteredTool(t *testing.T) {
	kit := newTestKit(t)

	// Register the "multiply" tool that the fixture expects
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.multiply", ShortName: "multiply", Namespace: "platform",
		Description: "Multiplies two numbers",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"number","description":"first number"},"b":{"type":"number","description":"second number"}},"required":["a","b"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ A, B float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.A * args.B})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/agent-with-registered-tool.js")
	result, err := kit.EvalModule(context.Background(), "agent-with-registered-tool.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		ToolCalls int    `json:"toolCalls"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(out.Text, "42") {
		t.Errorf("expected 42: %q", out.Text)
	}
	t.Logf("fixture agent-with-registered-tool: %q toolCalls=%d", out.Text, out.ToolCalls)
}

func TestFixture_TS_AIGenerate(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/ai-generate.js")

	result, err := kit.EvalModule(context.Background(), "ai-generate.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text     string `json:"text"`
		HasUsage bool   `json:"hasUsage"`
	}
	json.Unmarshal([]byte(result), &out)

	if !strings.Contains(strings.ToUpper(out.Text), "DIRECT") {
		t.Errorf("text = %q", out.Text)
	}
	t.Logf("fixture ai-generate: %q", out.Text)
}

func TestFixture_TS_ToolsCall(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Register the "uppercase" tool that the fixture expects
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.uppercase", ShortName: "uppercase", Namespace: "platform",
		Description: "Converts text to uppercase",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]string{"result": strings.ToUpper(args.Text)})
				return result, nil
			},
		},
	})

	code := loadFixture(t, "testdata/ts/tools-call.js")
	result, err := kit.EvalModule(context.Background(), "tools-call.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct{ Result string }
	json.Unmarshal([]byte(result), &out)

	if out.Result != "HELLO BRAINLET" {
		t.Errorf("result = %q", out.Result)
	}
	t.Logf("fixture tools-call: %q", out.Result)
}

func TestFixture_TS_SandboxContext(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/sandbox-context.js")

	result, err := kit.EvalModule(context.Background(), "sandbox-context.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		ID        string `json:"id"`
		Namespace string `json:"namespace"`
		CallerID  string `json:"callerID"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.ID == "" {
		t.Error("empty id")
	}
	if out.Namespace != "test" {
		t.Errorf("namespace = %q", out.Namespace)
	}
	t.Logf("fixture sandbox-context: %+v", out)
}

// ═══════════════════════════════════════════════════════════════
// AS/WASM FIXTURES — compile and run AssemblyScript
// ═══════════════════════════════════════════════════════════════

func TestFixture_AS_Return42(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/return-42.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile
	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := kit.Bus.Request(ctx, "wasm.compile", kit.callerID, compilePayload)
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct{ ModuleID string `json:"moduleId"` }
	json.Unmarshal(compileResp.Payload, &compiled)

	// Run
	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := kit.Bus.Request(ctx, "wasm.run", kit.callerID, runPayload)
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 42 {
		t.Errorf("exitCode = %d, want 42", result.ExitCode)
	}
	t.Logf("fixture as/return-42: exitCode=%d", result.ExitCode)
}

func TestFixture_AS_Fibonacci(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/fibonacci.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := kit.Bus.Request(ctx, "wasm.compile", kit.callerID, compilePayload)
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct{ ModuleID string `json:"moduleId"` }
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := kit.Bus.Request(ctx, "wasm.run", kit.callerID, runPayload)
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 55 {
		t.Errorf("exitCode = %d, want 55 (fib(10))", result.ExitCode)
	}
	t.Logf("fixture as/fibonacci: fib(10)=%d", result.ExitCode)
}

func TestFixture_AS_Arithmetic(t *testing.T) {
	kit := newTestKitNoKey(t)
	source := loadFixture(t, "testdata/as/arithmetic.ts")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compilePayload, _ := json.Marshal(map[string]string{"source": source})
	compileResp, err := kit.Bus.Request(ctx, "wasm.compile", kit.callerID, compilePayload)
	if err != nil {
		t.Fatal(err)
	}

	var compiled struct{ ModuleID string `json:"moduleId"` }
	json.Unmarshal(compileResp.Payload, &compiled)

	runPayload, _ := json.Marshal(map[string]string{"moduleId": compiled.ModuleID})
	runResp, err := kit.Bus.Request(ctx, "wasm.run", kit.callerID, runPayload)
	if err != nil {
		t.Fatal(err)
	}

	var result struct{ ExitCode int `json:"exitCode"` }
	json.Unmarshal(runResp.Payload, &result)

	if result.ExitCode != 43 {
		t.Errorf("exitCode = %d, want 43 (add(multiply(6,7),1))", result.ExitCode)
	}
	t.Logf("fixture as/arithmetic: add(multiply(6,7),1)=%d", result.ExitCode)
}

// ═══════════════════════════════════════════════════════════════
// COMPOSITION FIXTURE — .ts uses everything including WASM
// ═══════════════════════════════════════════════════════════════

func TestFixture_TS_FullComposition(t *testing.T) {
	kit := newTestKit(t)

	// Register the "reverse" tool the fixture expects
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.reverse", ShortName: "reverse", Namespace: "platform",
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
