// Ported from: packages/ai/src/generate-text/run-tools-transformation.test.ts
package generatetext

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func mockIDGenerator(prefix string) IdGenerator {
	var counter int64
	return func() string {
		n := atomic.AddInt64(&counter, 1)
		return fmt.Sprintf("%s-%d", prefix, n)
	}
}

func makeGeneratorStream(parts []LanguageModelV4StreamPart) <-chan LanguageModelV4StreamPart {
	ch := make(chan LanguageModelV4StreamPart, len(parts))
	for _, p := range parts {
		ch <- p
	}
	close(ch)
	return ch
}

var testUsage = struct {
	InputTokens  TokenCount
	OutputTokens TokenCount
}{
	InputTokens:  TokenCount{Total: 3},
	OutputTokens: TokenCount{Total: 10},
}

func collectParts(ch <-chan SingleRequestTextStreamPart, timeout time.Duration) []SingleRequestTextStreamPart {
	var result []SingleRequestTextStreamPart
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case part, ok := <-ch:
			if !ok {
				return result
			}
			result = append(result, part)
		case <-timer.C:
			return result
		}
	}
}

func TestRunToolsTransformation_ForwardTextParts(t *testing.T) {
	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", ID: "1", Delta: "text"},
		{Type: "text-end", ID: "1"},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonStop, Raw: "stop"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools:           nil,
		Messages:        []ModelMessage{},
		GenerateID:      mockIDGenerator("id"),
	}), 5*time.Second)

	// Should contain text-start, text-delta, text-end, finish
	types := make([]string, len(result))
	for i, r := range result {
		types[i] = r.Type
	}

	foundTextStart := false
	foundTextDelta := false
	foundTextEnd := false
	foundFinish := false
	for _, tp := range types {
		switch tp {
		case "text-start":
			foundTextStart = true
		case "text-delta":
			foundTextDelta = true
		case "text-end":
			foundTextEnd = true
		case "finish":
			foundFinish = true
		}
	}
	if !foundTextStart {
		t.Error("expected text-start")
	}
	if !foundTextDelta {
		t.Error("expected text-delta")
	}
	if !foundTextEnd {
		t.Error("expected text-end")
	}
	if !foundFinish {
		t.Error("expected finish")
	}
}

func TestRunToolsTransformation_ForwardFileParts(t *testing.T) {
	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:      "file",
			Data:      "Hello World",
			MediaType: "text/plain",
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonStop, Raw: "stop"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools:           nil,
		Messages:        []ModelMessage{},
		GenerateID:      mockIDGenerator("id"),
	}), 5*time.Second)

	foundFile := false
	for _, r := range result {
		if r.Type == "file" {
			foundFile = true
			if r.File == nil {
				t.Error("expected file to not be nil")
			}
		}
	}
	if !foundFile {
		t.Error("expected file part")
	}
}

func TestRunToolsTransformation_ForwardSourceParts(t *testing.T) {
	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type: "source",
			Source: &Source{
				Type:  "source",
				ID:    "src-1",
				URL:   "https://example.com",
				Title: "Example",
			},
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonStop, Raw: "stop"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools:           nil,
		Messages:        []ModelMessage{},
		GenerateID:      mockIDGenerator("id"),
	}), 5*time.Second)

	foundSource := false
	for _, r := range result {
		if r.Type == "source" {
			foundSource = true
			if r.Source == nil {
				t.Error("expected source to not be nil")
			}
		}
	}
	if !foundSource {
		t.Error("expected source part")
	}
}

func TestRunToolsTransformation_ToolCallParsing(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"city": "London"})

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "weather",
			Input:      string(inputJSON),
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonToolCalls, Raw: "tool_calls"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools: ToolSet{
			"weather": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
			},
		},
		Messages:   []ModelMessage{},
		GenerateID: mockIDGenerator("id"),
	}), 5*time.Second)

	foundToolCall := false
	for _, r := range result {
		if r.Type == "tool-call" {
			foundToolCall = true
			if r.ToolCall == nil {
				t.Error("expected tool call to not be nil")
			} else if r.ToolCall.ToolName != "weather" {
				t.Errorf("expected tool name 'weather', got %q", r.ToolCall.ToolName)
			}
		}
	}
	if !foundToolCall {
		t.Error("expected tool-call part")
	}
}

func TestRunToolsTransformation_ToolExecution(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"city": "London"})

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "weather",
			Input:      string(inputJSON),
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonToolCalls, Raw: "tool_calls"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools: ToolSet{
			"weather": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "sunny", nil
				},
			},
		},
		Messages:   []ModelMessage{},
		GenerateID: mockIDGenerator("id"),
	}), 5*time.Second)

	foundToolResult := false
	for _, r := range result {
		if r.Type == "tool-result" {
			foundToolResult = true
			if r.ToolResult == nil {
				t.Error("expected tool result to not be nil")
			} else if r.ToolResult.Output != "sunny" {
				t.Errorf("expected output 'sunny', got %v", r.ToolResult.Output)
			}
		}
	}
	if !foundToolResult {
		t.Error("expected tool-result part")
	}
}

func TestRunToolsTransformation_ToolExecutionError(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"city": "London"})

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "weather",
			Input:      string(inputJSON),
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonToolCalls, Raw: "tool_calls"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools: ToolSet{
			"weather": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return nil, fmt.Errorf("weather service unavailable")
				},
			},
		},
		Messages:   []ModelMessage{},
		GenerateID: mockIDGenerator("id"),
	}), 5*time.Second)

	foundToolError := false
	for _, r := range result {
		if r.Type == "tool-error" {
			foundToolError = true
			if r.ToolError == nil {
				t.Error("expected tool error to not be nil")
			}
		}
	}
	if !foundToolError {
		t.Error("expected tool-error part")
	}
}

func TestRunToolsTransformation_ToolApprovalRequest(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"city": "London"})

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "weather",
			Input:      string(inputJSON),
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonToolCalls, Raw: "tool_calls"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools: ToolSet{
			"weather": Tool{
				InputSchema:   map[string]interface{}{"type": "object"},
				NeedsApproval: true,
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "sunny", nil
				},
			},
		},
		Messages:   []ModelMessage{},
		GenerateID: mockIDGenerator("id"),
	}), 5*time.Second)

	foundApproval := false
	for _, r := range result {
		if r.Type == "tool-approval-request" {
			foundApproval = true
			if r.ToolApprovalRequest == nil {
				t.Error("expected tool approval request to not be nil")
			}
		}
	}
	if !foundApproval {
		t.Error("expected tool-approval-request part")
	}

	// Should NOT have a tool-result (waiting for approval)
	for _, r := range result {
		if r.Type == "tool-result" {
			t.Error("did not expect tool-result when approval is needed")
		}
	}
}

func TestRunToolsTransformation_ProviderExecutedToolResult(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"query": "test"})

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:             "tool-call",
			ToolCallID:       "call-1",
			ToolName:         "web_search",
			Input:            string(inputJSON),
			ProviderExecuted: true,
		},
		{
			Type:       "tool-result",
			ToolCallID: "call-1",
			ToolName:   "web_search",
			Result:     map[string]interface{}{"url": "https://example.com"},
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonStop, Raw: "stop"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools:           ToolSet{},
		Messages:        []ModelMessage{},
		GenerateID:      mockIDGenerator("id"),
	}), 5*time.Second)

	foundToolResult := false
	for _, r := range result {
		if r.Type == "tool-result" {
			foundToolResult = true
			if r.ToolResult == nil {
				t.Error("expected tool result to not be nil")
			} else if !r.ToolResult.ProviderExecuted {
				t.Error("expected ProviderExecuted to be true")
			}
		}
	}
	if !foundToolResult {
		t.Error("expected tool-result part")
	}
}

func TestRunToolsTransformation_FinishEvent(t *testing.T) {
	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{Type: "text-start", ID: "1"},
		{Type: "text-delta", ID: "1", Delta: "hello"},
		{Type: "text-end", ID: "1"},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonStop, Raw: "stop"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools:           nil,
		Messages:        []ModelMessage{},
		GenerateID:      mockIDGenerator("id"),
	}), 5*time.Second)

	foundFinish := false
	for _, r := range result {
		if r.Type == "finish" {
			foundFinish = true
			if r.FinishReason != FinishReasonStop {
				t.Errorf("expected finish reason 'stop', got %q", r.FinishReason)
			}
			if r.Usage.InputTokens == nil || *r.Usage.InputTokens != 3 {
				t.Errorf("expected input tokens 3")
			}
			if r.Usage.OutputTokens == nil || *r.Usage.OutputTokens != 10 {
				t.Errorf("expected output tokens 10")
			}
		}
	}
	if !foundFinish {
		t.Error("expected finish event")
	}
}

func TestRunToolsTransformation_NoToolsDoesNotCrash(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"test": true})

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "unknown",
			Input:      string(inputJSON),
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonStop, Raw: "stop"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	// Should not panic with nil tools
	result := collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools:           nil,
		Messages:        []ModelMessage{},
		GenerateID:      mockIDGenerator("id"),
	}), 5*time.Second)

	if len(result) == 0 {
		t.Error("expected at least some output")
	}
}

func TestRunToolsTransformation_ToolCallbacksInvoked(t *testing.T) {
	inputJSON, _ := json.Marshal(map[string]interface{}{"city": "London"})
	var startCalled, finishCalled bool

	stream := makeGeneratorStream([]LanguageModelV4StreamPart{
		{
			Type:       "tool-call",
			ToolCallID: "call-1",
			ToolName:   "weather",
			Input:      string(inputJSON),
		},
		{
			Type: "finish",
			FinishReason: struct {
				Unified FinishReason
				Raw     string
			}{Unified: FinishReasonToolCalls, Raw: "tool_calls"},
			Usage: struct {
				InputTokens  TokenCount
				OutputTokens TokenCount
			}{InputTokens: testUsage.InputTokens, OutputTokens: testUsage.OutputTokens},
		},
	})

	collectParts(RunToolsTransformation(RunToolsTransformationOptions{
		GeneratorStream: stream,
		Tools: ToolSet{
			"weather": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "sunny", nil
				},
			},
		},
		Messages:   []ModelMessage{},
		GenerateID: mockIDGenerator("id"),
		OnToolCallStart: []func(event OnToolCallStartEvent){
			func(event OnToolCallStartEvent) {
				startCalled = true
			},
		},
		OnToolCallFinish: []func(event OnToolCallFinishEvent){
			func(event OnToolCallFinishEvent) {
				finishCalled = true
			},
		},
	}), 5*time.Second)

	if !startCalled {
		t.Error("expected onToolCallStart to be called")
	}
	if !finishCalled {
		t.Error("expected onToolCallFinish to be called")
	}
}
