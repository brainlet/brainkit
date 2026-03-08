// Ported from: packages/core/src/stream/aisdk/v5/output-helpers.test.ts
package v5

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func boolPtrHelper(b bool) *bool {
	return &b
}

func TestDefaultStepResult(t *testing.T) {
	t.Run("Text should concatenate text parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "text", Text: "Hello"},
				{Type: "text", Text: " World"},
			},
		})
		if sr.Text() != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", sr.Text())
		}
	})

	t.Run("Text should return empty when tripwire is set", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "text", Text: "Hello"},
			},
			Tripwire: &StepTripwireData{Name: "pii", Message: "blocked"},
		})
		if sr.Text() != "" {
			t.Errorf("expected empty string when tripwire set, got %q", sr.Text())
		}
	})

	t.Run("Text should skip non-text parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "text", Text: "Hello"},
				{Type: "tool-call", ToolName: "search"},
				{Type: "text", Text: " World"},
			},
		})
		if sr.Text() != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", sr.Text())
		}
	})

	t.Run("Text should return empty for no content", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{})
		if sr.Text() != "" {
			t.Errorf("expected empty string, got %q", sr.Text())
		}
	})

	t.Run("Reasoning should return reasoning parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "reasoning", Text: "Let me think..."},
				{Type: "text", Text: "answer"},
				{Type: "reasoning", Text: " about this."},
			},
		})
		reasoning := sr.Reasoning()
		if len(reasoning) != 2 {
			t.Fatalf("expected 2 reasoning parts, got %d", len(reasoning))
		}
		if reasoning[0].Text != "Let me think..." {
			t.Errorf("expected first reasoning text, got %q", reasoning[0].Text)
		}
	})

	t.Run("ReasoningText should concatenate reasoning parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "reasoning", Text: "First "},
				{Type: "reasoning", Text: "second"},
			},
		})
		if sr.ReasoningText() != "First second" {
			t.Errorf("expected 'First second', got %q", sr.ReasoningText())
		}
	})

	t.Run("ReasoningText should return empty when no reasoning", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "text", Text: "no reasoning here"},
			},
		})
		if sr.ReasoningText() != "" {
			t.Errorf("expected empty string, got %q", sr.ReasoningText())
		}
	})

	t.Run("Files should return file parts", func(t *testing.T) {
		file1 := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      "iVBOR",
			MediaType: "image/png",
		})
		file2 := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      "/9j/",
			MediaType: "image/jpeg",
		})
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "file", File: file1},
				{Type: "text", Text: "description"},
				{Type: "file", File: file2},
			},
		})
		files := sr.Files()
		if len(files) != 2 {
			t.Fatalf("expected 2 files, got %d", len(files))
		}
		if files[0].MediaType() != "image/png" {
			t.Errorf("expected first file media type 'image/png', got %q", files[0].MediaType())
		}
	})

	t.Run("Files should skip file parts with nil file", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "file"}, // nil file
			},
		})
		files := sr.Files()
		if len(files) != 0 {
			t.Errorf("expected 0 files, got %d", len(files))
		}
	})

	t.Run("Sources should return source parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "source", SourceType: "url", URL: "https://example.com", Title: "Example"},
				{Type: "text", Text: "some text"},
				{Type: "source", SourceType: "document", Title: "Doc"},
			},
		})
		sources := sr.Sources()
		if len(sources) != 2 {
			t.Fatalf("expected 2 sources, got %d", len(sources))
		}
		if sources[0].SourceType != "url" {
			t.Errorf("expected sourceType 'url', got %q", sources[0].SourceType)
		}
	})

	t.Run("ToolCalls should return tool-call parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "tool-call", ToolCallID: "tc1", ToolName: "search", Input: map[string]any{"query": "test"}},
				{Type: "text", Text: "response"},
				{Type: "tool-call", ToolCallID: "tc2", ToolName: "calc"},
			},
		})
		calls := sr.ToolCalls()
		if len(calls) != 2 {
			t.Fatalf("expected 2 tool calls, got %d", len(calls))
		}
		if calls[0].ToolName != "search" {
			t.Errorf("expected toolName 'search', got %q", calls[0].ToolName)
		}
	})

	t.Run("StaticToolCalls should return calls where Dynamic is false", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "tool-call", ToolName: "static", Dynamic: boolPtrHelper(false)},
				{Type: "tool-call", ToolName: "dynamic", Dynamic: boolPtrHelper(true)},
				{Type: "tool-call", ToolName: "unset"}, // Dynamic is nil
			},
		})
		statics := sr.StaticToolCalls()
		if len(statics) != 1 {
			t.Fatalf("expected 1 static tool call, got %d", len(statics))
		}
		if statics[0].ToolName != "static" {
			t.Errorf("expected toolName 'static', got %q", statics[0].ToolName)
		}
	})

	t.Run("DynamicToolCalls should return calls where Dynamic is true", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "tool-call", ToolName: "static", Dynamic: boolPtrHelper(false)},
				{Type: "tool-call", ToolName: "dynamic", Dynamic: boolPtrHelper(true)},
			},
		})
		dynamics := sr.DynamicToolCalls()
		if len(dynamics) != 1 {
			t.Fatalf("expected 1 dynamic tool call, got %d", len(dynamics))
		}
		if dynamics[0].ToolName != "dynamic" {
			t.Errorf("expected toolName 'dynamic', got %q", dynamics[0].ToolName)
		}
	})

	t.Run("ToolResults should return tool-result parts", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "tool-result", ToolCallID: "tc1", Output: "found it"},
				{Type: "text", Text: "text"},
				{Type: "tool-result", ToolCallID: "tc2", Output: "calculated"},
			},
		})
		results := sr.ToolResults()
		if len(results) != 2 {
			t.Fatalf("expected 2 tool results, got %d", len(results))
		}
	})

	t.Run("StaticToolResults should return results where Dynamic is false", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "tool-result", ToolCallID: "tc1", Dynamic: boolPtrHelper(false)},
				{Type: "tool-result", ToolCallID: "tc2", Dynamic: boolPtrHelper(true)},
			},
		})
		statics := sr.StaticToolResults()
		if len(statics) != 1 {
			t.Fatalf("expected 1 static tool result, got %d", len(statics))
		}
	})

	t.Run("DynamicToolResults should return results where Dynamic is true", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Content: []ContentPart{
				{Type: "tool-result", ToolCallID: "tc1", Dynamic: boolPtrHelper(false)},
				{Type: "tool-result", ToolCallID: "tc2", Dynamic: boolPtrHelper(true)},
			},
		})
		dynamics := sr.DynamicToolResults()
		if len(dynamics) != 1 {
			t.Fatalf("expected 1 dynamic tool result, got %d", len(dynamics))
		}
	})

	t.Run("should preserve FinishReason", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			FinishReason: "stop",
		})
		if sr.FinishReason != "stop" {
			t.Errorf("expected finishReason 'stop', got %q", sr.FinishReason)
		}
	})

	t.Run("should preserve Usage", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			Usage: stream.LanguageModelUsage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		})
		if sr.Usage.InputTokens != 10 {
			t.Errorf("expected inputTokens 10, got %d", sr.Usage.InputTokens)
		}
	})

	t.Run("should preserve ProviderMetadata", func(t *testing.T) {
		sr := NewDefaultStepResult(StepResultData{
			ProviderMetadata: map[string]any{"key": "value"},
		})
		if sr.ProviderMetadata["key"] != "value" {
			t.Errorf("expected providerMetadata key, got %v", sr.ProviderMetadata["key"])
		}
	})
}
