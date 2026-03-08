// Ported from: packages/core/src/agent/message-list/conversion/output-converter-provider-executed.test.ts
package conversion

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/adapters"
)

// ---------------------------------------------------------------------------
// sanitizeV5UIMessages — provider-executed tool handling
// ---------------------------------------------------------------------------

func boolPtr(v bool) *bool { return &v }

func makeToolPart(typ string, toolCallID string, overrides ...func(*adapters.AIV5UIPart)) adapters.AIV5UIPart {
	p := adapters.AIV5UIPart{
		Type:       typ,
		ToolCallID: toolCallID,
		State:      "input-available",
		Input:      map[string]any{},
	}
	for _, fn := range overrides {
		fn(&p)
	}
	return p
}

func makeV5Message(parts []adapters.AIV5UIPart) *adapters.AIV5UIMessage {
	return &adapters.AIV5UIMessage{
		ID:    "msg-1",
		Role:  "assistant",
		Parts: parts,
	}
}

func TestSanitizeV5UIMessages_ProviderExecuted(t *testing.T) {
	t.Run("should filter out regular input-available tool parts when filterIncompleteToolCalls is true", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			makeToolPart("tool-get_info", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "input-available"
				p.Input = map[string]any{"name": "test"}
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})

	t.Run("should keep provider-executed input-available tool parts when filterIncompleteToolCalls is true", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			makeToolPart("tool-web_search_20250305", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "input-available"
				p.Input = map[string]any{"query": "test"}
				p.ProviderExecuted = boolPtr(true)
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if len(result[0].Parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(result[0].Parts))
		}
		if result[0].Parts[0].ToolCallID != "call-1" {
			t.Errorf("expected toolCallId call-1, got %s", result[0].Parts[0].ToolCallID)
		}
		if result[0].Parts[0].ProviderExecuted == nil || !*result[0].Parts[0].ProviderExecuted {
			t.Error("expected providerExecuted to be true")
		}
	})

	t.Run("should keep output-available parts for client-executed tools", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			makeToolPart("tool-get_info", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "output-available"
				p.Input = map[string]any{"name": "test"}
				p.Output = map[string]any{"company": "Acme"}
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if len(result[0].Parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(result[0].Parts))
		}
	})

	t.Run("should handle mid-loop parallel calls: keep client output-available + provider input-available, drop regular input-available", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			// Regular tool with result — keep
			makeToolPart("tool-get_company_info", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "output-available"
				p.Input = map[string]any{"name": "test"}
				p.Output = map[string]any{"company": "Acme"}
			}),
			// Provider-executed tool with no client result — keep
			makeToolPart("tool-web_search_20250305", "call-2", func(p *adapters.AIV5UIPart) {
				p.State = "input-available"
				p.Input = map[string]any{"query": "test"}
				p.ProviderExecuted = boolPtr(true)
			}),
			// Regular tool still pending — drop
			makeToolPart("tool-update_record", "call-3", func(p *adapters.AIV5UIPart) {
				p.State = "input-available"
				p.Input = map[string]any{"id": "123"}
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if len(result[0].Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(result[0].Parts))
		}

		callIDs := make(map[string]bool)
		for _, p := range result[0].Parts {
			callIDs[p.ToolCallID] = true
		}
		if !callIDs["call-1"] {
			t.Error("expected call-1 to be present")
		}
		if !callIDs["call-2"] {
			t.Error("expected call-2 to be present")
		}
		if callIDs["call-3"] {
			t.Error("expected call-3 to be filtered out")
		}
	})

	t.Run("should strip output-available provider-executed tool parts", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			makeToolPart("tool-web_search_20250305", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "output-available"
				p.Input = map[string]any{"query": "anthropic"}
				p.Output = map[string]any{"results": []any{"result1"}}
				p.ProviderExecuted = boolPtr(true)
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 0 {
			t.Errorf("expected 0 messages (entire message dropped), got %d", len(result))
		}
	})

	t.Run("should strip output-error provider-executed tool parts", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			makeToolPart("tool-web_search_20250305", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "output-error"
				p.Input = map[string]any{"query": "test"}
				p.ProviderExecuted = boolPtr(true)
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result))
		}
	})

	t.Run("should handle resume scenario: keep client output-available, strip completed provider output-available", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			// Client-executed tool with result — keep
			makeToolPart("tool-get_company_info", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "output-available"
				p.Input = map[string]any{"name": "test"}
				p.Output = map[string]any{"company": "Acme"}
			}),
			// Provider-executed tool already completed — strip
			makeToolPart("tool-web_search_20250305", "call-2", func(p *adapters.AIV5UIPart) {
				p.State = "output-available"
				p.Input = map[string]any{"query": "test"}
				p.Output = map[string]any{"results": []any{"result1"}}
				p.ProviderExecuted = boolPtr(true)
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, true)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if len(result[0].Parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(result[0].Parts))
		}

		callIDs := make(map[string]bool)
		for _, p := range result[0].Parts {
			callIDs[p.ToolCallID] = true
		}
		if !callIDs["call-1"] {
			t.Error("expected call-1 to be present")
		}
		if callIDs["call-2"] {
			t.Error("expected call-2 to be filtered out")
		}
	})

	t.Run("should not filter provider-executed tools when filterIncompleteToolCalls is false", func(t *testing.T) {
		msg := makeV5Message([]adapters.AIV5UIPart{
			makeToolPart("tool-web_search_20250305", "call-1", func(p *adapters.AIV5UIPart) {
				p.State = "input-available"
				p.Input = map[string]any{"query": "test"}
				p.ProviderExecuted = boolPtr(true)
			}),
			makeToolPart("tool-get_info", "call-2", func(p *adapters.AIV5UIPart) {
				p.State = "input-available"
				p.Input = map[string]any{"name": "test"}
			}),
		})

		result := SanitizeV5UIMessages([]*adapters.AIV5UIMessage{msg}, false)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if len(result[0].Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(result[0].Parts))
		}
	})
}
