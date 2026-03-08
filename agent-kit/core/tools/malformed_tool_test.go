// Ported from: packages/core/src/tools/__tests__/malformed-tool.test.ts
package tools

import (
	"testing"
)

// NOTE: The TypeScript malformed-tool.test.ts tests the ensureToolProperties()
// function from ../../utils, which validates that tool objects passed to agents
// are actual tool instances (not factory functions). This function is not yet
// ported to Go.
//
// In Go, type safety provides some protection at compile time (you can't pass
// a func() *Tool where a *Tool is expected), but runtime validation of
// map-based tool registries would still need ensureToolProperties equivalent.

func TestMalformedToolValidation(t *testing.T) {
	t.Skip("not yet implemented - requires ensureToolProperties from utils package")

	// The following tests would verify:
	// 1. Passing a function (tool factory) instead of a tool object should panic/error.
	// 2. The error message should mention the tool key name for debugging.
	//
	// In Go, since tools are typed as *Tool, the compiler catches most of these
	// issues. However, when using map[string]any for dynamic tool registries,
	// runtime validation would be needed.
}

func TestMalformedToolDetection(t *testing.T) {
	t.Run("should detect valid Mastra tool via IsMastraTool", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "valid-tool",
			Description: "A valid tool",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				return map[string]any{"result": "ok"}, nil
			},
		})

		if !IsMastraTool(tool) {
			t.Error("expected IsMastraTool to return true for a valid tool")
		}
	})

	t.Run("should detect non-tool values via IsMastraTool", func(t *testing.T) {
		// A function is not a tool.
		factory := func() *Tool {
			return CreateTool(ToolAction{
				ID:          "factory-tool",
				Description: "A tool from factory",
			})
		}

		if IsMastraTool(factory) {
			t.Error("expected IsMastraTool to return false for a function")
		}
	})

	t.Run("should detect nil as non-tool", func(t *testing.T) {
		if IsMastraTool(nil) {
			t.Error("expected IsMastraTool to return false for nil")
		}
	})

	t.Run("should detect plain map as non-tool", func(t *testing.T) {
		plainMap := map[string]any{
			"id":          "not-a-tool",
			"description": "Just a map",
		}

		if IsMastraTool(plainMap) {
			t.Error("expected IsMastraTool to return false for a plain map without marker")
		}
	})

	t.Run("should detect map with marker as tool", func(t *testing.T) {
		markedMap := map[string]any{
			MastraToolMarker: true,
		}

		if !IsMastraTool(markedMap) {
			t.Error("expected IsMastraTool to return true for a map with marker")
		}
	})
}

func TestIsVercelTool(t *testing.T) {
	t.Run("should return false for Mastra tools", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "mastra-tool",
			Description: "A Mastra tool",
		})

		if IsVercelTool(tool) {
			t.Error("expected IsVercelTool to return false for a Mastra tool")
		}
	})

	t.Run("should return true for map with parameters field", func(t *testing.T) {
		vercelTool := map[string]any{
			"parameters": map[string]any{"type": "object"},
		}

		if !IsVercelTool(vercelTool) {
			t.Error("expected IsVercelTool to return true for map with parameters")
		}
	})

	t.Run("should return true for map with execute and inputSchema", func(t *testing.T) {
		vercelToolV5 := map[string]any{
			"execute":     func(args any, opts any) (any, error) { return nil, nil },
			"inputSchema": map[string]any{"type": "object"},
		}

		if !IsVercelTool(vercelToolV5) {
			t.Error("expected IsVercelTool to return true for map with execute + inputSchema")
		}
	})

	t.Run("should return false for nil", func(t *testing.T) {
		if IsVercelTool(nil) {
			t.Error("expected IsVercelTool to return false for nil")
		}
	})
}

func TestIsProviderDefinedTool(t *testing.T) {
	t.Run("should return true for provider-defined tool", func(t *testing.T) {
		providerTool := map[string]any{
			"type": "provider-defined",
			"id":   "google.google_search",
		}

		if !IsProviderDefinedTool(providerTool) {
			t.Error("expected true for provider-defined tool")
		}
	})

	t.Run("should return true for provider type tool", func(t *testing.T) {
		providerTool := map[string]any{
			"type": "provider",
			"id":   "openai.web_search",
		}

		if !IsProviderDefinedTool(providerTool) {
			t.Error("expected true for provider type tool")
		}
	})

	t.Run("should return false for function type tool", func(t *testing.T) {
		functionTool := map[string]any{
			"type": "function",
			"id":   "my-tool",
		}

		if IsProviderDefinedTool(functionTool) {
			t.Error("expected false for function type tool")
		}
	})

	t.Run("should return false for nil", func(t *testing.T) {
		if IsProviderDefinedTool(nil) {
			t.Error("expected false for nil")
		}
	})

	t.Run("should return false for map without type", func(t *testing.T) {
		noType := map[string]any{
			"id": "some-tool",
		}

		if IsProviderDefinedTool(noType) {
			t.Error("expected false for map without type")
		}
	})
}
