// Ported from: packages/ai/src/generate-text/execute-tool-call.test.ts
package generatetext

import (
	"errors"
	"testing"
)

func createTestToolCall(overrides ...func(*ToolCall)) ToolCall {
	tc := ToolCall{
		Type:       "tool-call",
		ToolCallID: "call-1",
		ToolName:   "testTool",
		Input:      map[string]interface{}{"value": "test"},
		Dynamic:    false,
	}
	for _, o := range overrides {
		o(&tc)
	}
	return tc
}

func TestExecuteToolCall_NoExecuteFunction(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				// No Execute function
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestExecuteToolCall_Success(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					inputMap := input.(map[string]interface{})
					return inputMap["value"].(string) + "-result", nil
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, ok := result.(*ToolResult)
	if !ok {
		t.Fatalf("expected *ToolResult, got %T", result)
	}
	if tr.Type != "tool-result" {
		t.Errorf("expected type 'tool-result', got %q", tr.Type)
	}
	if tr.ToolCallID != "call-1" {
		t.Errorf("expected toolCallId 'call-1', got %q", tr.ToolCallID)
	}
	if tr.ToolName != "testTool" {
		t.Errorf("expected toolName 'testTool', got %q", tr.ToolName)
	}
	if tr.Output != "test-result" {
		t.Errorf("expected output 'test-result', got %v", tr.Output)
	}
}

func TestExecuteToolCall_PreservesProviderMetadata_OnSuccess(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(func(tc *ToolCall) {
			tc.ProviderMetadata = ProviderMetadata{"custom": {"key": "value"}}
		}),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					inputMap := input.(map[string]interface{})
					return inputMap["value"].(string) + "-result", nil
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, ok := result.(*ToolResult)
	if !ok {
		t.Fatalf("expected *ToolResult, got %T", result)
	}
	if tr.ProviderMetadata == nil {
		t.Fatal("expected providerMetadata to not be nil")
	}
	if tr.ProviderMetadata["custom"]["key"] != "value" {
		t.Errorf("expected custom.key 'value', got %v", tr.ProviderMetadata["custom"]["key"])
	}
}

func TestExecuteToolCall_Error(t *testing.T) {
	toolError := errors.New("execution failed")

	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return nil, toolError
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	te, ok := result.(*ToolError)
	if !ok {
		t.Fatalf("expected *ToolError, got %T", result)
	}
	if te.Type != "tool-error" {
		t.Errorf("expected type 'tool-error', got %q", te.Type)
	}
	if te.ToolCallID != "call-1" {
		t.Errorf("expected toolCallId 'call-1', got %q", te.ToolCallID)
	}
	if te.Error != toolError {
		t.Errorf("expected toolError, got %v", te.Error)
	}
}

func TestExecuteToolCall_PreservesProviderMetadata_OnError(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(func(tc *ToolCall) {
			tc.ProviderMetadata = ProviderMetadata{"custom": {"key": "value"}}
		}),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return nil, errors.New("execution failed")
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	te, ok := result.(*ToolError)
	if !ok {
		t.Fatalf("expected *ToolError, got %T", result)
	}
	if te.ProviderMetadata == nil {
		t.Fatal("expected providerMetadata to not be nil")
	}
	if te.ProviderMetadata["custom"]["key"] != "value" {
		t.Errorf("expected custom.key 'value', got %v", te.ProviderMetadata["custom"]["key"])
	}
}

func TestExecuteToolCall_OnToolCallStart_CalledBeforeExecution(t *testing.T) {
	var order []string
	var startEvents []OnToolCallStartEvent

	stepNum := 2
	model := &ModelInfo{Provider: "test-provider", ModelID: "test-model"}

	_, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					order = append(order, "execute")
					inputMap := input.(map[string]interface{})
					return inputMap["value"].(string) + "-result", nil
				},
			},
		},
		Messages:    []ModelMessage{{Role: "user", Content: "test message"}},
		StepNumber:  &stepNum,
		Model:       model,
		OnToolCallStart: []func(event OnToolCallStartEvent){
			func(event OnToolCallStartEvent) {
				order = append(order, "onToolCallStart")
				startEvents = append(startEvents, event)
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(startEvents) != 1 {
		t.Fatalf("expected 1 start event, got %d", len(startEvents))
	}
	if *startEvents[0].StepNumber != 2 {
		t.Errorf("expected step number 2, got %v", startEvents[0].StepNumber)
	}
	if len(order) != 2 || order[0] != "onToolCallStart" || order[1] != "execute" {
		t.Errorf("expected [onToolCallStart, execute], got %v", order)
	}
}

func TestExecuteToolCall_OnToolCallFinish_Success(t *testing.T) {
	var finishEvents []OnToolCallFinishEvent

	stepNum := 3
	model := &ModelInfo{Provider: "test-provider", ModelID: "test-model"}

	_, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					inputMap := input.(map[string]interface{})
					return inputMap["value"].(string) + "-result", nil
				},
			},
		},
		Messages:   []ModelMessage{{Role: "user", Content: "test message"}},
		StepNumber: &stepNum,
		Model:      model,
		OnToolCallFinish: []func(event OnToolCallFinishEvent){
			func(event OnToolCallFinishEvent) {
				finishEvents = append(finishEvents, event)
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(finishEvents) != 1 {
		t.Fatalf("expected 1 finish event, got %d", len(finishEvents))
	}
	if !finishEvents[0].Success {
		t.Error("expected success to be true")
	}
	if finishEvents[0].Output != "test-result" {
		t.Errorf("expected output 'test-result', got %v", finishEvents[0].Output)
	}
	if finishEvents[0].DurationMs < 0 {
		t.Errorf("expected non-negative durationMs, got %f", finishEvents[0].DurationMs)
	}
}

func TestExecuteToolCall_OnToolCallFinish_Error(t *testing.T) {
	var finishEvents []OnToolCallFinishEvent
	toolError := errors.New("execution failed")

	_, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return nil, toolError
				},
			},
		},
		Messages: []ModelMessage{},
		OnToolCallFinish: []func(event OnToolCallFinishEvent){
			func(event OnToolCallFinishEvent) {
				finishEvents = append(finishEvents, event)
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(finishEvents) != 1 {
		t.Fatalf("expected 1 finish event, got %d", len(finishEvents))
	}
	if finishEvents[0].Success {
		t.Error("expected success to be false")
	}
	if finishEvents[0].Error != toolError {
		t.Errorf("expected error to be the tool error")
	}
}

func TestExecuteToolCall_ToolsNil(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools:    nil,
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestExecuteToolCall_ToolNotFound(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(func(tc *ToolCall) {
			tc.ToolName = "nonexistent"
		}),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "result", nil
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestExecuteToolCall_MultipleCallbacks(t *testing.T) {
	var calls []string

	_, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					inputMap := input.(map[string]interface{})
					return inputMap["value"].(string) + "-result", nil
				},
			},
		},
		Messages: []ModelMessage{},
		OnToolCallStart: []func(event OnToolCallStartEvent){
			func(event OnToolCallStartEvent) { calls = append(calls, "first-start") },
			func(event OnToolCallStartEvent) { calls = append(calls, "second-start") },
		},
		OnToolCallFinish: []func(event OnToolCallFinishEvent){
			func(event OnToolCallFinishEvent) { calls = append(calls, "first-finish") },
			func(event OnToolCallFinishEvent) { calls = append(calls, "second-finish") },
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"first-start", "second-start", "first-finish", "second-finish"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}
	for i, exp := range expected {
		if calls[i] != exp {
			t.Errorf("expected calls[%d] = %q, got %q", i, exp, calls[i])
		}
	}
}

func TestExecuteToolCall_DynamicTool_Success(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(func(tc *ToolCall) {
			tc.Dynamic = true
		}),
		Tools: ToolSet{
			"testTool": Tool{
				Type:        "dynamic",
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return "dynamic-result", nil
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, ok := result.(*ToolResult)
	if !ok {
		t.Fatalf("expected *ToolResult, got %T", result)
	}
	if !tr.Dynamic {
		t.Error("expected dynamic to be true")
	}
}

func TestExecuteToolCall_DynamicTool_Error(t *testing.T) {
	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(func(tc *ToolCall) {
			tc.Dynamic = true
		}),
		Tools: ToolSet{
			"testTool": Tool{
				Type:        "dynamic",
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					return nil, errors.New("error")
				},
			},
		},
		Messages: []ModelMessage{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	te, ok := result.(*ToolError)
	if !ok {
		t.Fatalf("expected *ToolError, got %T", result)
	}
	if !te.Dynamic {
		t.Error("expected dynamic to be true")
	}
}

func TestExecuteToolCall_NilCallbacksInSlice(t *testing.T) {
	var calls []string

	result, err := ExecuteToolCall(ExecuteToolCallOptions{
		ToolCall: createTestToolCall(),
		Tools: ToolSet{
			"testTool": Tool{
				InputSchema: map[string]interface{}{"type": "object"},
				Execute: func(input interface{}, opts ToolExecuteOptions) (interface{}, error) {
					inputMap := input.(map[string]interface{})
					return inputMap["value"].(string) + "-result", nil
				},
			},
		},
		Messages: []ModelMessage{},
		OnToolCallStart: []func(event OnToolCallStartEvent){
			nil,
			func(event OnToolCallStartEvent) { calls = append(calls, "start") },
			nil,
		},
		OnToolCallFinish: []func(event OnToolCallFinishEvent){
			nil,
			func(event OnToolCallFinishEvent) { calls = append(calls, "finish") },
			nil,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, ok := result.(*ToolResult)
	if !ok {
		t.Fatalf("expected *ToolResult, got %T", result)
	}
	if tr.Output != "test-result" {
		t.Errorf("expected 'test-result', got %v", tr.Output)
	}
	if len(calls) != 2 || calls[0] != "start" || calls[1] != "finish" {
		t.Errorf("expected [start, finish], got %v", calls)
	}
}
