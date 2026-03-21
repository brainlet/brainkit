//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/registry"
)

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
		Name: "brainlet/platform@1.0.0/multiply", ShortName: "multiply",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
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

func TestFixture_TS_AgentWithMemory(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-with-memory.js")

	result, err := kit.EvalModule(context.Background(), "agent-with-memory.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("agent didn't remember: %q", out.Text)
	}
	t.Logf("fixture agent-with-memory: %q remembers=%v", out.Text, out.Remembers)
}

func TestFixture_TS_AgentDynamicModel(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-dynamic-model.js")
	result, err := kit.EvalModule(context.Background(), "agent-dynamic-model.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Text  string `json:"text"`
		Works bool   `json:"works"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.Works {
		t.Errorf("dynamic model failed: %q", out.Text)
	}
	t.Logf("agent-dynamic-model: text=%q works=%v", out.Text, out.Works)
}

func TestFixture_TS_AgentDynamicInstructions(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-dynamic-instructions.js")
	result, err := kit.EvalModule(context.Background(), "agent-dynamic-instructions.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Text1     string `json:"text1"`
		Text2     string `json:"text2"`
		HasAlpha  bool   `json:"hasAlpha"`
		HasBeta   bool   `json:"hasBeta"`
		Different bool   `json:"different"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.HasAlpha {
		t.Errorf("expected ALPHA: %q", out.Text1)
	}
	if !out.HasBeta {
		t.Errorf("expected BETA: %q", out.Text2)
	}
	if !out.Different {
		t.Error("expected different responses for different contexts")
	}
	t.Logf("agent-dynamic-instructions: text1=%q text2=%q", out.Text1, out.Text2)
}

func TestFixture_TS_AgentDynamicTools(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-dynamic-tools.js")
	result, err := kit.EvalModule(context.Background(), "agent-dynamic-tools.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		AddResult       string `json:"addResult"`
		MultiplyResult  string `json:"multiplyResult"`
		AddCorrect      bool   `json:"addCorrect"`
		MultiplyCorrect bool   `json:"multiplyCorrect"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.AddCorrect {
		t.Errorf("add should be 7: %q", out.AddResult)
	}
	if !out.MultiplyCorrect {
		t.Errorf("multiply should be 12: %q", out.MultiplyResult)
	}
	t.Logf("agent-dynamic-tools: add=%q multiply=%q", out.AddResult, out.MultiplyResult)
}

func TestFixture_TS_AgentWithProcessor(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-with-processor.js")

	result, err := kit.EvalModule(context.Background(), "agent-with-processor.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text            string `json:"text"`
		ProcessorCalled bool   `json:"processorCalled"`
		Works           bool   `json:"works"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.ProcessorCalled {
		t.Error("processor was not called")
	}
	if !out.Works {
		t.Errorf("processor test failed: text=%q processorCalled=%v", out.Text, out.ProcessorCalled)
	}
	t.Logf("fixture agent-with-processor: text=%q processorCalled=%v works=%v", out.Text, out.ProcessorCalled, out.Works)
}

func TestFixture_TS_AgentWithTripwire(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/agent-with-tripwire.js")

	result, err := kit.EvalModule(context.Background(), "agent-with-tripwire.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Text         string `json:"text"`
		FinishReason string `json:"finishReason"`
		Tripped      bool   `json:"tripped"`
		Error        string `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Tripped {
		// tripwire may manifest as finishReason != "stop", or as a caught error
		t.Errorf("tripwire didn't fire: text=%q finishReason=%s error=%q", out.Text, out.FinishReason, out.Error)
	}
	t.Logf("fixture agent-with-tripwire: tripped=%v finishReason=%s text=%q error=%q",
		out.Tripped, out.FinishReason, out.Text, out.Error)
}
