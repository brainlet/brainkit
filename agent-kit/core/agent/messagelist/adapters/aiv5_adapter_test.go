// Ported from: packages/core/src/agent/message-list/adapters/AIV5Adapter-provider-executed.test.ts
// Ported from: packages/core/src/agent/message-list/adapters/AIV5Adapter-tool-name-sanitization.test.ts
package adapters

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// ---------------------------------------------------------------------------
// AIV5Adapter.toUIMessage — providerExecuted propagation
// ---------------------------------------------------------------------------

func makeDbMessageWithParts(parts []MastraMessagePart) *MastraDBMessage {
	return &MastraDBMessage{
		MastraMessageShared: state.MastraMessageShared{
			ID:        "msg-1",
			Role:      "assistant",
			CreatedAt: time.Now(),
		},
		Content: MastraMessageContentV2{
			Format: 2,
			Parts:  parts,
		},
	}
}

func findToolPartByCallID(parts []AIV5UIPart, callID string) *AIV5UIPart {
	for i := range parts {
		if parts[i].ToolCallID == callID {
			return &parts[i]
		}
	}
	return nil
}

func TestAIV5ToUIMessage_ProviderExecutedPropagation(t *testing.T) {
	t.Run("should propagate providerExecuted on input-available (call state) tool parts", func(t *testing.T) {
		pe := true
		dbMsg := makeDbMessageWithParts([]MastraMessagePart{
			{
				Type: "tool-invocation",
				ToolInvocation: &ToolInvocation{
					State:      "call",
					ToolCallID: "call-1",
					ToolName:   "web_search_20250305",
					Args:       map[string]any{"query": "test"},
				},
				ProviderExecuted: &pe,
			},
		})

		uiMsg := AIV5ToUIMessage(dbMsg)

		toolPart := findToolPartByCallID(uiMsg.Parts, "call-1")
		if toolPart == nil {
			t.Fatal("expected to find tool part with callID call-1")
		}
		if toolPart.State != "input-available" {
			t.Errorf("expected state input-available, got %s", toolPart.State)
		}
		if toolPart.ProviderExecuted == nil || !*toolPart.ProviderExecuted {
			t.Error("expected providerExecuted to be true")
		}
	})

	t.Run("should propagate providerExecuted on output-available (result state) tool parts", func(t *testing.T) {
		pe := true
		dbMsg := makeDbMessageWithParts([]MastraMessagePart{
			{
				Type: "tool-invocation",
				ToolInvocation: &ToolInvocation{
					State:      "result",
					ToolCallID: "call-1",
					ToolName:   "web_search_20250305",
					Args:       map[string]any{"query": "test"},
					Result:     []any{map[string]any{"url": "https://example.com", "title": "Result", "content": "data"}},
				},
				ProviderExecuted: &pe,
			},
		})

		uiMsg := AIV5ToUIMessage(dbMsg)

		toolPart := findToolPartByCallID(uiMsg.Parts, "call-1")
		if toolPart == nil {
			t.Fatal("expected to find tool part with callID call-1")
		}
		if toolPart.State != "output-available" {
			t.Errorf("expected state output-available, got %s", toolPart.State)
		}
		if toolPart.ProviderExecuted == nil || !*toolPart.ProviderExecuted {
			t.Error("expected providerExecuted to be true")
		}
	})

	t.Run("should NOT add providerExecuted when it is absent from the DB message part", func(t *testing.T) {
		dbMsg := makeDbMessageWithParts([]MastraMessagePart{
			{
				Type: "tool-invocation",
				ToolInvocation: &ToolInvocation{
					State:      "call",
					ToolCallID: "call-1",
					ToolName:   "get_company_info",
					Args:       map[string]any{"name": "test"},
				},
			},
		})

		uiMsg := AIV5ToUIMessage(dbMsg)

		toolPart := findToolPartByCallID(uiMsg.Parts, "call-1")
		if toolPart == nil {
			t.Fatal("expected to find tool part with callID call-1")
		}
		if toolPart.State != "input-available" {
			t.Errorf("expected state input-available, got %s", toolPart.State)
		}
		if toolPart.ProviderExecuted != nil {
			t.Error("expected providerExecuted to be nil (absent)")
		}
	})

	t.Run("should handle mixed parts: provider-executed and regular tools in the same message", func(t *testing.T) {
		pe := true
		dbMsg := makeDbMessageWithParts([]MastraMessagePart{
			{
				Type: "tool-invocation",
				ToolInvocation: &ToolInvocation{
					State:      "call",
					ToolCallID: "call-1",
					ToolName:   "web_search_20250305",
					Args:       map[string]any{"query": "test"},
				},
				ProviderExecuted: &pe,
			},
			{
				Type: "tool-invocation",
				ToolInvocation: &ToolInvocation{
					State:      "result",
					ToolCallID: "call-2",
					ToolName:   "get_company_info",
					Args:       map[string]any{"name": "test"},
					Result:     map[string]any{"company": "Acme"},
				},
			},
		})

		uiMsg := AIV5ToUIMessage(dbMsg)

		webSearchPart := findToolPartByCallID(uiMsg.Parts, "call-1")
		regularPart := findToolPartByCallID(uiMsg.Parts, "call-2")

		if webSearchPart == nil {
			t.Fatal("expected to find web search part")
		}
		if regularPart == nil {
			t.Fatal("expected to find regular part")
		}

		if webSearchPart.ProviderExecuted == nil || !*webSearchPart.ProviderExecuted {
			t.Error("expected web search providerExecuted to be true")
		}
		if regularPart.ProviderExecuted != nil {
			t.Error("expected regular part providerExecuted to be nil (absent)")
		}
	})
}

// ---------------------------------------------------------------------------
// AIV5Adapter tool-name sanitization
// ---------------------------------------------------------------------------

func TestAIV5Adapter_ToolNameSanitization(t *testing.T) {
	t.Run("sanitizes invalid tool names from model tool-call parts", func(t *testing.T) {
		dbMessage := AIV5FromModelMessage(map[string]any{
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type":       "tool-call",
					"toolCallId": "call-1",
					"toolName":   "$FUNCTION_NAME",
					"input":      map[string]any{"query": "test"},
				},
			},
		}, "")

		var toolPart *MastraMessagePart
		for i, part := range dbMessage.Content.Parts {
			if part.Type == "tool-invocation" && part.ToolInvocation != nil && part.ToolInvocation.ToolCallID == "call-1" {
				toolPart = &dbMessage.Content.Parts[i]
				break
			}
		}

		if toolPart == nil {
			t.Fatal("expected to find tool-invocation part")
		}
		if toolPart.ToolInvocation.ToolName != "unknown_tool" {
			t.Errorf("expected tool name unknown_tool, got %s", toolPart.ToolInvocation.ToolName)
		}

		if len(dbMessage.Content.ToolInvocations) == 0 {
			t.Fatal("expected at least one tool invocation")
		}
		if dbMessage.Content.ToolInvocations[0].ToolName != "unknown_tool" {
			t.Errorf("expected toolInvocations[0] tool name unknown_tool, got %s", dbMessage.Content.ToolInvocations[0].ToolName)
		}
	})

	t.Run("sanitizes invalid tool names from model tool-result parts without matching calls", func(t *testing.T) {
		dbMessage := AIV5FromModelMessage(map[string]any{
			"role": "tool",
			"content": []any{
				map[string]any{
					"type":       "tool-result",
					"toolCallId": "call-1",
					"toolName":   "$FUNCTION_NAME",
					"output":     map[string]any{"ok": true},
				},
			},
		}, "")

		if len(dbMessage.Content.ToolInvocations) == 0 {
			t.Fatal("expected at least one tool invocation")
		}
		if dbMessage.Content.ToolInvocations[0].ToolName != "unknown_tool" {
			t.Errorf("expected toolInvocations[0] tool name unknown_tool, got %s", dbMessage.Content.ToolInvocations[0].ToolName)
		}

		var toolPart *MastraMessagePart
		for i, part := range dbMessage.Content.Parts {
			if part.Type == "tool-invocation" && part.ToolInvocation != nil && part.ToolInvocation.ToolCallID == "call-1" {
				toolPart = &dbMessage.Content.Parts[i]
				break
			}
		}

		if toolPart == nil {
			t.Fatal("expected to find tool-invocation part")
		}
		if toolPart.ToolInvocation.ToolName != "unknown_tool" {
			t.Errorf("expected tool name unknown_tool, got %s", toolPart.ToolInvocation.ToolName)
		}
	})

	t.Run("sanitizes invalid tool names from UI tool parts", func(t *testing.T) {
		dbMessage := AIV5FromUIMessage(&AIV5UIMessage{
			ID:   "msg-1",
			Role: "assistant",
			Parts: []AIV5UIPart{
				{
					Type:       "tool-$FUNCTION_NAME",
					State:      "input-available",
					ToolCallID: "call-1",
					Input:      map[string]any{"query": "test"},
				},
			},
		})

		if len(dbMessage.Content.ToolInvocations) == 0 {
			t.Fatal("expected at least one tool invocation")
		}
		if dbMessage.Content.ToolInvocations[0].ToolName != "unknown_tool" {
			t.Errorf("expected toolInvocations[0] tool name unknown_tool, got %s", dbMessage.Content.ToolInvocations[0].ToolName)
		}

		var toolPart *MastraMessagePart
		for i, part := range dbMessage.Content.Parts {
			if part.Type == "tool-invocation" && part.ToolInvocation != nil && part.ToolInvocation.ToolCallID == "call-1" {
				toolPart = &dbMessage.Content.Parts[i]
				break
			}
		}

		if toolPart == nil {
			t.Fatal("expected to find tool-invocation part")
		}
		if toolPart.ToolInvocation.ToolName != "unknown_tool" {
			t.Errorf("expected tool name unknown_tool, got %s", toolPart.ToolInvocation.ToolName)
		}
	})
}
