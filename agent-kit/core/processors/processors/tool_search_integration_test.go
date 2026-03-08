// Ported from: packages/core/src/processors/processors/tool-search-integration.test.ts
package concreteprocessors

import (
	"strings"
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// createTSIArgs creates ProcessInputStepArgs for integration tests.
func createTSIArgs(threadID string, tools map[string]any) processors.ProcessInputStepArgs {
	rc := requestcontext.NewRequestContext()
	if threadID != "" {
		rc.Set(requestcontext.MastraThreadIDKey, threadID)
	}
	return processors.ProcessInputStepArgs{
		ProcessorMessageContext: processors.ProcessorMessageContext{
			ProcessorContext: processors.ProcessorContext{
				RequestContext: rc,
			},
		},
		Tools: tools,
	}
}

func TestToolSearchProcessorIntegration(t *testing.T) {
	t.Run("should allow dynamic tool discovery via SearchTools", func(t *testing.T) {
		// Create tools
		weatherTool := &Tool{ID: "weather", Description: "Get current weather for a location"}
		calculatorTool := &Tool{ID: "calculator", Description: "Perform basic arithmetic calculations"}
		emailTool := &Tool{ID: "send_email", Description: "Send an email to a recipient"}

		toolSearch := NewToolSearchProcessor(ToolSearchProcessorOptions{
			Tools: map[string]*Tool{
				"weather":    weatherTool,
				"calculator": calculatorTool,
				"send_email": emailTool,
			},
		})

		// First call - should get system messages
		args1 := createTSIArgs("test-thread-1", map[string]any{})
		result1, _, err := toolSearch.ProcessInputStep(args1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result1 == nil {
			t.Fatal("expected non-nil result")
		}
		if len(result1.SystemMessages) == 0 {
			t.Fatal("expected system messages to be injected")
		}

		// Simulate agent searching for tools
		searchResults := toolSearch.SearchTools("weather")
		if len(searchResults) == 0 {
			t.Fatal("expected search results for 'weather'")
		}
		if searchResults[0].Name != "weather" {
			t.Fatalf("expected first result to be 'weather', got '%s'", searchResults[0].Name)
		}

		// Simulate loading the weather tool
		loadedNames := toolSearch.getLoadedToolNames("test-thread-1")
		loadedNames["weather"] = true

		// Second call - should now include the loaded weather tool
		args2 := createTSIArgs("test-thread-1", map[string]any{})
		result2, _, err := toolSearch.ProcessInputStep(args2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result2.Tools["weather"] == nil {
			t.Fatal("expected weather tool to be available after loading")
		}

		// Verify loaded tool is the actual weather tool
		loadedTool, ok := result2.Tools["weather"].(*Tool)
		if !ok {
			t.Fatal("expected loaded tool to be *Tool type")
		}
		if loadedTool.ID != "weather" {
			t.Fatalf("expected loaded tool ID 'weather', got '%s'", loadedTool.ID)
		}
	})

	t.Run("should maintain thread isolation across multiple threads", func(t *testing.T) {
		tool1 := &Tool{ID: "tool1", Description: "First tool"}
		tool2 := &Tool{ID: "tool2", Description: "Second tool"}

		toolSearch := NewToolSearchProcessor(ToolSearchProcessorOptions{
			Tools: map[string]*Tool{
				"tool1": tool1,
				"tool2": tool2,
			},
		})

		// Thread 1: load tool1
		names1 := toolSearch.getLoadedToolNames("thread-1")
		names1["tool1"] = true

		// Thread 2: load tool2
		names2 := toolSearch.getLoadedToolNames("thread-2")
		names2["tool2"] = true

		// Verify thread 1 only has tool1 loaded
		args1 := createTSIArgs("thread-1", map[string]any{})
		result1, _, err := toolSearch.ProcessInputStep(args1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result1.Tools["tool1"] == nil {
			t.Fatal("expected tool1 to be available in thread-1")
		}
		if result1.Tools["tool2"] != nil {
			t.Fatal("expected tool2 to NOT be available in thread-1")
		}

		// Verify thread 2 only has tool2 loaded
		args2 := createTSIArgs("thread-2", map[string]any{})
		result2, _, err := toolSearch.ProcessInputStep(args2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result2.Tools["tool1"] != nil {
			t.Fatal("expected tool1 to NOT be available in thread-2")
		}
		if result2.Tools["tool2"] == nil {
			t.Fatal("expected tool2 to be available in thread-2")
		}
	})

	t.Run("should merge existing agent tools with loaded tools", func(t *testing.T) {
		alwaysAvailableTool := &Tool{ID: "always_available", Description: "This tool is always available"}
		dynamicTool := &Tool{ID: "dynamic_tool", Description: "This tool must be loaded dynamically"}

		toolSearch := NewToolSearchProcessor(ToolSearchProcessorOptions{
			Tools: map[string]*Tool{
				"dynamic_tool": dynamicTool,
			},
		})

		existingTools := map[string]any{
			"always_available": alwaysAvailableTool,
		}

		// First call - should have meta-tools + always available tool
		args1 := createTSIArgs("test-thread", existingTools)
		result1, _, err := toolSearch.ProcessInputStep(args1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result1.Tools["always_available"] == nil {
			t.Fatal("expected always_available tool to be preserved")
		}
		if result1.Tools["dynamic_tool"] != nil {
			t.Fatal("expected dynamic_tool to NOT be available before loading")
		}

		// Load the dynamic tool
		names := toolSearch.getLoadedToolNames("test-thread")
		names["dynamic_tool"] = true

		// Second call - should have always available + dynamic tool
		args2 := createTSIArgs("test-thread", existingTools)
		result2, _, err := toolSearch.ProcessInputStep(args2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result2.Tools["always_available"] == nil {
			t.Fatal("expected always_available tool to be preserved")
		}
		if result2.Tools["dynamic_tool"] == nil {
			t.Fatal("expected dynamic_tool to be available after loading")
		}
	})

	t.Run("should handle search with no results gracefully", func(t *testing.T) {
		tool := &Tool{ID: "specific_tool", Description: "A very specific tool"}

		toolSearch := NewToolSearchProcessor(ToolSearchProcessorOptions{
			Tools: map[string]*Tool{"specific_tool": tool},
		})

		// Search for something that doesn't match
		results := toolSearch.SearchTools("completely unrelated xyz abc 123")

		// Should return empty results
		if len(results) != 0 {
			t.Fatalf("expected 0 results for unrelated query, got %d", len(results))
		}
	})

	t.Run("should handle loading non-existent tool with suggestions", func(t *testing.T) {
		t.Skip("not yet implemented: load_tool meta-tool requires createTool porting for suggestion mechanism")
	})

	t.Run("should handle loading already-loaded tool gracefully", func(t *testing.T) {
		t.Skip("not yet implemented: load_tool meta-tool requires createTool porting")
	})

	t.Run("should include system message explaining meta-tools", func(t *testing.T) {
		tool := &Tool{ID: "test_tool", Description: "Test tool"}

		toolSearch := NewToolSearchProcessor(ToolSearchProcessorOptions{
			Tools: map[string]*Tool{"test_tool": tool},
		})

		args := createTSIArgs("test-thread", map[string]any{})
		result, _, err := toolSearch.ProcessInputStep(args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify system message was added
		if len(result.SystemMessages) == 0 {
			t.Fatal("expected system messages")
		}

		foundSearchToolsMention := false
		foundLoadToolMention := false
		for _, msg := range result.SystemMessages {
			content, ok := msg.Content.(string)
			if ok {
				if strings.Contains(content, "search_tools") {
					foundSearchToolsMention = true
				}
				if strings.Contains(content, "load_tool") {
					foundLoadToolMention = true
				}
			}
		}

		if !foundSearchToolsMention {
			t.Fatal("expected system message to mention search_tools")
		}
		if !foundLoadToolMention {
			t.Fatal("expected system message to mention load_tool")
		}
	})
}
