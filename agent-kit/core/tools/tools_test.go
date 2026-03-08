// Ported from: packages/core/src/tools/tools.test.ts
package tools

import (
	"reflect"
	"sync/atomic"
	"testing"

	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// mockFindUser simulates the TS mock that looks up users by name.
func mockFindUser(name string) map[string]any {
	list := []map[string]any{
		{"name": "Dero Israel", "email": "dero@mail.com"},
		{"name": "Ife Dayo", "email": "dayo@mail.com"},
		{"name": "Tao Feeq", "email": "feeq@mail.com"},
	}
	for _, u := range list {
		if u["name"] == name {
			return u
		}
	}
	return map[string]any{"message": "User not found"}
}

func TestCreateTool(t *testing.T) {
	var callCount int64

	testTool := CreateTool(ToolAction{
		ID:          "Test tool",
		Description: "This is a test tool that returns the name and email",
		Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
			atomic.AddInt64(&callCount, 1)
			m, ok := inputData.(map[string]any)
			if !ok {
				return map[string]any{"message": "invalid input"}, nil
			}
			name, _ := m["name"].(string)
			return mockFindUser(name), nil
		},
	})

	t.Run("should call mockFindUser", func(t *testing.T) {
		atomic.StoreInt64(&callCount, 0)

		_, err := testTool.Execute(
			map[string]any{"name": "Dero Israel"},
			&ToolExecutionContext{
				RequestContext: requestcontext.NewRequestContext(),
				Agent: &AgentToolExecutionContext{
					ToolCallID: "123",
					Messages:   []any{},
				},
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got := atomic.LoadInt64(&callCount)
		if got != 1 {
			t.Errorf("expected mockFindUser to be called 1 time, got %d", got)
		}
	})

	t.Run("should return an object containing Dero Israel as name and dero@mail.com as email", func(t *testing.T) {
		result, err := testTool.Execute(
			map[string]any{"name": "Dero Israel"},
			&ToolExecutionContext{
				RequestContext: requestcontext.NewRequestContext(),
				Agent: &AgentToolExecutionContext{
					ToolCallID: "123",
					Messages:   []any{},
				},
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := map[string]any{"name": "Dero Israel", "email": "dero@mail.com"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})

	t.Run("should return an object containing User not found message", func(t *testing.T) {
		result, err := testTool.Execute(
			map[string]any{"name": "Taofeeq Oluderu"},
			&ToolExecutionContext{
				RequestContext: requestcontext.NewRequestContext(),
				Agent: &AgentToolExecutionContext{
					ToolCallID: "123",
					Messages:   []any{},
				},
			},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := map[string]any{"message": "User not found"}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("expected %v, got %v", expected, result)
		}
	})
}

func TestCreateToolWithProviderOptions(t *testing.T) {
	t.Run("should preserve providerOptions when creating a tool", func(t *testing.T) {
		toolWithProviderOptions := CreateTool(ToolAction{
			ID:          "cache-control-tool",
			Description: "A tool with cache control settings",
			ProviderOptions: map[string]map[string]any{
				"anthropic": {
					"cacheControl": map[string]any{"type": "ephemeral"},
				},
			},
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				city, _ := m["city"].(string)
				return map[string]any{"attractions": "Attractions in " + city}, nil
			},
		})

		expected := map[string]map[string]any{
			"anthropic": {
				"cacheControl": map[string]any{"type": "ephemeral"},
			},
		}
		if !reflect.DeepEqual(toolWithProviderOptions.ProviderOptions, expected) {
			t.Errorf("expected %v, got %v", expected, toolWithProviderOptions.ProviderOptions)
		}
	})

	t.Run("should support multiple provider options", func(t *testing.T) {
		toolWithMultipleProviders := CreateTool(ToolAction{
			ID:          "multi-provider-tool",
			Description: "A tool with multiple provider options",
			ProviderOptions: map[string]map[string]any{
				"anthropic": {
					"cacheControl": map[string]any{"type": "ephemeral"},
				},
				"openai": {
					"someOption": "value",
				},
			},
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				query, _ := m["query"].(string)
				return map[string]any{"result": query}, nil
			},
		})

		expected := map[string]map[string]any{
			"anthropic": {
				"cacheControl": map[string]any{"type": "ephemeral"},
			},
			"openai": {
				"someOption": "value",
			},
		}
		if !reflect.DeepEqual(toolWithMultipleProviders.ProviderOptions, expected) {
			t.Errorf("expected %v, got %v", expected, toolWithMultipleProviders.ProviderOptions)
		}
	})

	t.Run("should work without providerOptions", func(t *testing.T) {
		toolWithoutProviderOptions := CreateTool(ToolAction{
			ID:          "no-provider-options-tool",
			Description: "A tool without provider options",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				input, _ := m["input"].(string)
				return map[string]any{"output": input}, nil
			},
		})

		if toolWithoutProviderOptions.ProviderOptions != nil {
			t.Errorf("expected nil providerOptions, got %v", toolWithoutProviderOptions.ProviderOptions)
		}
	})

	t.Run("should preserve providerOptions through NewTool constructor", func(t *testing.T) {
		tool := NewTool(ToolAction{
			ID:          "direct-tool",
			Description: "Tool created directly with constructor",
			ProviderOptions: map[string]map[string]any{
				"anthropic": {
					"cacheControl": map[string]any{"type": "ephemeral"},
				},
			},
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				value, _ := m["value"].(string)
				return map[string]any{"result": value}, nil
			},
		})

		expected := map[string]map[string]any{
			"anthropic": {
				"cacheControl": map[string]any{"type": "ephemeral"},
			},
		}
		if !reflect.DeepEqual(tool.ProviderOptions, expected) {
			t.Errorf("expected %v, got %v", expected, tool.ProviderOptions)
		}
	})
}
