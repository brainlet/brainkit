//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestAgentOptionsPassthrough tests that generate() options are passed through to Mastra:
// temperature, maxSteps, onStepFinish, onFinish, per-call instructions.
func TestAgentOptionsPassthrough(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/agent-options-passthrough.js")
	result, err := kit.EvalModule(context.Background(), "agent-options-passthrough.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]string
	json.Unmarshal([]byte(result), &out)
	t.Logf("Agent options: %v", out)

	for _, key := range []string{"temperature", "onStepFinish", "onFinish", "maxSteps"} {
		val := out[key]
		if val == "" || strings.HasPrefix(val, "error:") {
			t.Errorf("%s: %v", key, val)
		}
	}
}

// TestRunEvals tests batch evaluation with runEvals().
func TestRunEvals(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/eval-run-evals.js")
	result, err := kit.EvalModule(context.Background(), "eval-run-evals.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("runEvals: %v", out)

	if out["error"] != nil {
		t.Fatalf("runEvals error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if out["hasScores"] != "ok" {
		t.Errorf("hasScores: %v", out["hasScores"])
	}
	if v, ok := out["totalItems"].(float64); !ok || v != 3 {
		t.Errorf("totalItems: got %v, want 3", out["totalItems"])
	}
	if out["hasAccuracy"] != "ok" {
		t.Errorf("accuracy score should be a number: %v", out["hasAccuracy"])
	}
	if v, ok := out["accuracyScore"].(float64); ok {
		t.Logf("accuracy score: %.2f (1.0 = all items matched ground truth)", v)
	}
}

// TestProcessorsBuiltin tests that built-in processors are available and constructible.
func TestProcessorsBuiltin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/processors-builtin.js")
	result, err := kit.EvalModule(context.Background(), "processors-builtin.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Processors: %v", out)

	if out["unicodeNormalizer"] != "ok" {
		t.Errorf("UnicodeNormalizer: %v", out["unicodeNormalizer"])
	}
	if out["tokenLimiter"] != "ok" {
		t.Errorf("TokenLimiterProcessor: %v", out["tokenLimiter"])
	}
	if out["toolCallFilter"] != "ok" {
		t.Errorf("ToolCallFilter: %v", out["toolCallFilter"])
	}
	if out["batchParts"] != "ok" {
		t.Errorf("BatchPartsProcessor: %v", out["batchParts"])
	}
	if v, ok := out["availableCount"].(float64); !ok || v < 11 {
		t.Errorf("expected 11 processors available, got %v", out["availableCount"])
	}
	t.Logf("Available: %v", out["availableList"])
}

// TestAgentSubagents tests agent networks / sub-agent delegation.
func TestAgentSubagents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/agent-subagents.js")
	result, err := kit.EvalModule(context.Background(), "agent-subagents.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Subagents: %v", out)

	if out["error"] != nil {
		t.Fatalf("subagent error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if out["hasAnswer"] != "ok" {
		t.Errorf("should contain 105: %v", out["hasAnswer"])
	}
}

// TestAgentConstrainedSubagents tests the createSubagent() + subagents config pattern.
func TestAgentConstrainedSubagents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/agent-constrained-subagents.js")
	result, err := kit.EvalModule(context.Background(), "agent-constrained-subagents.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Constrained subagents: %v", out)

	if out["error"] != nil {
		t.Fatalf("error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if out["hasResponse"] != "ok" {
		t.Errorf("hasResponse: %v", out["hasResponse"])
	}
	if out["hasStartEvent"] != "ok" {
		t.Errorf("should have start event: %v", out["hasStartEvent"])
	}
	if out["hasEndEvent"] != "ok" {
		t.Errorf("should have end event: %v", out["hasEndEvent"])
	}
	if out["explorerUsed"] != "ok" {
		t.Errorf("explorer subagent should have been used: %v", out["explorerUsed"])
	}
	t.Logf("Events: %v", out["eventCount"])
}
