// Ported from: packages/core/src/loop/test-utils/tools.ts
package testutils

// ---------------------------------------------------------------------------
// Stub types for unported packages (tools-specific)
// ---------------------------------------------------------------------------

// DynamicTool is a stub for @internal/ai-sdk-v5.dynamicTool.
// TODO: import from tools package once ported.
type DynamicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Execute     func(args map[string]any) (any, error)
}

// StepCountIs is a stub for @internal/ai-sdk-v5.stepCountIs.
// TODO: import from tools package once ported.
// StepCountIs returns a StopCondition that stops after n steps.
func StepCountIs(n int) func(args map[string]any) bool {
	return func(args map[string]any) bool {
		steps, ok := args["steps"].([]any)
		if !ok {
			return false
		}
		return len(steps) >= n
	}
}

// ---------------------------------------------------------------------------
// ToolsTestsConfig
// ---------------------------------------------------------------------------

// ToolsTestsConfig configures the toolsTests test suite.
type ToolsTestsConfig struct {
	LoopFn LoopFn
	RunID  string
}

// ToolsTests contains the test definitions for tool call scenarios.
// In the TS source, this is a vitest describe block that validates
// provider-executed tools, dynamic tools, tool errors, stop conditions,
// multi-step tool calls, and tool call streaming.
type ToolsTests struct {
	Config ToolsTestsConfig
}

// NewToolsTests creates a new ToolsTests instance.
func NewToolsTests(config ToolsTestsConfig) *ToolsTests {
	return &ToolsTests{Config: config}
}

// ---------------------------------------------------------------------------
// Pre-built tool test helpers
// ---------------------------------------------------------------------------

// CreateToolCallStream creates a stream with a single tool call and finish.
func CreateToolCallStream(toolCallID, toolName, input string) <-chan LanguageModelV2StreamPart {
	return ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
		{
			"type":       "tool-call",
			"toolCallId": toolCallID,
			"toolName":   toolName,
			"input":      input,
		},
		{
			"type":         "finish",
			"finishReason": "stop",
			"usage":        TestUsage,
		},
	})
}

// CreateProviderExecutedToolCallStream creates a stream with provider-executed
// tool call and result parts.
func CreateProviderExecutedToolCallStream(toolCallID, toolName, input, result string) <-chan LanguageModelV2StreamPart {
	return ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
		{
			"type":             "tool-input-start",
			"id":               toolCallID,
			"toolName":         toolName,
			"providerExecuted": true,
		},
		{
			"type":  "tool-input-delta",
			"id":    toolCallID,
			"delta": input,
		},
		{
			"type": "tool-input-end",
			"id":   toolCallID,
		},
		{
			"type":             "tool-call",
			"toolCallId":       toolCallID,
			"toolName":         toolName,
			"input":            input,
			"providerExecuted": true,
		},
		{
			"type":             "tool-result",
			"toolCallId":       toolCallID,
			"toolName":         toolName,
			"result":           result,
			"providerExecuted": true,
		},
		{
			"type":         "finish",
			"finishReason": "stop",
			"usage":        TestUsage,
		},
	})
}

// CreateMultiStepToolCallModel creates a mock V2 model that produces a
// multi-step tool call followed by text, used for testing agentic loops.
func CreateMultiStepToolCallModel(toolCallID, toolName, input string) *MastraLanguageModelV2Mock {
	callCount := 0
	return NewMastraLanguageModelV2Mock(MastraLanguageModelV2MockConfig{
		DoStream: func(options map[string]any) (*DoStreamResult, error) {
			callCount++
			if callCount == 1 {
				stream := CreateToolCallStream(toolCallID, toolName, input)
				return &DoStreamResult{Stream: stream}, nil
			}
			// Second call returns text
			stream := ConvertArrayToReadableStream([]LanguageModelV2StreamPart{
				{"type": "text-start", "id": "text-1"},
				{"type": "text-delta", "id": "text-1", "delta": "Final answer"},
				{"type": "text-end", "id": "text-1"},
				{
					"type":         "finish",
					"finishReason": "stop",
					"usage":        TestUsage,
				},
			})
			return &DoStreamResult{Stream: stream}, nil
		},
	})
}
