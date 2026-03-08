// Ported from: packages/core/src/evals/scoreTraces/utils.test.ts
package scoretraces

import (
	"testing"
	"time"
)

func TestBuildSpanTree(t *testing.T) {
	now := time.Now()

	t.Run("builds empty tree from empty spans", func(t *testing.T) {
		tree := BuildSpanTree([]SpanRecord{})
		if len(tree.SpanMap) != 0 {
			t.Errorf("SpanMap should be empty, got %d entries", len(tree.SpanMap))
		}
		if len(tree.ChildrenMap) != 0 {
			t.Errorf("ChildrenMap should be empty, got %d entries", len(tree.ChildrenMap))
		}
		if len(tree.RootSpans) != 0 {
			t.Errorf("RootSpans should be empty, got %d entries", len(tree.RootSpans))
		}
	})

	t.Run("identifies root spans (no parent)", func(t *testing.T) {
		spans := []SpanRecord{
			{SpanID: "root-1", StartedAt: now},
			{SpanID: "root-2", StartedAt: now.Add(time.Second)},
		}
		tree := BuildSpanTree(spans)
		if len(tree.RootSpans) != 2 {
			t.Fatalf("expected 2 root spans, got %d", len(tree.RootSpans))
		}
	})

	t.Run("builds parent-child relationships", func(t *testing.T) {
		parentID := "parent-1"
		spans := []SpanRecord{
			{SpanID: "parent-1", StartedAt: now},
			{SpanID: "child-1", ParentSpanID: &parentID, StartedAt: now.Add(time.Second)},
			{SpanID: "child-2", ParentSpanID: &parentID, StartedAt: now.Add(2 * time.Second)},
		}
		tree := BuildSpanTree(spans)

		if len(tree.RootSpans) != 1 {
			t.Fatalf("expected 1 root span, got %d", len(tree.RootSpans))
		}
		if tree.RootSpans[0].SpanID != "parent-1" {
			t.Errorf("root span ID = %q, want %q", tree.RootSpans[0].SpanID, "parent-1")
		}

		children := tree.ChildrenMap["parent-1"]
		if len(children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(children))
		}
		if children[0].SpanID != "child-1" {
			t.Errorf("first child = %q, want %q", children[0].SpanID, "child-1")
		}
		if children[1].SpanID != "child-2" {
			t.Errorf("second child = %q, want %q", children[1].SpanID, "child-2")
		}
	})

	t.Run("sorts children by startedAt", func(t *testing.T) {
		parentID := "parent"
		spans := []SpanRecord{
			{SpanID: "parent", StartedAt: now},
			{SpanID: "late-child", ParentSpanID: &parentID, StartedAt: now.Add(2 * time.Second)},
			{SpanID: "early-child", ParentSpanID: &parentID, StartedAt: now.Add(time.Second)},
		}
		tree := BuildSpanTree(spans)

		children := tree.ChildrenMap["parent"]
		if len(children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(children))
		}
		if children[0].SpanID != "early-child" {
			t.Errorf("first child = %q, want %q (should be sorted by startedAt)", children[0].SpanID, "early-child")
		}
		if children[1].SpanID != "late-child" {
			t.Errorf("second child = %q, want %q", children[1].SpanID, "late-child")
		}
	})

	t.Run("sorts root spans by startedAt", func(t *testing.T) {
		spans := []SpanRecord{
			{SpanID: "late-root", StartedAt: now.Add(time.Second)},
			{SpanID: "early-root", StartedAt: now},
		}
		tree := BuildSpanTree(spans)

		if tree.RootSpans[0].SpanID != "early-root" {
			t.Errorf("first root = %q, want %q", tree.RootSpans[0].SpanID, "early-root")
		}
	})

	t.Run("populates SpanMap for lookup", func(t *testing.T) {
		spans := []SpanRecord{
			{SpanID: "span-a", StartedAt: now},
			{SpanID: "span-b", StartedAt: now},
		}
		tree := BuildSpanTree(spans)

		if _, ok := tree.SpanMap["span-a"]; !ok {
			t.Error("SpanMap should contain span-a")
		}
		if _, ok := tree.SpanMap["span-b"]; !ok {
			t.Error("SpanMap should contain span-b")
		}
		if _, ok := tree.SpanMap["nonexistent"]; ok {
			t.Error("SpanMap should not contain nonexistent")
		}
	})
}

func TestValidateTrace(t *testing.T) {
	now := time.Now()

	t.Run("returns error for nil trace", func(t *testing.T) {
		err := ValidateTrace(nil)
		if err == nil {
			t.Fatal("expected error for nil trace")
		}
	})

	t.Run("returns error for nil spans", func(t *testing.T) {
		err := ValidateTrace(&TraceRecord{TraceID: "t1"})
		if err == nil {
			t.Fatal("expected error for nil spans")
		}
	})

	t.Run("returns error for empty spans", func(t *testing.T) {
		err := ValidateTrace(&TraceRecord{
			TraceID: "t1",
			Spans:   []SpanRecord{},
		})
		if err == nil {
			t.Fatal("expected error for empty spans")
		}
	})

	t.Run("returns nil for valid trace", func(t *testing.T) {
		err := ValidateTrace(&TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{SpanID: "s1", StartedAt: now},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error for orphan parent reference", func(t *testing.T) {
		nonExistentParent := "nonexistent"
		err := ValidateTrace(&TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{SpanID: "s1", ParentSpanID: &nonExistentParent, StartedAt: now},
			},
		})
		if err == nil {
			t.Fatal("expected error for orphan parent reference")
		}
	})

	t.Run("accepts valid parent-child relationships", func(t *testing.T) {
		parentID := "parent"
		err := ValidateTrace(&TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{SpanID: "parent", StartedAt: now},
				{SpanID: "child", ParentSpanID: &parentID, StartedAt: now},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestIsSpanMessage(t *testing.T) {
	t.Run("returns true for valid span message", func(t *testing.T) {
		msg, ok := isSpanMessage(map[string]any{
			"role":    "user",
			"content": "hello",
		})
		if !ok {
			t.Fatal("should recognize valid span message")
		}
		if msg.Role != "user" {
			t.Errorf("Role = %q, want %q", msg.Role, "user")
		}
	})

	t.Run("returns false for missing role", func(t *testing.T) {
		_, ok := isSpanMessage(map[string]any{
			"content": "hello",
		})
		if ok {
			t.Error("should not recognize message without role")
		}
	})

	t.Run("returns false for missing content", func(t *testing.T) {
		_, ok := isSpanMessage(map[string]any{
			"role": "user",
		})
		if ok {
			t.Error("should not recognize message without content")
		}
	})

	t.Run("returns false for non-map value", func(t *testing.T) {
		_, ok := isSpanMessage("not a map")
		if ok {
			t.Error("should not recognize string as span message")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		_, ok := isSpanMessage(nil)
		if ok {
			t.Error("should not recognize nil as span message")
		}
	})
}

func TestHasMessagesArray(t *testing.T) {
	t.Run("returns true for map with messages array", func(t *testing.T) {
		msgs, ok := hasMessagesArray(map[string]any{
			"messages": []any{
				map[string]any{"role": "user", "content": "hi"},
			},
		})
		if !ok {
			t.Fatal("should recognize messages array")
		}
		if len(msgs) != 1 {
			t.Errorf("expected 1 message, got %d", len(msgs))
		}
	})

	t.Run("returns false for non-map value", func(t *testing.T) {
		_, ok := hasMessagesArray("not a map")
		if ok {
			t.Error("should not recognize string")
		}
	})

	t.Run("returns false for map without messages", func(t *testing.T) {
		_, ok := hasMessagesArray(map[string]any{"other": "value"})
		if ok {
			t.Error("should not recognize map without messages key")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		_, ok := hasMessagesArray(nil)
		if ok {
			t.Error("should not recognize nil")
		}
	})
}

func TestHasTextProperty(t *testing.T) {
	t.Run("returns true for map with text string", func(t *testing.T) {
		text, ok := hasTextProperty(map[string]any{"text": "hello"})
		if !ok {
			t.Fatal("should recognize text property")
		}
		if text != "hello" {
			t.Errorf("text = %q, want %q", text, "hello")
		}
	})

	t.Run("returns false for non-string text", func(t *testing.T) {
		_, ok := hasTextProperty(map[string]any{"text": 42})
		if ok {
			t.Error("should not accept non-string text")
		}
	})

	t.Run("returns false for missing text", func(t *testing.T) {
		_, ok := hasTextProperty(map[string]any{"other": "value"})
		if ok {
			t.Error("should not accept map without text key")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		_, ok := hasTextProperty(nil)
		if ok {
			t.Error("should not recognize nil")
		}
	})
}

func TestNormalizeMessageContent(t *testing.T) {
	t.Run("returns string content as-is", func(t *testing.T) {
		result := normalizeMessageContent("hello world")
		if result != "hello world" {
			t.Errorf("got %q, want %q", result, "hello world")
		}
	})

	t.Run("returns last text part from array", func(t *testing.T) {
		result := normalizeMessageContent([]any{
			map[string]any{"type": "text", "text": "first"},
			map[string]any{"type": "text", "text": "second"},
		})
		if result != "second" {
			t.Errorf("got %q, want %q", result, "second")
		}
	})

	t.Run("ignores non-text parts", func(t *testing.T) {
		result := normalizeMessageContent([]any{
			map[string]any{"type": "image", "url": "http://example.com"},
			map[string]any{"type": "text", "text": "caption"},
		})
		if result != "caption" {
			t.Errorf("got %q, want %q", result, "caption")
		}
	})

	t.Run("returns empty string for empty array", func(t *testing.T) {
		result := normalizeMessageContent([]any{})
		if result != "" {
			t.Errorf("got %q, want empty string", result)
		}
	})

	t.Run("returns empty string for nil", func(t *testing.T) {
		result := normalizeMessageContent(nil)
		if result != "" {
			t.Errorf("got %q, want empty string", result)
		}
	})

	t.Run("returns empty string for unrecognized type", func(t *testing.T) {
		result := normalizeMessageContent(42)
		if result != "" {
			t.Errorf("got %q, want empty string", result)
		}
	})
}

func TestCreateMastraDBMessage(t *testing.T) {
	now := time.Now()

	t.Run("creates message with string content", func(t *testing.T) {
		msg := createMastraDBMessage("user", "hello", now, "msg-1")
		if msg.Role != "user" {
			t.Errorf("Role = %q, want %q", msg.Role, "user")
		}
		if msg.ID != "msg-1" {
			t.Errorf("ID = %q, want %q", msg.ID, "msg-1")
		}
		if msg.Content.Content != "hello" {
			t.Errorf("Content.Content = %q, want %q", msg.Content.Content, "hello")
		}
		if msg.Content.Format != 2 {
			t.Errorf("Content.Format = %d, want 2", msg.Content.Format)
		}
		if len(msg.Content.Parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(msg.Content.Parts))
		}
	})

	t.Run("normalizes array content", func(t *testing.T) {
		content := []any{
			map[string]any{"type": "text", "text": "normalized"},
		}
		msg := createMastraDBMessage("assistant", content, now, "")
		if msg.Content.Content != "normalized" {
			t.Errorf("Content.Content = %q, want %q", msg.Content.Content, "normalized")
		}
	})
}

func TestTransformTraceToScorerInputAndOutput(t *testing.T) {
	now := time.Now()
	endedAt := now.Add(time.Second)

	t.Run("returns error for nil trace", func(t *testing.T) {
		_, _, err := TransformTraceToScorerInputAndOutput(nil)
		if err == nil {
			t.Fatal("expected error for nil trace")
		}
	})

	t.Run("returns error for empty spans", func(t *testing.T) {
		_, _, err := TransformTraceToScorerInputAndOutput(&TraceRecord{
			TraceID: "t1",
			Spans:   []SpanRecord{},
		})
		if err == nil {
			t.Fatal("expected error for empty spans")
		}
	})

	t.Run("returns error when no root agent_run span exists", func(t *testing.T) {
		_, _, err := TransformTraceToScorerInputAndOutput(&TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{SpanID: "s1", SpanTyp: "generic", StartedAt: now},
			},
		})
		if err == nil {
			t.Fatal("expected error when no agent_run span exists")
		}
	})

	t.Run("returns error when root agent span has no output", func(t *testing.T) {
		agentSpanID := "agent-1"
		_, _, err := TransformTraceToScorerInputAndOutput(&TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{SpanID: agentSpanID, SpanTyp: "agent_run", StartedAt: now},
				{SpanID: "llm-1", SpanTyp: "model_generation", ParentSpanID: &agentSpanID, StartedAt: now},
			},
		})
		if err == nil {
			t.Fatal("expected error when agent span has no output")
		}
	})

	t.Run("transforms simple trace with string input and text output", func(t *testing.T) {
		agentSpanID := "agent-1"
		trace := &TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{
					SpanID:    agentSpanID,
					SpanTyp:  "agent_run",
					StartedAt: now,
					EndedAt:   &endedAt,
					Input:     "What is Go?",
					Output:    map[string]any{"text": "Go is a programming language."},
				},
				{
					SpanID:       "llm-1",
					SpanTyp:     "model_generation",
					ParentSpanID: &agentSpanID,
					StartedAt:    now,
					Input: map[string]any{
						"messages": []any{
							map[string]any{"role": "system", "content": "You are helpful."},
							map[string]any{"role": "user", "content": "What is Go?"},
						},
					},
				},
			},
		}

		input, output, err := TransformTraceToScorerInputAndOutput(trace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify input messages.
		if len(input.InputMessages) != 1 {
			t.Fatalf("expected 1 input message, got %d", len(input.InputMessages))
		}
		if input.InputMessages[0].Content.Content != "What is Go?" {
			t.Errorf("input content = %q, want %q", input.InputMessages[0].Content.Content, "What is Go?")
		}

		// Verify system messages.
		if len(input.SystemMessages) != 1 {
			t.Fatalf("expected 1 system message, got %d", len(input.SystemMessages))
		}
		if input.SystemMessages[0].Content != "You are helpful." {
			t.Errorf("system message = %q, want %q", input.SystemMessages[0].Content, "You are helpful.")
		}

		// Verify output.
		if len(output) != 1 {
			t.Fatalf("expected 1 output message, got %d", len(output))
		}
		if output[0].Role != "assistant" {
			t.Errorf("output role = %q, want %q", output[0].Role, "assistant")
		}
		if output[0].Content.Content != "Go is a programming language." {
			t.Errorf("output content = %q, want %q", output[0].Content.Content, "Go is a programming language.")
		}
	})

	t.Run("extracts tool invocations from tool call spans", func(t *testing.T) {
		agentSpanID := "agent-1"
		toolName := "search"
		trace := &TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{
					SpanID:    agentSpanID,
					SpanTyp:  "agent_run",
					StartedAt: now,
					EndedAt:   &endedAt,
					Input:     "search for Go",
					Output:    map[string]any{"text": "Found results."},
				},
				{
					SpanID:       "llm-1",
					SpanTyp:     "model_generation",
					ParentSpanID: &agentSpanID,
					StartedAt:    now,
					Input: map[string]any{
						"messages": []any{},
					},
				},
				{
					SpanID:       "tool-1",
					SpanTyp:     "tool_call",
					ParentSpanID: &agentSpanID,
					StartedAt:    now,
					EntityName:   &toolName,
					EntityID:     &toolName,
					Input:        map[string]any{"query": "Go"},
					Output:       map[string]any{"results": []any{"Go lang"}},
				},
			},
		}

		_, output, err := TransformTraceToScorerInputAndOutput(trace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(output) != 1 {
			t.Fatalf("expected 1 output message, got %d", len(output))
		}
		if output[0].Content.ToolInvocations == nil {
			t.Fatal("ToolInvocations should not be nil")
		}
		if len(output[0].Content.ToolInvocations) != 1 {
			t.Fatalf("expected 1 tool invocation, got %d", len(output[0].Content.ToolInvocations))
		}
	})

	t.Run("extracts remembered messages excluding current input and system", func(t *testing.T) {
		agentSpanID := "agent-1"
		trace := &TraceRecord{
			TraceID: "t1",
			Spans: []SpanRecord{
				{
					SpanID:    agentSpanID,
					SpanTyp:  "agent_run",
					StartedAt: now,
					EndedAt:   &endedAt,
					Input:     "current question",
					Output:    map[string]any{"text": "answer"},
				},
				{
					SpanID:       "llm-1",
					SpanTyp:     "model_generation",
					ParentSpanID: &agentSpanID,
					StartedAt:    now,
					Input: map[string]any{
						"messages": []any{
							map[string]any{"role": "system", "content": "system prompt"},
							map[string]any{"role": "user", "content": "previous question"},
							map[string]any{"role": "assistant", "content": "previous answer"},
							map[string]any{"role": "user", "content": "current question"},
						},
					},
				},
			},
		}

		input, _, err := TransformTraceToScorerInputAndOutput(trace)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have remembered messages excluding system and current input.
		if len(input.RememberedMessages) != 2 {
			t.Fatalf("expected 2 remembered messages, got %d", len(input.RememberedMessages))
		}
		if input.RememberedMessages[0].Content.Content != "previous question" {
			t.Errorf("first remembered = %q, want %q", input.RememberedMessages[0].Content.Content, "previous question")
		}
		if input.RememberedMessages[1].Content.Content != "previous answer" {
			t.Errorf("second remembered = %q, want %q", input.RememberedMessages[1].Content.Content, "previous answer")
		}
	})
}

func TestGetChildrenOfType(t *testing.T) {
	now := time.Now()
	parentID := "parent"

	t.Run("filters children by span type", func(t *testing.T) {
		spans := []SpanRecord{
			{SpanID: "parent", SpanTyp: "agent_run", StartedAt: now},
			{SpanID: "tool-1", SpanTyp: "tool_call", ParentSpanID: &parentID, StartedAt: now},
			{SpanID: "model-1", SpanTyp: "model_generation", ParentSpanID: &parentID, StartedAt: now},
			{SpanID: "tool-2", SpanTyp: "tool_call", ParentSpanID: &parentID, StartedAt: now.Add(time.Second)},
		}
		tree := BuildSpanTree(spans)

		toolChildren := getChildrenOfType(tree, "parent", "tool_call")
		if len(toolChildren) != 2 {
			t.Fatalf("expected 2 tool_call children, got %d", len(toolChildren))
		}

		modelChildren := getChildrenOfType(tree, "parent", "model_generation")
		if len(modelChildren) != 1 {
			t.Fatalf("expected 1 model_generation child, got %d", len(modelChildren))
		}
	})

	t.Run("returns nil for no matching children", func(t *testing.T) {
		spans := []SpanRecord{
			{SpanID: "parent", SpanTyp: "agent_run", StartedAt: now},
		}
		tree := BuildSpanTree(spans)

		result := getChildrenOfType(tree, "parent", "tool_call")
		if len(result) != 0 {
			t.Errorf("expected 0 children, got %d", len(result))
		}
	})

	t.Run("returns nil for nonexistent parent", func(t *testing.T) {
		tree := BuildSpanTree([]SpanRecord{})
		result := getChildrenOfType(tree, "nonexistent", "tool_call")
		if len(result) != 0 {
			t.Errorf("expected 0 children, got %d", len(result))
		}
	})
}
