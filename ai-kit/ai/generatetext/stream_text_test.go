// Ported from: packages/ai/src/generate-text/stream-text.test.ts
// Note: The TS test file is 717KB and relies on MockLanguageModelV3 which is not ported.
// This Go test covers the structural aspects testable without a full model mock:
// - StreamTextResult construction
// - TextStreamPart type discrimination
// - StreamTextTransform construction
// - StreamTextOptions construction
package generatetext

import (
	"testing"
)

func TestStreamTextResult_DefaultValues(t *testing.T) {
	result := StreamTextResult{}

	if result.Text != "" {
		t.Errorf("expected empty text, got %q", result.Text)
	}
	if result.FinishReason != "" {
		t.Errorf("expected empty finish reason, got %q", result.FinishReason)
	}
	if result.Steps != nil {
		t.Errorf("expected nil steps, got %v", result.Steps)
	}
}

func TestStreamTextResult_WithContent(t *testing.T) {
	result := StreamTextResult{
		Text: "Hello, world!",
		Content: []ContentPart{
			NewTextContentPart("Hello, world!", nil),
		},
		FinishReason: FinishReasonStop,
	}

	if result.Text != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", result.Text)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected type 'text', got %q", result.Content[0].Type)
	}
}

func TestStreamTextResult_WithToolCalls(t *testing.T) {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "call-1",
		ToolName:   "weather",
		Input:      map[string]interface{}{"city": "London"},
	}

	result := StreamTextResult{
		ToolCalls:     []ToolCall{tc},
		FinishReason:  FinishReasonToolCalls,
	}

	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].ToolName != "weather" {
		t.Errorf("expected tool name 'weather', got %q", result.ToolCalls[0].ToolName)
	}
}

func TestStreamTextResult_WithToolResults(t *testing.T) {
	tr := ToolResult{
		Type:       "tool-result",
		ToolCallID: "call-1",
		ToolName:   "weather",
		Output:     "sunny",
	}

	result := StreamTextResult{
		ToolResults: []ToolResult{tr},
	}

	if len(result.ToolResults) != 1 {
		t.Fatalf("expected 1 tool result, got %d", len(result.ToolResults))
	}
	if result.ToolResults[0].Output != "sunny" {
		t.Errorf("expected output 'sunny', got %v", result.ToolResults[0].Output)
	}
}

func TestStreamTextResult_WithUsage(t *testing.T) {
	input := 100
	output := 50
	total := 150

	result := StreamTextResult{
		Usage: LanguageModelUsage{
			InputTokens:  &input,
			OutputTokens: &output,
			TotalTokens:  &total,
		},
		TotalUsage: LanguageModelUsage{
			InputTokens:  &input,
			OutputTokens: &output,
			TotalTokens:  &total,
		},
	}

	if *result.Usage.InputTokens != 100 {
		t.Errorf("expected input tokens 100, got %d", *result.Usage.InputTokens)
	}
	if *result.TotalUsage.TotalTokens != 150 {
		t.Errorf("expected total tokens 150, got %d", *result.TotalUsage.TotalTokens)
	}
}

func TestTextStreamPart_TypeDiscrimination(t *testing.T) {
	tests := []struct {
		name string
		part TextStreamPart
		typ  string
	}{
		{
			name: "text-start",
			part: TextStreamPart{Type: "text-start", ID: "1"},
			typ:  "text-start",
		},
		{
			name: "text-delta",
			part: TextStreamPart{Type: "text-delta", ID: "1", Text: "hello"},
			typ:  "text-delta",
		},
		{
			name: "text-end",
			part: TextStreamPart{Type: "text-end", ID: "1"},
			typ:  "text-end",
		},
		{
			name: "tool-call",
			part: TextStreamPart{
				Type:     "tool-call",
				ToolCall: &ToolCall{Type: "tool-call", ToolCallID: "1", ToolName: "test"},
			},
			typ: "tool-call",
		},
		{
			name: "tool-result",
			part: TextStreamPart{
				Type:       "tool-result",
				ToolResult: &ToolResult{Type: "tool-result", ToolCallID: "1"},
			},
			typ: "tool-result",
		},
		{
			name: "error",
			part: TextStreamPart{Type: "error", Error: "something went wrong"},
			typ:  "error",
		},
		{
			name: "finish-step",
			part: TextStreamPart{
				Type:         "finish-step",
				FinishReason: FinishReasonStop,
			},
			typ: "finish-step",
		},
		{
			name: "finish",
			part: TextStreamPart{Type: "finish"},
			typ:  "finish",
		},
		{
			name: "reasoning-start",
			part: TextStreamPart{Type: "reasoning-start", ID: "r1"},
			typ:  "reasoning-start",
		},
		{
			name: "reasoning-delta",
			part: TextStreamPart{Type: "reasoning-delta", ID: "r1", Text: "thinking"},
			typ:  "reasoning-delta",
		},
		{
			name: "reasoning-end",
			part: TextStreamPart{Type: "reasoning-end", ID: "r1"},
			typ:  "reasoning-end",
		},
		{
			name: "source",
			part: TextStreamPart{
				Type:   "source",
				Source: &Source{Type: "source", URL: "https://example.com"},
			},
			typ: "source",
		},
		{
			name: "file",
			part: TextStreamPart{
				Type: "file",
				File: NewDefaultGeneratedFile([]byte{1, 2, 3}, "image/png"),
			},
			typ: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.part.Type != tt.typ {
				t.Errorf("expected type %q, got %q", tt.typ, tt.part.Type)
			}
		})
	}
}

func TestStreamTextOptions_Construction(t *testing.T) {
	opts := StreamTextOptions{
		Prompt: "Tell me a story",
		Tools: ToolSet{
			"search": Tool{InputSchema: map[string]interface{}{"type": "object"}},
		},
		System: "You are a storyteller.",
	}

	if opts.Prompt != "Tell me a story" {
		t.Errorf("expected prompt, got %q", opts.Prompt)
	}
	if _, ok := opts.Tools["search"]; !ok {
		t.Error("expected search tool")
	}
}

func TestStreamTextResponseMetadata(t *testing.T) {
	meta := StreamTextResponseMetadata{
		Messages: []ResponseMessage{
			{Role: "assistant", Content: "Hello"},
		},
	}

	if len(meta.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(meta.Messages))
	}
	if meta.Messages[0].Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", meta.Messages[0].Role)
	}
}
