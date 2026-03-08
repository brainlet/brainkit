// Ported from: packages/core/src/tools/unified-integration.test.ts
package tools

import (
	"strings"
	"testing"
)

// NOTE: The TypeScript unified-integration.test.ts tests tool execution across
// different contexts: Agent, Workflow, and Direct. The Agent and Workflow tests
// require MockLanguageModelV2, Agent, Workflow, createStep, etc. which are not
// yet ported. We port what is testable directly and skip context-dependent tests.

func TestToolUnifiedArguments_DirectExecution(t *testing.T) {
	t.Run("should handle direct tool execution with minimal context", func(t *testing.T) {
		var toolInputCapture any
		var toolContextCapture *ToolExecutionContext

		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "A test tool that captures its arguments",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				toolInputCapture = inputData
				toolContextCapture = ctx

				return map[string]any{
					"message":            "Processed",
					"hasWorkflowContext": ctx.Workflow != nil,
					"hasAgentContext":    ctx.Agent != nil,
				}, nil
			},
		})

		result, err := tool.Execute(
			map[string]any{"text": "Direct call", "count": float64(5)},
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Tool was called.
		if toolInputCapture == nil {
			t.Fatal("expected toolInputCapture to be set")
		}

		m, ok := toolInputCapture.(map[string]any)
		if !ok {
			t.Fatalf("expected map input, got %T", toolInputCapture)
		}
		if m["text"] != "Direct call" {
			t.Errorf("expected text=Direct call, got %v", m["text"])
		}

		// Should not have agent or workflow context.
		if toolContextCapture != nil && toolContextCapture.Agent != nil {
			t.Error("expected no agent context for direct execution")
		}
		if toolContextCapture != nil && toolContextCapture.Workflow != nil {
			t.Error("expected no workflow context for direct execution")
		}

		rm, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		if rm["hasWorkflowContext"] != false {
			t.Error("expected hasWorkflowContext=false")
		}
		if rm["hasAgentContext"] != false {
			t.Error("expected hasAgentContext=false")
		}
	})

	t.Run("should handle direct tool execution with agent context", func(t *testing.T) {
		var toolContextCapture *ToolExecutionContext

		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "A test tool that captures context",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				toolContextCapture = ctx
				return map[string]any{
					"message":            "Processed",
					"hasWorkflowContext": ctx.Workflow != nil,
					"hasAgentContext":    ctx.Agent != nil,
				}, nil
			},
		})

		_, err := tool.Execute(
			map[string]any{"text": "With agent context"},
			&ToolExecutionContext{
				Agent: &AgentToolExecutionContext{
					ToolCallID: "agent-call-123",
					Messages:   []any{},
					Suspend: func(payload any, opts *SuspendOptions) error {
						return nil
					},
				},
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if toolContextCapture == nil {
			t.Fatal("expected context to be captured")
		}
		if toolContextCapture.Agent == nil {
			t.Fatal("expected agent context")
		}
		if toolContextCapture.Agent.ToolCallID != "agent-call-123" {
			t.Errorf("expected toolCallId=agent-call-123, got %s", toolContextCapture.Agent.ToolCallID)
		}
		if toolContextCapture.Workflow != nil {
			t.Error("expected no workflow context")
		}
	})

	t.Run("should handle direct tool execution with workflow context", func(t *testing.T) {
		var toolContextCapture *ToolExecutionContext

		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "A test tool that captures context",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				toolContextCapture = ctx
				return map[string]any{
					"message":            "Processed",
					"hasWorkflowContext": ctx.Workflow != nil,
					"hasAgentContext":    ctx.Agent != nil,
					"workflowId":        "",
				}, nil
			},
		})

		_, err := tool.Execute(
			map[string]any{"text": "With workflow context"},
			&ToolExecutionContext{
				Workflow: &WorkflowToolExecutionContext{
					RunID:      "workflow-run-123",
					WorkflowID: "test-workflow",
				},
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if toolContextCapture == nil {
			t.Fatal("expected context to be captured")
		}
		if toolContextCapture.Workflow == nil {
			t.Fatal("expected workflow context")
		}
		if toolContextCapture.Workflow.RunID != "workflow-run-123" {
			t.Errorf("expected runId=workflow-run-123, got %s", toolContextCapture.Workflow.RunID)
		}
		if toolContextCapture.Workflow.WorkflowID != "test-workflow" {
			t.Errorf("expected workflowId=test-workflow, got %s", toolContextCapture.Workflow.WorkflowID)
		}
		if toolContextCapture.Agent != nil {
			t.Error("expected no agent context")
		}
	})
}

func TestToolUnifiedArguments_TypeSafety(t *testing.T) {
	t.Run("should enforce type safety for tool input via schema", func(t *testing.T) {
		// Schema that validates name is a string, age is present, email is present.
		schema := newMockSchemaWithValidation(func(data any) SafeParseResult {
			m, ok := data.(map[string]any)
			if !ok {
				return SafeParseResult{
					Success: false,
					Error:   &SchemaError{Issues: []SchemaIssue{{Message: "Expected object"}}},
				}
			}
			var issues []SchemaIssue
			if _, ok := m["name"].(string); !ok {
				issues = append(issues, SchemaIssue{Path: []string{"name"}, Message: "Expected string"})
			}
			if _, ok := m["age"]; !ok {
				issues = append(issues, SchemaIssue{Path: []string{"age"}, Message: "Required"})
			}
			if _, ok := m["email"].(string); !ok {
				issues = append(issues, SchemaIssue{Path: []string{"email"}, Message: "Expected string"})
			}
			if len(issues) > 0 {
				return SafeParseResult{Success: false, Error: &SchemaError{Issues: issues}}
			}
			return SafeParseResult{Success: true, Data: data}
		})

		tool := CreateTool(ToolAction{
			ID:          "typed-tool",
			Description: "Tool with strict types",
			InputSchema: schema,
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m := inputData.(map[string]any)
				name := m["name"].(string)
				age := m["age"]
				email := m["email"].(string)
				return map[string]any{
					"greeting": "Hello " + name + ", age " + strings.Repeat("x", int(age.(float64))) + ", email " + email,
				}, nil
			},
		})

		// Valid input.
		result, err := tool.Execute(map[string]any{
			"name":  "Alice",
			"age":   float64(30),
			"email": "alice@example.com",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			// Check if it's a validation error.
			if ve, isVE := result.(*ValidationError); isVE {
				t.Fatalf("unexpected validation error: %s", ve.Message)
			}
			t.Fatalf("expected map result, got %T", result)
		}
		greeting, _ := m["greeting"].(string)
		if !strings.Contains(greeting, "Hello Alice") {
			t.Errorf("expected greeting to contain 'Hello Alice', got %s", greeting)
		}

		// Invalid input should return validation error.
		errorResult, err := tool.Execute(map[string]any{
			"name":  123,
			"age":   float64(150),
			"email": "not-an-email",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ve, ok := errorResult.(*ValidationError)
		if !ok {
			t.Fatalf("expected *ValidationError, got %T: %v", errorResult, errorResult)
		}
		if !ve.Error {
			t.Error("expected error=true")
		}
		if !strings.Contains(ve.Message, "validation failed") {
			t.Errorf("expected validation failure message, got: %s", ve.Message)
		}
	})

	t.Run("should provide proper context types in execute function", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "context-typed-tool",
			Description: "Tool that uses context",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				// Verify context properties are accessible without panicking.
				_ = ctx.Mastra
				if ctx.Workflow != nil {
					_ = ctx.Workflow.RunID
					_ = ctx.Workflow.State
				}
				if ctx.Agent != nil {
					_ = ctx.Agent.Messages
				}
				return map[string]any{"success": true}, nil
			},
		})

		result, err := tool.Execute(map[string]any{"key": "test"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", result)
		}
		if m["success"] != true {
			t.Errorf("expected success=true, got %v", m["success"])
		}
	})
}

func TestToolUnifiedArguments_ErrorHandling(t *testing.T) {
	t.Run("should return validation error for invalid input", func(t *testing.T) {
		schema := newMockSchemaWithValidation(func(data any) SafeParseResult {
			m, ok := data.(map[string]any)
			if !ok {
				return SafeParseResult{
					Success: false,
					Error:   &SchemaError{Issues: []SchemaIssue{{Message: "Expected object"}}},
				}
			}
			if _, ok := m["text"].(string); !ok {
				return SafeParseResult{
					Success: false,
					Error: &SchemaError{Issues: []SchemaIssue{
						{Path: []string{"text"}, Message: "Expected string"},
					}},
				}
			}
			return SafeParseResult{Success: true, Data: data}
		})

		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "Test tool",
			InputSchema: schema,
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m := inputData.(map[string]any)
				return map[string]any{"message": "Processed " + m["text"].(string)}, nil
			},
		})

		// Direct call with invalid input.
		result, err := tool.Execute(map[string]any{
			"text":  123,
			"count": "not a number",
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ve, ok := result.(*ValidationError)
		if !ok {
			t.Fatalf("expected *ValidationError, got %T: %v", result, result)
		}
		if !ve.Error {
			t.Error("expected error=true")
		}
		if !strings.Contains(ve.Message, "validation failed") {
			t.Errorf("expected validation failed message, got: %s", ve.Message)
		}
	})

	t.Run("should handle tool execution errors gracefully", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "error-tool",
			Description: "Tool that throws",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				shouldFail, _ := m["shouldFail"].(bool)
				if shouldFail {
					return nil, &toolError{message: "Tool execution failed"}
				}
				return map[string]any{"success": true}, nil
			},
		})

		// Success case.
		successResult, err := tool.Execute(map[string]any{"shouldFail": false}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := successResult.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", successResult)
		}
		if m["success"] != true {
			t.Errorf("expected success=true, got %v", m["success"])
		}

		// Error case.
		_, err = tool.Execute(map[string]any{"shouldFail": true}, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Tool execution failed") {
			t.Errorf("expected 'Tool execution failed' error, got: %v", err)
		}
	})
}

func TestToolUnifiedArguments_MigrationExamples(t *testing.T) {
	t.Run("should demonstrate migration from old to new tool structure", func(t *testing.T) {
		newTool := CreateTool(ToolAction{
			ID:          "migrated-tool",
			Description: "A migrated tool",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				data, _ := m["data"].(string)

				// Clean, organized context.
				if ctx.Workflow != nil {
					_ = ctx.Workflow.WorkflowID // accessible
				}
				if ctx.Agent != nil {
					_ = ctx.Agent.ToolCallID // accessible
				}

				return map[string]any{"processed": strings.ToUpper(data)}, nil
			},
		})

		result, err := newTool.Execute(map[string]any{"data": "test"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", result)
		}
		if m["processed"] != "TEST" {
			t.Errorf("expected processed=TEST, got %v", m["processed"])
		}
	})
}

func TestToolUnifiedArguments_AgentExecution(t *testing.T) {
	t.Skip("not yet implemented - requires Agent, MockLanguageModelV2")
}

func TestToolUnifiedArguments_WorkflowExecution(t *testing.T) {
	t.Skip("not yet implemented - requires Workflow, createStep, createWorkflow")
}

// toolError is a simple error implementation for testing.
type toolError struct {
	message string
}

func (e *toolError) Error() string {
	return e.message
}
