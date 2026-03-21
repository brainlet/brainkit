//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	mcppkg "github.com/brainlet/brainkit/mcp"
)

func TestFixture_TS_ToolFullConfig(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/tool-full-config.js")
	result, err := kit.EvalModule(context.Background(), "tool-full-config.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Text      string `json:"text"`
		HasAnswer bool   `json:"hasAnswer"`
		ToolCalls int    `json:"toolCalls"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.HasAnswer {
		t.Errorf("tool didn't compute 42: %q", out.Text)
	}
	if out.ToolCalls < 1 {
		t.Error("expected at least 1 tool call")
	}
	t.Logf("fixture tool-full-config: text=%q toolCalls=%d", out.Text, out.ToolCalls)
}

func TestFixture_TS_EvalLLMScorer(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/eval-llm-scorer.js")
	result, err := kit.EvalModule(context.Background(), "eval-llm-scorer.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Score     float64 `json:"score"`
		Reason    string  `json:"reason"`
		HasScore  bool    `json:"hasScore"`
		HasReason bool    `json:"hasReason"`
		Error     string  `json:"error"`
		Stack     string  `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("LLM scorer error: %s\n%s", out.Error, out.Stack)
	}
	if !out.HasScore {
		t.Errorf("expected score 0-1, got %v: %s", out.Score, result)
	}
	if !out.HasReason {
		t.Errorf("expected reason: %s", result)
	}
	t.Logf("eval-llm-scorer: score=%.2f reason=%q", out.Score, out.Reason)
}

func TestFixture_TS_ObservabilityTrace(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/observability-trace.js")
	result, err := kit.EvalModule(context.Background(), "observability-trace.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Text       string `json:"text"`
		HasTraceId bool   `json:"hasTraceId"`
		TraceId    string `json:"traceId"`
		HasRunId   bool   `json:"hasRunId"`
		RunId      string `json:"runId"`
		Works      bool   `json:"works"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Works {
		t.Errorf("agent didn't respond: %q", out.Text)
	}
	if !out.HasTraceId {
		t.Error("expected traceId — observability not active")
	}
	t.Logf("observability-trace: traceId=%s runId=%s", out.TraceId, out.RunId)
}

func TestFixture_TS_ObservabilitySpans(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/observability-spans.js")
	result, err := kit.EvalModule(context.Background(), "observability-spans.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Text               string   `json:"text"`
		HasAnswer          bool     `json:"hasAnswer"`
		ToolCalls          int      `json:"toolCalls"`
		TraceId            string   `json:"traceId"`
		RunId              string   `json:"runId"`
		HasTraceId         bool     `json:"hasTraceId"`
		HasUsage           bool     `json:"hasUsage"`
		HasTrace           bool     `json:"hasTrace"`
		SpanCount          int      `json:"spanCount"`
		SpanTypes          []string `json:"spanTypes"`
		SpanNames          []string `json:"spanNames"`
		HasAgentRun        bool     `json:"hasAgentRun"`
		HasModelGeneration bool     `json:"hasModelGeneration"`
		HasToolCall        bool     `json:"hasToolCall"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasTraceId {
		t.Error("expected 32-char hex traceId")
	}
	if !out.HasAnswer {
		t.Errorf("expected 42: %q", out.Text)
	}
	if !out.HasUsage {
		t.Error("expected token usage")
	}
	if !out.HasTrace {
		t.Error("expected spans persisted in storage")
	}
	if !out.HasAgentRun {
		t.Error("expected AGENT_RUN span")
	}
	if !out.HasModelGeneration {
		t.Error("expected MODEL_GENERATION span")
	}
	if !out.HasToolCall {
		t.Error("expected TOOL_CALL span")
	}
	t.Logf("observability-spans: traceId=%s %d spans types=%v",
		out.TraceId, out.SpanCount, out.SpanTypes)
}

func TestFixture_TS_MCPTools(t *testing.T) {
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not found — needed for MCP server test")
	}

	kit, err := New(Config{
		Namespace: "test",
		MCPServers: map[string]mcppkg.ServerConfig{
			"test": {
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-everything"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/mcp-tools.js")
	result, err := kit.EvalModule(context.Background(), "mcp-tools.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		ToolCount  int      `json:"toolCount"`
		ToolNames  []string `json:"toolNames"`
		EchoResult any      `json:"echoResult"`
		HasTools   bool     `json:"hasTools"`
		Error      string   `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("MCP error: %s", out.Error)
	}
	if !out.HasTools {
		t.Error("expected MCP tools")
	}
	t.Logf("mcp-tools: %d tools, names=%v echo=%v", out.ToolCount, out.ToolNames, out.EchoResult)
}

func TestFixture_TS_EvalCustomScorer(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/eval-custom-scorer.js")

	result, err := kit.EvalModule(context.Background(), "eval-custom-scorer.js", code)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("raw: %s", result)

	var out struct {
		CustomScore         float64 `json:"customScore"`
		CustomReason        string  `json:"customReason"`
		SimilarityExact     float64 `json:"similarityExact"`
		SimilarityDifferent float64 `json:"similarityDifferent"`
		AllCorrect          bool    `json:"allCorrect"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.AllCorrect {
		t.Errorf("scores incorrect: custom=%.2f exact=%.2f diff=%.2f", out.CustomScore, out.SimilarityExact, out.SimilarityDifferent)
	}
	t.Logf("fixture eval-custom-scorer: custom=%.2f(%s) exactSimilarity=%.2f diffSimilarity=%.2f",
		out.CustomScore, out.CustomReason, out.SimilarityExact, out.SimilarityDifferent)
}
