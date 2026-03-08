// Ported from: packages/core/src/agent/message-list/conversion/to-prompt-tool-name-sanitization.test.ts
package conversion

import (
	"testing"
)

// ---------------------------------------------------------------------------
// AIV5ModelMessageToV2PromptMessage tool-name sanitization
// ---------------------------------------------------------------------------

func TestAIV5ModelMessageToV2PromptMessage_ToolNameSanitization(t *testing.T) {
	t.Run("sanitizes invalid tool names in tool-call parts", func(t *testing.T) {
		result, err := AIV5ModelMessageToV2PromptMessage(map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type":       "tool-call",
					"toolCallId": "call-1",
					"toolName":   "$FUNCTION_NAME",
					"input":      map[string]any{"query": "test"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["role"] != "assistant" {
			t.Errorf("expected role assistant, got %v", result["role"])
		}

		content, ok := result["content"].([]map[string]any)
		if !ok {
			t.Fatalf("expected content to be []map[string]any, got %T", result["content"])
		}
		if len(content) == 0 {
			t.Fatal("expected at least one content part")
		}

		if content[0]["type"] != "tool-call" {
			t.Errorf("expected type tool-call, got %v", content[0]["type"])
		}
		if content[0]["toolCallId"] != "call-1" {
			t.Errorf("expected toolCallId call-1, got %v", content[0]["toolCallId"])
		}
		if content[0]["toolName"] != "unknown_tool" {
			t.Errorf("expected toolName unknown_tool, got %v", content[0]["toolName"])
		}
	})

	t.Run("sanitizes invalid tool names in tool-result parts", func(t *testing.T) {
		result, err := AIV5ModelMessageToV2PromptMessage(map[string]any{
			"role": "tool",
			"content": []any{
				map[string]any{
					"type":       "tool-result",
					"toolCallId": "call-1",
					"toolName":   "$FUNCTION_NAME",
					"output":     map[string]any{"ok": true},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["role"] != "tool" {
			t.Errorf("expected role tool, got %v", result["role"])
		}

		content, ok := result["content"].([]map[string]any)
		if !ok {
			t.Fatalf("expected content to be []map[string]any, got %T", result["content"])
		}
		if len(content) == 0 {
			t.Fatal("expected at least one content part")
		}

		if content[0]["type"] != "tool-result" {
			t.Errorf("expected type tool-result, got %v", content[0]["type"])
		}
		if content[0]["toolCallId"] != "call-1" {
			t.Errorf("expected toolCallId call-1, got %v", content[0]["toolCallId"])
		}
		if content[0]["toolName"] != "unknown_tool" {
			t.Errorf("expected toolName unknown_tool, got %v", content[0]["toolName"])
		}
	})
}
