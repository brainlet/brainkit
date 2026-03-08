// Ported from: packages/ai/src/generate-text/to-response-messages.test.ts
package generatetext

import (
	"testing"
)

func TestToResponseMessages_TextOnly(t *testing.T) {
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Hello, world!", nil),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	if result[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", result[0].Role)
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	if parts[0].Type != "text" {
		t.Errorf("expected type 'text', got %q", parts[0].Type)
	}
	if parts[0].Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %q", parts[0].Text)
	}
}

func TestToResponseMessages_IncludeToolCalls(t *testing.T) {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "123",
		ToolName:   "testTool",
		Input:      map[string]interface{}{},
	}
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Using a tool", nil),
			NewToolCallContentPart(tc),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[1].Type != "tool-call" {
		t.Errorf("expected second part type 'tool-call', got %q", parts[1].Type)
	}
	if parts[1].ToolCallID != "123" {
		t.Errorf("expected toolCallId '123', got %q", parts[1].ToolCallID)
	}
}

func TestToResponseMessages_ToolResultsSeparateMessage(t *testing.T) {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "123",
		ToolName:   "testTool",
		Input:      map[string]interface{}{},
	}
	tr := ToolResult{
		Type:       "tool-result",
		ToolCallID: "123",
		ToolName:   "testTool",
		Input:      map[string]interface{}{},
		Output:     "Tool result",
	}
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Tool used", nil),
			NewToolCallContentPart(tc),
			NewToolResultContentPart(tr),
		},
		ToolSet{},
	)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].Role != "assistant" {
		t.Errorf("expected first message role 'assistant', got %q", result[0].Role)
	}
	if result[1].Role != "tool" {
		t.Errorf("expected second message role 'tool', got %q", result[1].Role)
	}

	toolParts, ok := result[1].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[1].Content)
	}
	if len(toolParts) != 1 {
		t.Fatalf("expected 1 tool part, got %d", len(toolParts))
	}
	if toolParts[0].Type != "tool-result" {
		t.Errorf("expected type 'tool-result', got %q", toolParts[0].Type)
	}
}

func TestToResponseMessages_ToolErrorsSeparateMessage(t *testing.T) {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "123",
		ToolName:   "testTool",
		Input:      map[string]interface{}{},
	}
	te := ToolError{
		Type:       "tool-error",
		ToolCallID: "123",
		ToolName:   "testTool",
		Input:      map[string]interface{}{},
		Error:      "Tool error",
	}
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Tool used", nil),
			NewToolCallContentPart(tc),
			NewToolErrorContentPart(te),
		},
		ToolSet{},
	)

	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[1].Role != "tool" {
		t.Errorf("expected second message role 'tool', got %q", result[1].Role)
	}
}

func TestToResponseMessages_ReasoningParts(t *testing.T) {
	result := ToResponseMessages(
		[]ContentPart{
			NewReasoningContentPart("Thinking text", ProviderMetadata{"testProvider": {"signature": "sig"}}),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	if parts[0].Type != "reasoning" {
		t.Errorf("expected type 'reasoning', got %q", parts[0].Type)
	}
	if parts[0].Text != "Thinking text" {
		t.Errorf("expected text 'Thinking text', got %q", parts[0].Text)
	}
}

func TestToResponseMessages_SkipEmptyText(t *testing.T) {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "123",
		ToolName:   "testTool",
		Input:      map[string]interface{}{},
	}
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("", nil),
			NewToolCallContentPart(tc),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	// Only tool-call should be present, text was empty
	if len(parts) != 1 {
		t.Fatalf("expected 1 part (empty text skipped), got %d", len(parts))
	}
	if parts[0].Type != "tool-call" {
		t.Errorf("expected type 'tool-call', got %q", parts[0].Type)
	}
}

func TestToResponseMessages_EmptyContent(t *testing.T) {
	result := ToResponseMessages([]ContentPart{}, ToolSet{})
	if len(result) != 0 {
		t.Errorf("expected 0 messages for empty content, got %d", len(result))
	}
}

func TestToResponseMessages_ProviderExecutedToolCall(t *testing.T) {
	tc := ToolCall{
		Type:             "tool-call",
		ToolCallID:       "srvtoolu_123",
		ToolName:         "web_search",
		Input:            map[string]interface{}{"query": "test"},
		ProviderExecuted: true,
	}
	tr := ToolResult{
		Type:             "tool-result",
		ToolCallID:       "srvtoolu_123",
		ToolName:         "web_search",
		Input:            map[string]interface{}{"query": "test"},
		Output:           []interface{}{map[string]interface{}{"url": "https://example.com"}},
		ProviderExecuted: true,
	}
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Search results", nil),
			NewToolCallContentPart(tc),
			NewToolResultContentPart(tr),
			NewTextContentPart("Based on results...", nil),
		},
		ToolSet{},
	)

	// Provider-executed tool results should stay in the assistant message
	if len(result) != 1 {
		t.Fatalf("expected 1 message (provider-executed stays in assistant), got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	// Should have: text, tool-call, tool-result, text
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(parts))
	}
}

func TestToResponseMessages_ToolApprovalRequest(t *testing.T) {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "123",
		ToolName:   "weather",
		Input:      map[string]interface{}{"city": "Tokyo"},
	}
	tar := ToolApprovalRequestOutput{
		Type:       "tool-approval-request",
		ApprovalID: "approval-1",
		ToolCall:   tc,
	}
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Let me check the weather", nil),
			NewToolCallContentPart(tc),
			NewToolApprovalRequestContentPart(tar),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}
	if parts[2].Type != "tool-approval-request" {
		t.Errorf("expected type 'tool-approval-request', got %q", parts[2].Type)
	}
	if parts[2].ApprovalID != "approval-1" {
		t.Errorf("expected approvalId 'approval-1', got %q", parts[2].ApprovalID)
	}
}

func TestToResponseMessages_FileParts(t *testing.T) {
	pngFile := NewDefaultGeneratedFile([]byte{137, 80, 78, 71, 13, 10, 26, 10}, "image/png")

	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Here is an image", nil),
			NewFileContentPart(pngFile, nil),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	if parts[1].Type != "file" {
		t.Errorf("expected type 'file', got %q", parts[1].Type)
	}
	if parts[1].MediaType != "image/png" {
		t.Errorf("expected mediaType 'image/png', got %q", parts[1].MediaType)
	}
}

func TestToResponseMessages_ProviderMetadataInTextParts(t *testing.T) {
	result := ToResponseMessages(
		[]ContentPart{
			NewTextContentPart("Here is a text", ProviderMetadata{"testProvider": {"signature": "sig"}}),
		},
		ToolSet{},
	)

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	parts, ok := result[0].Content.([]ModelMessageContent)
	if !ok {
		t.Fatalf("expected []ModelMessageContent, got %T", result[0].Content)
	}
	if parts[0].ProviderOptions == nil {
		t.Error("expected providerOptions to be set")
	}
	if parts[0].ProviderOptions["testProvider"]["signature"] != "sig" {
		t.Errorf("expected testProvider.signature 'sig', got %v", parts[0].ProviderOptions["testProvider"]["signature"])
	}
}
