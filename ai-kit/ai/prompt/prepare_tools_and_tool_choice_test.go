// Ported from: packages/ai/src/prompt/prepare-tools-and-tool-choice.test.ts
package prompt

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func strPtr(s string) *string { return &s }
func boolPtr2(b bool) *bool  { return &b }

func TestPrepareToolsAndToolChoice(t *testing.T) {
	t.Run("should return nil for both tools and toolChoice when tools is not provided", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(nil, nil, nil)
		assert.Nil(t, result.Tools)
		assert.Nil(t, result.ToolChoice)
	})

	t.Run("should return nil for both tools and toolChoice when tools is empty", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(ToolSet{}, nil, nil)
		assert.Nil(t, result.Tools)
		assert.Nil(t, result.ToolChoice)
	})

	t.Run("should return all tools when activeTools is not provided", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description: strPtr("Tool 1 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		}

		result := PrepareToolsAndToolChoice(tools, nil, nil)
		assert.NotNil(t, result.Tools)
		assert.Equal(t, 1, len(result.Tools))
		assert.NotNil(t, result.ToolChoice)
		assert.Equal(t, "auto", result.ToolChoice.Type)
	})

	t.Run("should filter tools based on activeTools", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description: strPtr("Tool 1 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
			"tool2": {
				Description: strPtr("Tool 2 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		}

		result := PrepareToolsAndToolChoice(tools, nil, []string{"tool1"})
		assert.Equal(t, 1, len(result.Tools))

		ft := result.Tools[0].(LanguageModelV4FunctionTool)
		assert.Equal(t, "tool1", ft.Name)
	})

	t.Run("should handle string toolChoice", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description: strPtr("Tool 1 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		}

		result := PrepareToolsAndToolChoice(tools, &ToolChoice{Type: "none"}, nil)
		assert.NotNil(t, result.ToolChoice)
		assert.Equal(t, "none", result.ToolChoice.Type)
	})

	t.Run("should handle object toolChoice with toolName", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description: strPtr("Tool 1 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
		}

		result := PrepareToolsAndToolChoice(tools, &ToolChoice{
			Type:     "tool",
			ToolName: strPtr("tool1"),
		}, nil)
		assert.NotNil(t, result.ToolChoice)
		assert.Equal(t, "tool", result.ToolChoice.Type)
		assert.Equal(t, "tool1", *result.ToolChoice.ToolName)
	})

	t.Run("should handle provider-defined tool type", func(t *testing.T) {
		tools := ToolSet{
			"funcTool": {
				Description: strPtr("Function tool"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
			},
			"providerTool": {
				Type: ToolTypeProvider,
				ID:   "provider.tool-id",
				Args: map[string]string{"key": "value"},
			},
		}

		result := PrepareToolsAndToolChoice(tools, nil, nil)
		assert.Equal(t, 2, len(result.Tools))

		// Find the provider tool
		var foundProvider bool
		for _, tool := range result.Tools {
			if pt, ok := tool.(LanguageModelV4ProviderTool); ok {
				assert.Equal(t, "provider", pt.Type)
				assert.Equal(t, "providerTool", pt.Name)
				assert.Equal(t, "provider.tool-id", pt.ID)
				foundProvider = true
			}
		}
		assert.True(t, foundProvider)
	})

	t.Run("should pass through provider options", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description: strPtr("Tool 1 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
				ProviderOptions: ProviderOptions{
					"aProvider": {"aSetting": "aValue"},
				},
			},
		}

		result := PrepareToolsAndToolChoice(tools, nil, nil)
		ft := result.Tools[0].(LanguageModelV4FunctionTool)
		assert.NotNil(t, ft.ProviderOptions)
		assert.Equal(t, "aValue", ft.ProviderOptions["aProvider"]["aSetting"])
	})

	t.Run("should pass through strict mode setting", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description: strPtr("Tool 1 description"),
				InputSchema: json.RawMessage(`{"type":"object"}`),
				Strict:      boolPtr2(true),
			},
		}

		result := PrepareToolsAndToolChoice(tools, nil, nil)
		ft := result.Tools[0].(LanguageModelV4FunctionTool)
		assert.NotNil(t, ft.Strict)
		assert.True(t, *ft.Strict)
	})

	t.Run("should pass through input examples", func(t *testing.T) {
		tools := ToolSet{
			"tool1": {
				Description:   strPtr("Tool 1 description"),
				InputSchema:   json.RawMessage(`{"type":"object"}`),
				InputExamples: []interface{}{map[string]interface{}{"input": map[string]string{"city": "New York"}}},
			},
		}

		result := PrepareToolsAndToolChoice(tools, nil, nil)
		ft := result.Tools[0].(LanguageModelV4FunctionTool)
		assert.NotNil(t, ft.InputExamples)
		assert.Equal(t, 1, len(ft.InputExamples))
	})
}
