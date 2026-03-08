// Ported from: packages/core/src/harness/subagent-tool.test.ts
package harness

import (
	"reflect"
	"testing"
)

func TestCreateSubagentToolRequestContextForwarding(t *testing.T) {
	t.Skip("not yet implemented - requires createSubagentTool function, Agent.Stream method, and RequestContext integration")

	// The TS tests verify:
	// 1. Forwards the parent requestContext to subagent.stream()
	// 2. Forwards requestContext even when harness context is not set
	// 3. Passes maxSteps, abortSignal, and requireToolApproval alongside requestContext
	// 4. Does not default maxSteps when stopWhen is configured
	// 5. Forwards default RequestContext when parent context has no explicit requestContext
	//
	// These require:
	// - createSubagentTool() function
	// - Agent type with Stream() method
	// - RequestContext type and integration
	// - Tool execute interface
}

// TestParseSubagentMeta tests the ParseSubagentMeta function which IS available in Go.
func TestParseSubagentMeta(t *testing.T) {
	t.Run("parses metadata from result string", func(t *testing.T) {
		content := "Some result text\n<subagent-meta modelId=\"gpt-4o\" durationMs=\"1234\" tools=\"read_file:ok,write_file:err\" />"
		meta := ParseSubagentMeta(content)

		if meta.Text != "Some result text" {
			t.Errorf("expected Text = %q, got %q", "Some result text", meta.Text)
		}
		if meta.ModelID != "gpt-4o" {
			t.Errorf("expected ModelID = %q, got %q", "gpt-4o", meta.ModelID)
		}
		if meta.DurationMs != 1234 {
			t.Errorf("expected DurationMs = 1234, got %d", meta.DurationMs)
		}
		if len(meta.ToolCalls) != 2 {
			t.Fatalf("expected 2 tool calls, got %d", len(meta.ToolCalls))
		}
		if meta.ToolCalls[0].Name != "read_file" || meta.ToolCalls[0].IsError {
			t.Errorf("expected first tool call = {read_file, false}, got %+v", meta.ToolCalls[0])
		}
		if meta.ToolCalls[1].Name != "write_file" || !meta.ToolCalls[1].IsError {
			t.Errorf("expected second tool call = {write_file, true}, got %+v", meta.ToolCalls[1])
		}
	})

	t.Run("returns full text when no metadata tag present", func(t *testing.T) {
		content := "Just some plain text without metadata"
		meta := ParseSubagentMeta(content)

		if meta.Text != content {
			t.Errorf("expected Text = %q, got %q", content, meta.Text)
		}
		if meta.ModelID != "" {
			t.Errorf("expected empty ModelID, got %q", meta.ModelID)
		}
		if meta.DurationMs != 0 {
			t.Errorf("expected DurationMs = 0, got %d", meta.DurationMs)
		}
		if meta.ToolCalls != nil {
			t.Errorf("expected nil ToolCalls, got %v", meta.ToolCalls)
		}
	})

	t.Run("handles empty tools string", func(t *testing.T) {
		content := "Result\n<subagent-meta modelId=\"claude\" durationMs=\"500\" tools=\"\" />"
		meta := ParseSubagentMeta(content)

		if meta.Text != "Result" {
			t.Errorf("expected Text = %q, got %q", "Result", meta.Text)
		}
		if meta.ModelID != "claude" {
			t.Errorf("expected ModelID = %q, got %q", "claude", meta.ModelID)
		}
		if meta.ToolCalls != nil {
			t.Errorf("expected nil ToolCalls for empty tools string, got %v", meta.ToolCalls)
		}
	})
}

// TestBuildSubagentMeta tests the BuildSubagentMeta function which IS available in Go.
func TestBuildSubagentMeta(t *testing.T) {
	t.Run("builds metadata tag", func(t *testing.T) {
		toolCalls := []SubagentToolCall{
			{Name: "read_file", IsError: false},
			{Name: "write_file", IsError: true},
		}

		result := BuildSubagentMeta("gpt-4o", 1234, toolCalls)
		expected := "\n<subagent-meta modelId=\"gpt-4o\" durationMs=\"1234\" tools=\"read_file:ok,write_file:err\" />"

		if result != expected {
			t.Errorf("BuildSubagentMeta mismatch\ngot:  %q\nwant: %q", result, expected)
		}
	})

	t.Run("roundtrips with ParseSubagentMeta", func(t *testing.T) {
		toolCalls := []SubagentToolCall{
			{Name: "search", IsError: false},
		}
		tag := BuildSubagentMeta("test-model", 999, toolCalls)
		content := "Hello world" + tag

		meta := ParseSubagentMeta(content)
		if meta.Text != "Hello world" {
			t.Errorf("expected Text = %q, got %q", "Hello world", meta.Text)
		}
		if meta.ModelID != "test-model" {
			t.Errorf("expected ModelID = %q, got %q", "test-model", meta.ModelID)
		}
		if meta.DurationMs != 999 {
			t.Errorf("expected DurationMs = 999, got %d", meta.DurationMs)
		}
		if !reflect.DeepEqual(meta.ToolCalls, toolCalls) {
			t.Errorf("expected ToolCalls = %+v, got %+v", toolCalls, meta.ToolCalls)
		}
	})
}
