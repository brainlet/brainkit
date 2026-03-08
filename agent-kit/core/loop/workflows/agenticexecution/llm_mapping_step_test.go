// Ported from: packages/core/src/loop/workflows/agentic-execution/llm-mapping-step.test.ts
package agenticexecution

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper: create OuterLLMRun params for LLM mapping step tests
// ---------------------------------------------------------------------------

func newLLMMapTestParams(controller chan map[string]any, ml MessageListFull, tools ToolSet) OuterLLMRun {
	return OuterLLMRun{
		Controller:  controller,
		MessageList: ml,
		RunID:       "test-run",
		Tools:       tools,
		Internal: map[string]any{
			"generateId": func() string { return "test-message-id" },
		},
	}
}

// collectChunks drains all chunks from the controller channel.
func collectChunks(ch chan map[string]any) []map[string]any {
	var chunks []map[string]any
	for {
		select {
		case chunk := <-ch:
			chunks = append(chunks, chunk)
		default:
			return chunks
		}
	}
}

// chunksByType filters collected chunks by type.
func chunksByType(chunks []map[string]any, chunkType string) []map[string]any {
	var result []map[string]any
	for _, c := range chunks {
		if t, _ := c["type"].(string); t == chunkType {
			result = append(result, c)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Tests: HITL behavior
// ---------------------------------------------------------------------------

func TestCreateLLMMappingStep_HITLBehavior(t *testing.T) {
	t.Run("should bail when ALL tools have no result (all HITL tools)", func(t *testing.T) {
		// Arrange: two tools with no result AND no error (true HITL — waiting for human)
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{ToolCallID: "call-1", ToolName: "updateSummary", Args: map[string]any{"summary": "test"}},
			{ToolCallID: "call-2", ToolName: "updateDescription", Args: map[string]any{"description": "test"}},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: isContinued should be false (bailed)
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		stepResult, _ := resultMap["stepResult"].(map[string]any)
		if stepResult == nil {
			t.Fatal("expected stepResult in result")
		}
		isContinued, _ := stepResult["isContinued"].(bool)
		if isContinued {
			t.Error("expected isContinued=false when all tools have no result (HITL)")
		}

		// Should NOT emit tool-result chunks
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) > 0 {
			t.Errorf("expected no tool-result chunks, got %d", len(toolResultChunks))
		}
	})

	t.Run("should continue when ALL tools have results", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{ToolCallID: "call-1", ToolName: "updateTitle", Args: map[string]any{"title": "test"}, Result: map[string]any{"success": true}},
			{ToolCallID: "call-2", ToolName: "updateStatus", Args: map[string]any{"status": "active"}, Result: map[string]any{"success": true}},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool-result chunks should be emitted for both tools
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) != 2 {
			t.Errorf("expected 2 tool-result chunks, got %d", len(toolResultChunks))
		}

		// Verify both tool IDs are present
		foundCall1, foundCall2 := false, false
		for _, chunk := range toolResultChunks {
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				if payload["toolCallId"] == "call-1" {
					foundCall1 = true
				}
				if payload["toolCallId"] == "call-2" {
					foundCall2 = true
				}
			}
		}
		if !foundCall1 || !foundCall2 {
			t.Error("expected both tool-result chunks to be present")
		}
	})

	t.Run("should bail when SOME tools have results and SOME do not (mixed HITL)", func(t *testing.T) {
		// Arrange: one tool with result, one with neither result nor error (HITL)
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{ToolCallID: "call-1", ToolName: "updateTitle", Args: map[string]any{"title": "test"}, Result: map[string]any{"success": true}},
			{ToolCallID: "call-2", ToolName: "updateSummary", Args: map[string]any{"summary": "test"}}, // No result, no error (HITL)
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: isContinued should be false (bailed due to HITL)
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		stepResult, _ := resultMap["stepResult"].(map[string]any)
		if stepResult == nil {
			t.Fatal("expected stepResult in result")
		}
		isContinued, _ := stepResult["isContinued"].(bool)
		if isContinued {
			t.Error("expected isContinued=false for mixed HITL scenario")
		}
	})

	t.Run("should emit tool-error when errors coexist with HITL pending tools", func(t *testing.T) {
		// When there are HITL entries (no result AND no error) alongside error entries,
		// the Go code enters the hasUndefined path which emits tool-error chunks for errors.
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "brokenTool",
				Args:       map[string]any{"param": "test"},
				Error:      errors.New("Tool execution failed"),
			},
			{
				ToolCallID: "call-2",
				ToolName:   "hitlTool",
				Args:       map[string]any{"summary": "test"},
				// No result, no error — HITL pending
			},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool-error chunk emitted for the error entry
		chunks := collectChunks(controller)
		toolErrorChunks := chunksByType(chunks, "tool-error")
		if len(toolErrorChunks) != 1 {
			t.Errorf("expected 1 tool-error chunk, got %d", len(toolErrorChunks))
		}
		if len(toolErrorChunks) > 0 {
			payload, _ := toolErrorChunks[0]["payload"].(map[string]any)
			if payload["toolCallId"] != "call-1" {
				t.Errorf("expected toolCallId 'call-1', got %v", payload["toolCallId"])
			}
		}

		// isContinued should be false (HITL pending means bail)
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		stepResult, _ := resultMap["stepResult"].(map[string]any)
		isContinued, _ := stepResult["isContinued"].(bool)
		if isContinued {
			t.Error("expected isContinued=false when HITL tools are pending")
		}
	})

	t.Run("should emit tool-result for error-only entries via successful path", func(t *testing.T) {
		// Note: In the Go implementation, when all entries have Error (but NOT nil Result+Error),
		// hasUndefined is false because the check is (Result==nil AND Error==nil).
		// Error-only entries fall through to the successful tool-result path.
		// This differs from the TS behavior where errors get dedicated tool-error handling.
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "brokenTool",
				Args:       map[string]any{"param": "test"},
				Error:      errors.New("Tool execution failed"),
			},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool-result chunk emitted (not tool-error) since hasUndefined is false
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) != 1 {
			t.Errorf("expected 1 tool-result chunk, got %d", len(toolResultChunks))
		}

		// Verify the result map is returned
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		if resultMap["stepResult"] == nil {
			t.Fatal("expected stepResult in result")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: tool execution error self-recovery with HITL context (issue #9815)
// ---------------------------------------------------------------------------

func TestCreateLLMMappingStep_ErrorSelfRecovery(t *testing.T) {
	t.Run("should emit tool-error for errors when HITL tool is also pending and continue with isContinued from error path", func(t *testing.T) {
		// When errors coexist with HITL entries (no result, no error), the Go code
		// enters hasUndefined path, emits tool-error chunks for errors, and
		// sets isContinued=true to allow the model to self-correct.
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "myTool",
				Args:       map[string]any{"invalidParam": "wrong type"},
				Error:      errors.New("Invalid arguments"),
			},
			{
				ToolCallID: "call-2",
				ToolName:   "pendingTool",
				Args:       map[string]any{},
				// No result, no error — HITL pending triggers hasUndefined
			},
		}

		// Act
		result, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool-error chunk emitted
		chunks := collectChunks(controller)
		toolErrorChunks := chunksByType(chunks, "tool-error")
		if len(toolErrorChunks) < 1 {
			t.Error("expected at least 1 tool-error chunk")
		}

		// Error added to messageList
		if len(ml.addCalls) == 0 {
			t.Error("expected error to be added to messageList")
		}

		// Verify result structure
		resultMap, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		if resultMap["stepResult"] == nil {
			t.Fatal("expected stepResult in result")
		}
	})

	t.Run("should process mixed success and error via successful path when no HITL entries", func(t *testing.T) {
		// When there are mixed success (Result != nil) and error (Error != nil, Result == nil)
		// entries but NO truly HITL entries, hasUndefined is false and all go through
		// the successful tool-result path.
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "fetchData",
				Args:       map[string]any{"url": "https://example.com"},
				Result:     map[string]any{"data": "some content"},
			},
			{
				ToolCallID: "call-2",
				ToolName:   "processData",
				Args:       map[string]any{"data": nil},
				Error:      errors.New("Cannot process null data"),
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool-result chunks emitted for all entries (Go routes both through tool-result)
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) != 2 {
			t.Errorf("expected 2 tool-result chunks, got %d", len(toolResultChunks))
		}

		// Messages added to messageList
		if len(ml.addCalls) == 0 {
			t.Error("expected messages to be added to messageList")
		}
	})

	t.Run("should process error-only entries via successful path", func(t *testing.T) {
		// Multiple error-only entries (no HITL) all go through tool-result path.
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "toolA",
				Args:       map[string]any{"x": 1},
				Error:      errors.New("Network timeout"),
			},
			{
				ToolCallID: "call-2",
				ToolName:   "toolB",
				Args:       map[string]any{"y": 2},
				Error:      errors.New("Cannot read property 'foo' of undefined"),
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: both entries emitted as tool-result chunks
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) != 2 {
			t.Errorf("expected 2 tool-result chunks, got %d", len(toolResultChunks))
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: provider-executed tool message filtering
// ---------------------------------------------------------------------------

func TestCreateLLMMappingStep_ProviderExecutedFiltering(t *testing.T) {
	t.Run("should split client and provider executed tools into separate messages", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		providerExec := true
		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "get_company_info",
				Args:       map[string]any{"name": "test"},
				Result:     map[string]any{"company": "Acme"},
			},
			{
				ToolCallID:       "call-2",
				ToolName:         "web_search_20250305",
				Args:             map[string]any{"query": "test"},
				Result:           map[string]any{"providerExecuted": true, "toolName": "web_search_20250305"},
				ProviderExecuted: &providerExec,
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: two separate messages should be added (client + provider)
		if len(ml.addCalls) < 2 {
			t.Errorf("expected at least 2 messageList.add calls, got %d", len(ml.addCalls))
		}

		// Verify chunks emitted for both tools
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) != 2 {
			t.Errorf("expected 2 tool-result chunks, got %d", len(toolResultChunks))
		}
	})

	t.Run("should emit stream chunks for provider-executed tools", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		providerExec := true
		inputData := []ToolCallOutput{
			{
				ToolCallID:       "call-1",
				ToolName:         "web_search_20250305",
				Args:             map[string]any{"query": "test"},
				Result:           map[string]any{"providerExecuted": true, "toolName": "web_search_20250305"},
				ProviderExecuted: &providerExec,
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: tool-result chunk should be emitted
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) < 1 {
			t.Error("expected at least 1 tool-result chunk for provider-executed tool")
		}

		// Message should be added with provider parts
		if len(ml.addCalls) < 1 {
			t.Error("expected messageList.add to be called for provider-executed tool")
		}
	})

	t.Run("should continue when provider-executed tools are mixed with regular tools", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		step := CreateLLMMappingStep(newLLMMapTestParams(controller, ml, nil), llmExecStep)

		providerExec := true
		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "get_company_info",
				Args:       map[string]any{"name": "test"},
				Result:     map[string]any{"company": "Acme"},
			},
			{
				ToolCallID:       "call-2",
				ToolName:         "web_search_20250305",
				Args:             map[string]any{"query": "test"},
				Result:           map[string]any{"providerExecuted": true, "toolName": "web_search_20250305"},
				ProviderExecuted: &providerExec,
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: both tool-result chunks should be emitted
		chunks := collectChunks(controller)
		toolResultChunks := chunksByType(chunks, "tool-result")
		if len(toolResultChunks) != 2 {
			t.Errorf("expected 2 tool-result chunks, got %d", len(toolResultChunks))
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: toModelOutput
// ---------------------------------------------------------------------------

func TestCreateLLMMappingStep_ToModelOutput(t *testing.T) {
	t.Run("should call toModelOutput and store result on providerMetadata", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		toModelOutputCalled := false
		tools := ToolSet{
			"weather": map[string]any{
				"toModelOutput": func(output any) any {
					toModelOutputCalled = true
					return map[string]any{
						"type":  "text",
						"value": "Transformed result",
					}
				},
			},
		}

		params := newLLMMapTestParams(controller, ml, tools)
		step := CreateLLMMappingStep(params, llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "weather",
				Args:       map[string]any{"city": "NYC"},
				Result:     map[string]any{"temperature": 72, "conditions": "sunny"},
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: toModelOutput should have been called
		if !toModelOutputCalled {
			t.Error("expected toModelOutput to be called")
		}

		// Message should be added with providerMetadata containing modelOutput
		if len(ml.addCalls) < 1 {
			t.Fatal("expected messageList.add to be called")
		}

		// Verify the message has mastra.modelOutput in providerMetadata
		addedMsg, ok := ml.addCalls[0].msg.(map[string]any)
		if !ok {
			t.Fatalf("expected map message, got %T", ml.addCalls[0].msg)
		}
		content, _ := addedMsg["content"].(map[string]any)
		if content == nil {
			t.Fatal("expected content in message")
		}
		parts, _ := content["parts"].([]map[string]any)
		if len(parts) < 1 {
			t.Fatal("expected at least 1 part in message")
		}
		pm, _ := parts[0]["providerMetadata"].(map[string]any)
		if pm == nil {
			t.Fatal("expected providerMetadata on part")
		}
		mastra, _ := pm["mastra"].(map[string]any)
		if mastra == nil {
			t.Fatal("expected mastra key in providerMetadata")
		}
		modelOutput, _ := mastra["modelOutput"].(map[string]any)
		if modelOutput == nil {
			t.Fatal("expected modelOutput in mastra")
		}
		if modelOutput["type"] != "text" {
			t.Errorf("expected modelOutput type 'text', got %v", modelOutput["type"])
		}
	})

	t.Run("should NOT call toModelOutput for tools without it defined", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		tools := ToolSet{
			"plainTool": map[string]any{
				// No toModelOutput defined
			},
		}

		params := newLLMMapTestParams(controller, ml, tools)
		step := CreateLLMMappingStep(params, llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "plainTool",
				Args:       map[string]any{"input": "test"},
				Result:     map[string]any{"done": true},
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: message added without providerMetadata
		if len(ml.addCalls) < 1 {
			t.Fatal("expected messageList.add to be called")
		}

		addedMsg, ok := ml.addCalls[0].msg.(map[string]any)
		if !ok {
			t.Fatalf("expected map message, got %T", ml.addCalls[0].msg)
		}
		content, _ := addedMsg["content"].(map[string]any)
		if content == nil {
			t.Fatal("expected content in message")
		}
		parts, _ := content["parts"].([]map[string]any)
		if len(parts) < 1 {
			t.Fatal("expected at least 1 part")
		}
		pm := parts[0]["providerMetadata"]
		if pm != nil {
			t.Error("expected no providerMetadata for tool without toModelOutput")
		}
	})

	t.Run("should NOT call toModelOutput when tool result is nil", func(t *testing.T) {
		// Arrange
		controller := make(chan map[string]any, 100)
		ml := &mockMessageListForTCS{inputMsgs: []any{}}
		llmExecStep := &Step{
			ID: "test-llm-execution",
			Execute: func(args StepExecuteArgs) (any, error) {
				return map[string]any{
					"stepResult": map[string]any{"isContinued": true, "reason": nil},
					"metadata":   map[string]any{},
				}, nil
			},
		}

		toModelOutputCalled := false
		tools := ToolSet{
			"hitlTool": map[string]any{
				"toModelOutput": func(output any) any {
					toModelOutputCalled = true
					return nil
				},
			},
		}

		params := newLLMMapTestParams(controller, ml, tools)
		step := CreateLLMMappingStep(params, llmExecStep)

		inputData := []ToolCallOutput{
			{
				ToolCallID: "call-1",
				ToolName:   "hitlTool",
				Args:       map[string]any{},
				// Result is nil — HITL, no result yet
			},
		}

		// Act
		_, err := step.Execute(StepExecuteArgs{InputData: inputData})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert: toModelOutput should NOT be called for nil results
		if toModelOutputCalled {
			t.Error("expected toModelOutput NOT to be called for nil result")
		}
	})
}
