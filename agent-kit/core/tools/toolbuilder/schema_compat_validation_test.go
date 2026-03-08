// Ported from: packages/core/src/tools/tool-builder/schema-compat-validation.test.ts
package toolbuilder

import (
	"testing"

	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/tools"
)

// NOTE: The TypeScript schema-compat-validation.test.ts tests are heavily dependent
// on Zod schemas, AnthropicSchemaCompatLayer, OpenAIReasoningSchemaCompatLayer,
// and other schema-compat packages that are not yet ported to Go.
//
// Since CoreToolBuilder's validation stubs currently pass through all input/output
// without real schema validation (see validation stub functions in builder.go),
// we test what we can at the CoreToolBuilder level and skip tests that require
// real schema compatibility layers.

func TestCoreToolBuilderSchemaCompatValidation(t *testing.T) {
	t.Run("should build a CoreTool from a ToolAction", func(t *testing.T) {
		toolAction := &tools.ToolAction{
			ID:          "test-tool",
			Description: "A test tool with string constraints",
			Execute: func(inputData any, ctx *tools.ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				msg, _ := m["message"].(string)
				return map[string]any{"result": "Received: " + msg}, nil
			},
		}

		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: toolAction,
			Options: ToolOptions{
				Name:           "test-tool",
				RequestContext: requestcontext.NewRequestContext(),
			},
		})

		coreTool := builder.Build()

		if coreTool.Description != "A test tool with string constraints" {
			t.Errorf("expected description, got %s", coreTool.Description)
		}
		if coreTool.Type != tools.CoreToolTypeFunction {
			t.Errorf("expected type=function, got %s", coreTool.Type)
		}
	})

	t.Run("should execute tool through CoreToolBuilder", func(t *testing.T) {
		toolAction := &tools.ToolAction{
			ID:          "exec-test",
			Description: "Test execution through builder",
			Execute: func(inputData any, ctx *tools.ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				msg, _ := m["message"].(string)
				return map[string]any{"result": "Received: " + msg}, nil
			},
		}

		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: toolAction,
			Options: ToolOptions{
				Name:           "exec-test",
				RequestContext: requestcontext.NewRequestContext(),
			},
		})

		coreTool := builder.Build()

		if coreTool.Execute == nil {
			t.Fatal("expected Execute to be defined")
		}

		result, err := coreTool.Execute(
			map[string]any{"message": "Hi there"},
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T: %v", result, result)
		}
		if m["result"] != "Received: Hi there" {
			t.Errorf("expected 'Received: Hi there', got %v", m["result"])
		}
	})

	t.Run("should use schema-compat transformed schema for validation", func(t *testing.T) {
		t.Skip("not yet implemented - requires AnthropicSchemaCompatLayer and Zod schemas")

		// This test would verify:
		// 1. The parameters sent to the LLM are transformed (constraints removed)
		// 2. The validation uses the SAME transformed schema
		// 3. Input that passes the transformed schema should succeed
	})

	t.Run("should validate against transformed schema for number constraints", func(t *testing.T) {
		t.Skip("not yet implemented - requires schema-compat layers")
	})

	t.Run("should demonstrate the bug: validation rejects input that LLM was told is valid", func(t *testing.T) {
		t.Skip("not yet implemented - requires AnthropicSchemaCompatLayer")
	})

	t.Run("should handle OpenAI o3 reasoning model converting optional to nullable", func(t *testing.T) {
		t.Skip("not yet implemented - requires OpenAIReasoningSchemaCompatLayer")
	})

	t.Run("should respect structured outputs for v2 models and preserve enums/constraints", func(t *testing.T) {
		t.Skip("not yet implemented - requires OpenAISchemaCompatLayer and zodToJsonSchema")
	})
}

func TestCoreToolBuilderLogMessages(t *testing.T) {
	t.Run("should create log messages without agent name", func(t *testing.T) {
		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: &tools.ToolAction{
				ID:          "log-test",
				Description: "Test logging",
			},
			Options: ToolOptions{
				Name: "log-test",
			},
		})

		logMsgs := builder.createLogMessageOptions(LogOptions{
			ToolName: "log-test",
		})

		if logMsgs.Start != "Executing tool log-test" {
			t.Errorf("expected 'Executing tool log-test', got %s", logMsgs.Start)
		}
		if logMsgs.Error != "Failed tool execution" {
			t.Errorf("expected 'Failed tool execution', got %s", logMsgs.Error)
		}
	})

	t.Run("should create log messages with agent name", func(t *testing.T) {
		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: &tools.ToolAction{
				ID:          "agent-log-test",
				Description: "Test agent logging",
			},
			Options: ToolOptions{
				Name: "agent-log-test",
			},
		})

		logMsgs := builder.createLogMessageOptions(LogOptions{
			AgentName: "MyAgent",
			ToolName:  "agent-log-test",
			Type:      LogTypeTool,
		})

		if logMsgs.Start != "[Agent:MyAgent] - Executing tool agent-log-test" {
			t.Errorf("expected agent-prefixed start message, got %s", logMsgs.Start)
		}
		if logMsgs.Error != "[Agent:MyAgent] - Failed tool execution" {
			t.Errorf("expected agent-prefixed error message, got %s", logMsgs.Error)
		}
	})

	t.Run("should create log messages for toolset type", func(t *testing.T) {
		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: &tools.ToolAction{
				ID:          "toolset-test",
				Description: "Test toolset logging",
			},
			Options: ToolOptions{
				Name: "toolset-test",
			},
		})

		logMsgs := builder.createLogMessageOptions(LogOptions{
			AgentName: "MyAgent",
			ToolName:  "toolset-test",
			Type:      LogTypeToolset,
		})

		if logMsgs.Start != "[Agent:MyAgent] - Executing toolset toolset-test" {
			t.Errorf("expected toolset log message, got %s", logMsgs.Start)
		}
	})
}

func TestCoreToolBuilderProviderTool(t *testing.T) {
	t.Run("should build provider-defined tool", func(t *testing.T) {
		providerTool := map[string]any{
			"type":        "provider-defined",
			"id":          "google.google_search",
			"description": "Google search tool",
		}

		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: providerTool,
			Options: ToolOptions{
				Name: "google_search",
			},
		})

		coreTool := builder.Build()

		if coreTool.Type != tools.CoreToolTypeProviderDefined {
			t.Errorf("expected type=provider-defined, got %s", coreTool.Type)
		}
		if coreTool.ID != "google.google_search" {
			t.Errorf("expected ID=google.google_search, got %s", coreTool.ID)
		}
	})

	t.Run("should not detect non-provider tools as provider-defined", func(t *testing.T) {
		normalTool := &tools.ToolAction{
			ID:          "normal-tool",
			Description: "A normal tool",
			Execute: func(inputData any, ctx *tools.ToolExecutionContext) (any, error) {
				return nil, nil
			},
		}

		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: normalTool,
			Options: ToolOptions{
				Name: "normal-tool",
			},
		})

		coreTool := builder.Build()

		if coreTool.Type != tools.CoreToolTypeFunction {
			t.Errorf("expected type=function, got %s", coreTool.Type)
		}
	})
}

func TestCoreToolBuilderV5(t *testing.T) {
	t.Run("should build V5-compatible tool format", func(t *testing.T) {
		toolAction := &tools.ToolAction{
			ID:          "v5-test",
			Description: "V5 format test",
			InputSchema: &tools.SchemaWithValidation{
				Schema: map[string]any{"type": "object"},
			},
			Execute: func(inputData any, ctx *tools.ToolExecutionContext) (any, error) {
				return map[string]any{"ok": true}, nil
			},
		}

		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: toolAction,
			Options: ToolOptions{
				Name: "v5-test",
			},
		})

		v5Tool := builder.BuildV5()

		if v5Tool["type"] != tools.CoreToolTypeFunction {
			t.Errorf("expected type=function, got %v", v5Tool["type"])
		}
		if v5Tool["description"] != "V5 format test" {
			t.Errorf("expected description, got %v", v5Tool["description"])
		}
		if v5Tool["parameters"] == nil {
			t.Error("expected parameters to be defined")
		}
		if v5Tool["inputSchema"] == nil {
			t.Error("expected inputSchema to be defined")
		}
	})

	t.Run("should panic when parameters are nil", func(t *testing.T) {
		toolAction := &tools.ToolAction{
			ID:          "no-params",
			Description: "No parameters",
			Execute: func(inputData any, ctx *tools.ToolExecutionContext) (any, error) {
				return nil, nil
			},
		}

		builder := NewCoreToolBuilder(CoreToolBuilderInput{
			OriginalTool: toolAction,
			Options: ToolOptions{
				Name: "no-params",
			},
		})

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil parameters")
			}
		}()

		builder.BuildV5()
	})
}
