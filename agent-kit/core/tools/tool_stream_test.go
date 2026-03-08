// Ported from: packages/core/src/tools/tool-stream.test.ts
package tools

import (
	"reflect"
	"sync"
	"testing"
)

// NOTE: The TypeScript tool-stream.test.ts tests are heavily dependent on Agent,
// Mastra, MockLanguageModelV2, Workflow, MockMemory, and streaming infrastructure
// (convertArrayToReadableStream, fullStream, etc.) that are not yet ported to Go.
//
// We port what is testable at the ToolStream level directly and skip tests that
// require the full agent/workflow/memory stack.

func TestToolStream_WriteData(t *testing.T) {
	t.Run("should write data with tool metadata wrapping", func(t *testing.T) {
		var captured []any
		var mu sync.Mutex

		ts := NewToolStream(ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-123",
			Name:   "myTool",
			RunID:  "run-456",
		}, func(data any) error {
			mu.Lock()
			defer mu.Unlock()
			captured = append(captured, data)
			return nil
		})

		err := ts.WriteData("hello world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		if len(captured) != 1 {
			t.Fatalf("expected 1 write, got %d", len(captured))
		}

		event, ok := captured[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map event, got %T", captured[0])
		}

		if event["type"] != "tool-output" {
			t.Errorf("expected type=tool-output, got %v", event["type"])
		}
		if event["runId"] != "run-456" {
			t.Errorf("expected runId=run-456, got %v", event["runId"])
		}
		if event["from"] != "USER" {
			t.Errorf("expected from=USER, got %v", event["from"])
		}

		payload, ok := event["payload"].(map[string]any)
		if !ok {
			t.Fatalf("expected payload map, got %T", event["payload"])
		}
		if payload["output"] != "hello world" {
			t.Errorf("expected output=hello world, got %v", payload["output"])
		}
		if payload["toolCallId"] != "call-123" {
			t.Errorf("expected toolCallId=call-123, got %v", payload["toolCallId"])
		}
		if payload["toolName"] != "myTool" {
			t.Errorf("expected toolName=myTool, got %v", payload["toolName"])
		}
	})

	t.Run("should use workflow-step prefix with runId and stepName", func(t *testing.T) {
		var captured []any
		var mu sync.Mutex

		ts := NewToolStream(ToolStreamConfig{
			Prefix: "workflow-step",
			CallID: "call-123",
			Name:   "generate-chapters",
			RunID:  "run-789",
		}, func(data any) error {
			mu.Lock()
			defer mu.Unlock()
			captured = append(captured, data)
			return nil
		})

		err := ts.WriteData(map[string]any{"status": "processing"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		if len(captured) != 1 {
			t.Fatalf("expected 1 write, got %d", len(captured))
		}

		event := captured[0].(map[string]any)
		if event["type"] != "workflow-step-output" {
			t.Errorf("expected type=workflow-step-output, got %v", event["type"])
		}

		payload := event["payload"].(map[string]any)
		if payload["runId"] != "run-789" {
			t.Errorf("expected runId=run-789, got %v", payload["runId"])
		}
		if payload["stepName"] != "generate-chapters" {
			t.Errorf("expected stepName=generate-chapters, got %v", payload["stepName"])
		}
	})
}

func TestToolStream_Custom(t *testing.T) {
	t.Run("should write custom data directly without metadata wrapping", func(t *testing.T) {
		var captured []any
		var mu sync.Mutex

		ts := NewToolStream(ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-custom-1",
			Name:   "customTool",
			RunID:  "run-001",
		}, func(data any) error {
			mu.Lock()
			defer mu.Unlock()
			captured = append(captured, data)
			return nil
		})

		customData := map[string]any{
			"type": "data-custom-progress",
			"data": map[string]any{
				"status":   "processing",
				"message":  "test",
				"progress": 50,
			},
		}

		err := ts.Custom(customData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		if len(captured) != 1 {
			t.Fatalf("expected 1 custom write, got %d", len(captured))
		}

		// Custom writes go through directly without metadata wrapping.
		event, ok := captured[0].(map[string]any)
		if !ok {
			t.Fatalf("expected map event, got %T", captured[0])
		}
		if event["type"] != "data-custom-progress" {
			t.Errorf("expected type=data-custom-progress, got %v", event["type"])
		}
		data, ok := event["data"].(map[string]any)
		if !ok {
			t.Fatalf("expected data map, got %T", event["data"])
		}
		if data["status"] != "processing" {
			t.Errorf("expected status=processing, got %v", data["status"])
		}
		if data["progress"] != 50 {
			t.Errorf("expected progress=50, got %v", data["progress"])
		}
	})
}

func TestToolStream_Write(t *testing.T) {
	t.Run("should implement io.Writer interface", func(t *testing.T) {
		var captured []any
		var mu sync.Mutex

		ts := NewToolStream(ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-io",
			Name:   "ioTool",
			RunID:  "run-io",
		}, func(data any) error {
			mu.Lock()
			defer mu.Unlock()
			captured = append(captured, data)
			return nil
		})

		n, err := ts.Write([]byte("raw bytes"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 9 {
			t.Errorf("expected 9 bytes written, got %d", n)
		}

		mu.Lock()
		defer mu.Unlock()

		if len(captured) != 1 {
			t.Fatalf("expected 1 write, got %d", len(captured))
		}

		event := captured[0].(map[string]any)
		payload := event["payload"].(map[string]any)
		if payload["output"] != "raw bytes" {
			t.Errorf("expected output='raw bytes', got %v", payload["output"])
		}
	})
}

func TestToolStream_NilWriteFn(t *testing.T) {
	t.Run("should silently discard writes when writeFn is nil", func(t *testing.T) {
		ts := NewToolStream(ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-nil",
			Name:   "nilTool",
			RunID:  "run-nil",
		}, nil)

		err := ts.WriteData("should not fail")
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}

		err = ts.Custom(map[string]any{"type": "data-test"})
		if err != nil {
			t.Fatalf("expected nil error for custom, got %v", err)
		}

		n, err := ts.Write([]byte("bytes"))
		if err != nil {
			t.Fatalf("expected nil error for Write, got %v", err)
		}
		if n != 5 {
			t.Errorf("expected 5 bytes, got %d", n)
		}
	})
}

func TestToolStream_MixedWriteAndCustom(t *testing.T) {
	t.Run("should handle both regular writes and custom data chunks", func(t *testing.T) {
		var captured []any
		var mu sync.Mutex

		ts := NewToolStream(ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-mixed",
			Name:   "mixedTool",
			RunID:  "run-mixed",
		}, func(data any) error {
			mu.Lock()
			defer mu.Unlock()
			captured = append(captured, data)
			return nil
		})

		// Regular write.
		err := ts.WriteData(map[string]any{"type": "status-update", "message": "Starting"})
		if err != nil {
			t.Fatalf("unexpected error on write: %v", err)
		}

		// Custom write.
		err = ts.Custom(map[string]any{
			"type": "data-processing-metrics",
			"data": map[string]any{"value": "test"},
		})
		if err != nil {
			t.Fatalf("unexpected error on custom: %v", err)
		}

		// Another regular write.
		err = ts.WriteData(map[string]any{"type": "status-update", "message": "Done"})
		if err != nil {
			t.Fatalf("unexpected error on write: %v", err)
		}

		mu.Lock()
		defer mu.Unlock()

		if len(captured) != 3 {
			t.Fatalf("expected 3 writes, got %d", len(captured))
		}

		// First and third should be wrapped (tool-output), second should be raw custom.
		first := captured[0].(map[string]any)
		if first["type"] != "tool-output" {
			t.Errorf("expected first type=tool-output, got %v", first["type"])
		}

		second := captured[1].(map[string]any)
		if second["type"] != "data-processing-metrics" {
			t.Errorf("expected second type=data-processing-metrics, got %v", second["type"])
		}

		third := captured[2].(map[string]any)
		if third["type"] != "tool-output" {
			t.Errorf("expected third type=tool-output, got %v", third["type"])
		}
	})
}

func TestToolStream_ConcurrentWrites(t *testing.T) {
	t.Run("should handle concurrent writes safely", func(t *testing.T) {
		var captured []any
		var mu sync.Mutex

		ts := NewToolStream(ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-concurrent",
			Name:   "concurrentTool",
			RunID:  "run-concurrent",
		}, func(data any) error {
			mu.Lock()
			defer mu.Unlock()
			captured = append(captured, data)
			return nil
		})

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				_ = ts.WriteData(map[string]any{"index": idx})
			}(i)
		}
		wg.Wait()

		mu.Lock()
		defer mu.Unlock()

		if len(captured) != 10 {
			t.Errorf("expected 10 writes, got %d", len(captured))
		}
	})
}

func TestToolStream_AgentStreamIntegration(t *testing.T) {
	t.Skip("not yet implemented - requires Agent, Mastra, and MockLanguageModelV2")
}

func TestToolStream_WorkflowStreamIntegration(t *testing.T) {
	t.Skip("not yet implemented - requires Workflow, createStep, createWorkflow")
}

func TestToolStream_SubAgentIntegration(t *testing.T) {
	t.Skip("not yet implemented - requires Agent with sub-agents")
}

func TestToolStream_MemoryPersistence(t *testing.T) {
	t.Skip("not yet implemented - requires MockMemory and agent memory integration")
}

func TestToolStreamConfig(t *testing.T) {
	t.Run("should store all config fields", func(t *testing.T) {
		cfg := ToolStreamConfig{
			Prefix: "tool",
			CallID: "call-1",
			Name:   "testTool",
			RunID:  "run-1",
		}

		ts := NewToolStream(cfg, nil)

		if ts.prefix != "tool" {
			t.Errorf("expected prefix=tool, got %v", ts.prefix)
		}
		if ts.callID != "call-1" {
			t.Errorf("expected callID=call-1, got %v", ts.callID)
		}
		if ts.name != "testTool" {
			t.Errorf("expected name=testTool, got %v", ts.name)
		}
		if ts.runID != "run-1" {
			t.Errorf("expected runID=run-1, got %v", ts.runID)
		}
	})
}

func TestToolStream_MetadataKeys(t *testing.T) {
	t.Run("should use prefix-based key names for non-workflow-step", func(t *testing.T) {
		var captured []any
		ts := NewToolStream(ToolStreamConfig{
			Prefix: "agent",
			CallID: "call-agent",
			Name:   "agentTool",
			RunID:  "run-agent",
		}, func(data any) error {
			captured = append(captured, data)
			return nil
		})

		_ = ts.WriteData("test")

		event := captured[0].(map[string]any)
		payload := event["payload"].(map[string]any)

		// Should have agentCallId and agentName.
		expected := map[string]any{
			"output":      "test",
			"agentCallId": "call-agent",
			"agentName":   "agentTool",
		}
		if !reflect.DeepEqual(payload, expected) {
			t.Errorf("expected payload %v, got %v", expected, payload)
		}
	})
}
