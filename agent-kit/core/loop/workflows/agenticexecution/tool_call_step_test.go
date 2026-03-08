// Ported from: packages/core/src/loop/workflows/agentic-execution/tool-call-step.test.ts
package agenticexecution

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock types for tool call step tests
// ---------------------------------------------------------------------------

// mockMessageListForTCS implements MessageListFull for tool call step tests.
type mockMessageListForTCS struct {
	addCalls       []mockAddCall
	inputMsgs      []any
	responseMsgs   []map[string]any
	allMsgs        []map[string]any
	systemMsgs     []any
}

type mockAddCall struct {
	msg    any
	source string
}

func (m *mockMessageListForTCS) GetAllSystemMessages() []any       { return m.systemMsgs }
func (m *mockMessageListForTCS) ReplaceAllSystemMessages(msgs []any) { m.systemMsgs = msgs }
func (m *mockMessageListForTCS) AddSystem(content string, tag string) {}
func (m *mockMessageListForTCS) Add(msg any, source string) {
	m.addCalls = append(m.addCalls, mockAddCall{msg: msg, source: source})
}
func (m *mockMessageListForTCS) RemoveByIds(ids []string) {}
func (m *mockMessageListForTCS) GetAll() MessageListView  { return &mockMLView{db: m.allMsgs, model: m.inputMsgs} }
func (m *mockMessageListForTCS) GetInput() MessageListView {
	return &mockMLView{db: nil, model: m.inputMsgs}
}
func (m *mockMessageListForTCS) GetResponse() MessageListView {
	return &mockMLView{db: m.responseMsgs, model: nil}
}

// mockMLView implements MessageListView.
type mockMLView struct {
	db    []map[string]any
	model []any
}

func (v *mockMLView) DB() []map[string]any                        { return v.db }
func (v *mockMLView) AIV5Model() []any                            { return v.model }
func (v *mockMLView) AIV5ModelContent(stepNumber int) []any       { return nil }
func (v *mockMLView) AIV5LLMPrompt(args map[string]any) ([]any, error) { return nil, nil }

// ---------------------------------------------------------------------------
// Tests: tool execution error handling
// ---------------------------------------------------------------------------

func TestCreateToolCallStep_ErrorHandling(t *testing.T) {
	t.Run("should return error field when tool execute throws", func(t *testing.T) {
		// Arrange: tool with execute that returns an error
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		tools := ToolSet{
			"failing-tool": map[string]any{
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					return nil, errors.New("External API error: 503 Service Unavailable")
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:       tools,
			MessageList: ml,
			Controller:  controller,
			RunID:       "test-run",
		})

		input := ToolCallInput{
			ToolCallID: "test-call-id",
			ToolName:   "failing-tool",
			Args:       map[string]any{"param": "test"},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("Execute returned unexpected error: %v", err)
		}

		// Assert: result should have Error field, not Result
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		if tc.Error == nil {
			t.Fatal("expected Error to be set")
		}
		if tc.Result != nil {
			t.Fatal("expected Result to be nil when Error is set")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: tool approval workflow
// ---------------------------------------------------------------------------

func TestCreateToolCallStep_ApprovalWorkflow(t *testing.T) {
	t.Run("should emit approval chunk and return without result when approval required", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		executeCalled := false
		tools := ToolSet{
			"test-tool": map[string]any{
				"requireApproval": true,
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					executeCalled = true
					return map[string]any{"success": true}, nil
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:               tools,
			MessageList:         ml,
			Controller:          controller,
			RunID:               "test-run",
			RequireToolApproval: true,
		})

		input := ToolCallInput{
			ToolCallID: "test-call-id",
			ToolName:   "test-tool",
			Args:       map[string]any{"param": "test"},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: approval chunk should be emitted
		select {
		case chunk := <-controller:
			chunkType, _ := chunk["type"].(string)
			if chunkType != "tool-call-approval" {
				t.Errorf("expected chunk type 'tool-call-approval', got %q", chunkType)
			}
			payload, _ := chunk["payload"].(map[string]any)
			if payload["toolCallId"] != "test-call-id" {
				t.Errorf("expected toolCallId 'test-call-id', got %v", payload["toolCallId"])
			}
		default:
			t.Error("expected approval chunk to be emitted")
		}

		// Tool should NOT be executed
		if executeCalled {
			t.Error("tool should not have been executed when approval is required")
		}

		// Result should have no result (pending approval)
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		if tc.Result != nil {
			t.Error("expected Result to be nil for pending approval")
		}
		if tc.Error != nil {
			t.Error("expected Error to be nil for pending approval")
		}
	})

	t.Run("should handle declined tool calls without executing", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{
			inputMsgs:    []any{},
			responseMsgs: []map[string]any{},
			allMsgs:      []map[string]any{},
		}
		controller := make(chan map[string]any, 100)

		executeCalled := false
		tools := ToolSet{
			"test-tool": map[string]any{
				"requireApproval": true,
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					executeCalled = true
					return nil, nil
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:               tools,
			MessageList:         ml,
			Controller:          controller,
			RunID:               "test-run",
			RequireToolApproval: true,
		})

		// Simulate resumeData with approved=false via args
		input := ToolCallInput{
			ToolCallID: "test-call-id",
			ToolName:   "test-tool",
			Args:       map[string]any{"param": "test", "resumeData": map[string]any{"approved": false}},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		if tc.Result != "Tool call was not approved by the user" {
			t.Errorf("expected declined message, got %v", tc.Result)
		}
		if executeCalled {
			t.Error("tool should not be executed when declined")
		}
	})

	t.Run("should return fallback result for provider-executed tools without output", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		providerExec := true
		input := ToolCallInput{
			ToolCallID:       "test-call-id",
			ToolName:         "web_search_20250305",
			Args:             map[string]any{"query": "test"},
			ProviderExecuted: &providerExec,
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:       ToolSet{},
			MessageList: ml,
			Controller:  controller,
			RunID:       "test-run",
		})

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: fallback result should be returned
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		resultMap, ok := tc.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", tc.Result)
		}
		if resultMap["providerExecuted"] != true {
			t.Error("expected providerExecuted=true in fallback result")
		}
		if resultMap["toolName"] != "web_search_20250305" {
			t.Errorf("expected toolName 'web_search_20250305', got %v", resultMap["toolName"])
		}
	})

	t.Run("should pass through output for provider-executed tools when output present", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		providerExec := true
		input := ToolCallInput{
			ToolCallID:       "test-call-id",
			ToolName:         "web_search_20250305",
			Args:             map[string]any{"query": "test"},
			ProviderExecuted: &providerExec,
			Output:           map[string]any{"searchResults": []string{"result1"}},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:       ToolSet{},
			MessageList: ml,
			Controller:  controller,
			RunID:       "test-run",
		})

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: actual output should be used
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		resultMap, ok := tc.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", tc.Result)
		}
		if resultMap["searchResults"] == nil {
			t.Error("expected searchResults in output")
		}
	})

	t.Run("should execute tool and return result when approval granted", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{
			inputMsgs:    []any{},
			responseMsgs: []map[string]any{},
			allMsgs:      []map[string]any{},
		}
		controller := make(chan map[string]any, 100)

		tools := ToolSet{
			"test-tool": map[string]any{
				"requireApproval": true,
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					return map[string]any{"success": true, "data": "test-result"}, nil
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:               tools,
			MessageList:         ml,
			Controller:          controller,
			RunID:               "test-run",
			RequireToolApproval: true,
		})

		// Simulate approved resumeData via args
		input := ToolCallInput{
			ToolCallID: "test-call-id",
			ToolName:   "test-tool",
			Args:       map[string]any{"param": "test", "resumeData": map[string]any{"approved": true}},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool was executed and returned result
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		resultMap, ok := tc.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", tc.Result)
		}
		if resultMap["success"] != true {
			t.Error("expected success=true in result")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: requestContext forwarding
// ---------------------------------------------------------------------------

func TestCreateToolCallStep_RequestContextForwarding(t *testing.T) {
	t.Run("should forward requestContext to tool execute in toolOptions", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		// Use a map as requestContext since the Go implementation
		// passes params.RequestContext directly to toolOptions.
		requestContext := map[string]any{
			"testKey":   "testValue",
			"apiClient": map[string]any{"fetch": "mocked"},
		}

		var capturedOpts MastraToolInvocationOptions
		tools := ToolSet{
			"ctx-tool": map[string]any{
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					capturedOpts = opts
					return map[string]any{"ok": true}, nil
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:          tools,
			MessageList:    ml,
			Controller:     controller,
			RunID:          "ctx-run",
			RequestContext: requestContext,
		})

		input := ToolCallInput{
			ToolCallID: "ctx-call-id",
			ToolName:   "ctx-tool",
			Args:       map[string]any{"key": "value"},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: requestContext was forwarded
		if capturedOpts.RequestContext == nil {
			t.Fatal("expected RequestContext to be set")
		}
		rc, ok := capturedOpts.RequestContext.(map[string]any)
		if !ok {
			t.Fatalf("expected map requestContext, got %T", capturedOpts.RequestContext)
		}
		if rc["testKey"] != "testValue" {
			t.Errorf("expected testKey=testValue, got %v", rc["testKey"])
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: tool not found
// ---------------------------------------------------------------------------

func TestCreateToolCallStep_ToolNotFound(t *testing.T) {
	t.Run("should return ToolNotFoundError when tool is not in toolset", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		tools := ToolSet{
			"existingTool": map[string]any{
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					return map[string]any{"ok": true}, nil
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:       tools,
			MessageList: ml,
			Controller:  controller,
			RunID:       "test-run",
		})

		input := ToolCallInput{
			ToolCallID: "call-1",
			ToolName:   "nonExistentTool",
			Args:       map[string]any{"param": "test"},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		if tc.Error == nil {
			t.Fatal("expected Error to be set for unknown tool")
		}
		toolErr, ok := tc.Error.(*ToolNotFoundError)
		if !ok {
			t.Fatalf("expected *ToolNotFoundError, got %T", tc.Error)
		}
		if !strings.Contains(toolErr.Message, "nonExistentTool") {
			t.Errorf("error message should mention tool name, got: %s", toolErr.Message)
		}
		if !strings.Contains(toolErr.Message, "existingTool") {
			t.Errorf("error message should list available tools, got: %s", toolErr.Message)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: malformed JSON args (issue #9815)
// ---------------------------------------------------------------------------

func TestCreateToolCallStep_MalformedArgs(t *testing.T) {
	t.Run("should return error when args are nil (malformed JSON from model)", func(t *testing.T) {
		// Arrange
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		controller := make(chan map[string]any, 100)

		executeCalled := false
		tools := ToolSet{
			"test-tool": map[string]any{
				"execute": func(args map[string]any, opts MastraToolInvocationOptions) (any, error) {
					executeCalled = true
					return map[string]any{"success": true}, nil
				},
			},
		}

		step := CreateToolCallStep(OuterLLMRun{
			Tools:       tools,
			MessageList: ml,
			Controller:  controller,
			RunID:       "test-run",
		})

		input := ToolCallInput{
			ToolCallID: "call-1",
			ToolName:   "test-tool",
			Args:       nil, // Simulates malformed JSON from model
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: input})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool should NOT be executed
		if executeCalled {
			t.Error("tool should not be executed when args are nil")
		}

		// Should return an error
		tc, ok := result.(ToolCallOutput)
		if !ok {
			t.Fatalf("expected ToolCallOutput, got %T", result)
		}
		if tc.Error == nil {
			t.Fatal("expected error for nil args")
		}
		errMsg := fmt.Sprintf("%v", tc.Error)
		if !strings.Contains(strings.ToLower(errMsg), "invalid") && !strings.Contains(strings.ToLower(errMsg), "json") {
			t.Errorf("error message should mention invalid/JSON, got: %s", errMsg)
		}
	})
}
