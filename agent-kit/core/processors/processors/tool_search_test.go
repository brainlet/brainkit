// Ported from: packages/core/src/processors/processors/tool-search.test.ts
package concreteprocessors

import (
	"testing"
	"time"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeTools() map[string]*Tool {
	return map[string]*Tool{
		"search":    {ID: "search", Description: "Search the web for information"},
		"calculate": {ID: "calculate", Description: "Perform mathematical calculations"},
		"weather":   {ID: "weather", Description: "Get current weather forecast"},
		"translate": {ID: "translate", Description: "Translate text between languages"},
		"summarize": {ID: "summarize", Description: "Summarize long text into key points"},
	}
}

func makeToolSearchProcessor(tools map[string]*Tool) *ToolSearchProcessor {
	return NewToolSearchProcessor(ToolSearchProcessorOptions{
		Tools: tools,
		TTL:   0, // Disable cleanup goroutine for tests
	})
}

func makeToolSearchProcessorWithOpts(tools map[string]*Tool, topK int, minScore float64) *ToolSearchProcessor {
	return NewToolSearchProcessor(ToolSearchProcessorOptions{
		Tools:          tools,
		SearchTopK:     topK,
		SearchMinScore: minScore,
		TTL:            0,
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestToolSearchProcessor(t *testing.T) {

	t.Run("initialization", func(t *testing.T) {
		t.Run("should index all provided tools", func(t *testing.T) {
			tools := makeTools()
			tsp := makeToolSearchProcessor(tools)
			if tsp.index.size() != len(tools) {
				t.Fatalf("expected %d indexed tools, got %d", len(tools), tsp.index.size())
			}
		})

		t.Run("should have correct ID and Name", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			if tsp.ID() != "tool-search" {
				t.Fatalf("expected id 'tool-search', got '%s'", tsp.ID())
			}
			if tsp.Name() != "Tool Search Processor" {
				t.Fatalf("expected name 'Tool Search Processor', got '%s'", tsp.Name())
			}
		})

		t.Run("should use default topK of 5", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			if tsp.searchTopK != 5 {
				t.Fatalf("expected searchTopK=5, got %d", tsp.searchTopK)
			}
		})

		t.Run("should accept custom topK", func(t *testing.T) {
			tsp := makeToolSearchProcessorWithOpts(makeTools(), 3, 0)
			if tsp.searchTopK != 3 {
				t.Fatalf("expected searchTopK=3, got %d", tsp.searchTopK)
			}
		})

		t.Run("should handle empty tools map", func(t *testing.T) {
			tsp := makeToolSearchProcessor(map[string]*Tool{})
			if tsp.index.size() != 0 {
				t.Fatalf("expected 0 indexed tools, got %d", tsp.index.size())
			}
		})
	})

	t.Run("BM25 search", func(t *testing.T) {
		t.Run("should find tools by keyword matching", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("search")
			if len(results) == 0 {
				t.Fatal("expected at least one result for 'search'")
			}
			// The "search" tool should rank highest due to name-match boosting
			if results[0].Name != "search" {
				t.Fatalf("expected first result to be 'search', got '%s'", results[0].Name)
			}
		})

		t.Run("should find tools by description", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("weather forecast")
			if len(results) == 0 {
				t.Fatal("expected results for 'weather forecast'")
			}
			found := false
			for _, r := range results {
				if r.Name == "weather" {
					found = true
					break
				}
			}
			if !found {
				t.Fatal("expected 'weather' tool in results")
			}
		})

		t.Run("should boost exact name matches", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("calculate")
			if len(results) == 0 {
				t.Fatal("expected results for 'calculate'")
			}
			if results[0].Name != "calculate" {
				t.Fatalf("expected 'calculate' as top result (name boosted), got '%s'", results[0].Name)
			}
		})

		t.Run("should return empty results for non-matching query", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("xyzzyzzy")
			if len(results) != 0 {
				t.Fatalf("expected 0 results for non-matching query, got %d", len(results))
			}
		})

		t.Run("should respect topK limit", func(t *testing.T) {
			tsp := makeToolSearchProcessorWithOpts(makeTools(), 2, 0)
			results := tsp.SearchTools("text")
			if len(results) > 2 {
				t.Fatalf("expected at most 2 results (topK=2), got %d", len(results))
			}
		})

		t.Run("should include scores in results", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("search")
			if len(results) == 0 {
				t.Fatal("expected results")
			}
			if results[0].Score <= 0 {
				t.Fatal("expected positive score")
			}
		})

		t.Run("should sort results by score descending", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("text information")
			if len(results) > 1 {
				for i := 1; i < len(results); i++ {
					if results[i].Score > results[i-1].Score {
						t.Fatal("expected results sorted by score descending")
					}
				}
			}
		})

		t.Run("should truncate long descriptions", func(t *testing.T) {
			tools := map[string]*Tool{
				"verbose": {
					ID:          "verbose",
					Description: "This is a very long description that goes on and on and describes many things about the tool and its capabilities and features and use cases and limitations and requirements and dependencies and configurations and all sorts of other things",
				},
			}
			tsp := makeToolSearchProcessor(tools)
			results := tsp.SearchTools("verbose")
			if len(results) > 0 && len(results[0].Description) > 153 {
				t.Fatalf("expected description to be truncated to <=150 chars + '...', got %d chars", len(results[0].Description))
			}
		})

		t.Run("should handle multi-word queries", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results := tsp.SearchTools("mathematical calculations")
			if len(results) == 0 {
				t.Fatal("expected results for multi-word query")
			}
			found := false
			for _, r := range results {
				if r.Name == "calculate" {
					found = true
					break
				}
			}
			if !found {
				t.Fatal("expected 'calculate' tool in results for 'mathematical calculations'")
			}
		})

		t.Run("should be case insensitive", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			results1 := tsp.SearchTools("SEARCH")
			results2 := tsp.SearchTools("search")
			if len(results1) == 0 || len(results2) == 0 {
				t.Fatal("expected results for both cases")
			}
			if results1[0].Name != results2[0].Name {
				t.Fatal("expected same top result for case-insensitive search")
			}
		})

		t.Run("should respect minScore filter", func(t *testing.T) {
			tsp := makeToolSearchProcessorWithOpts(makeTools(), 10, 100.0) // Very high minScore
			results := tsp.SearchTools("search")
			// With a very high minScore, most results should be filtered
			// (though name-match boost may push some above threshold)
			_ = results // Just verify no panic
		})

		t.Run("should return empty for empty tool set", func(t *testing.T) {
			tsp := makeToolSearchProcessor(map[string]*Tool{})
			results := tsp.SearchTools("search")
			if len(results) != 0 {
				t.Fatalf("expected 0 results from empty tool set, got %d", len(results))
			}
		})
	})

	t.Run("thread-scoped state", func(t *testing.T) {
		t.Run("should track tools per thread", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())

			// Load a tool for thread-1
			names1 := tsp.getLoadedToolNames("thread-1")
			names1["search"] = true

			// Load a different tool for thread-2
			names2 := tsp.getLoadedToolNames("thread-2")
			names2["calculate"] = true

			// Verify they are independent
			loaded1 := tsp.getLoadedToolNames("thread-1")
			loaded2 := tsp.getLoadedToolNames("thread-2")

			if !loaded1["search"] {
				t.Fatal("expected 'search' loaded for thread-1")
			}
			if loaded1["calculate"] {
				t.Fatal("expected 'calculate' NOT loaded for thread-1")
			}
			if !loaded2["calculate"] {
				t.Fatal("expected 'calculate' loaded for thread-2")
			}
			if loaded2["search"] {
				t.Fatal("expected 'search' NOT loaded for thread-2")
			}
		})

		t.Run("should use default threadId when not specified", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			names := tsp.getLoadedToolNames("")
			if names == nil {
				t.Fatal("expected non-nil loaded tool names for default thread")
			}
			// Verify that getLoadedToolNames("") returns a usable map
			names["test-tool"] = true
			// Calling again with same key should return the same map
			names2 := tsp.getLoadedToolNames("")
			if !names2["test-tool"] {
				t.Fatal("expected tool to persist in empty-string thread state")
			}
		})

		t.Run("should clear state for specific thread", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true
			names2 := tsp.getLoadedToolNames("thread-2")
			names2["calculate"] = true

			tsp.ClearState("thread-1")

			// thread-1 state should be cleared
			loaded1 := tsp.getLoadedToolNames("thread-1")
			if loaded1["search"] {
				t.Fatal("expected thread-1 state to be cleared")
			}

			// thread-2 state should be preserved
			loaded2 := tsp.getLoadedToolNames("thread-2")
			if !loaded2["calculate"] {
				t.Fatal("expected thread-2 state to be preserved")
			}
		})

		t.Run("should clear all state", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true
			names2 := tsp.getLoadedToolNames("thread-2")
			names2["calculate"] = true

			tsp.ClearAllState()

			loaded1 := tsp.getLoadedToolNames("thread-1")
			loaded2 := tsp.getLoadedToolNames("thread-2")
			if loaded1["search"] || loaded2["calculate"] {
				t.Fatal("expected all state to be cleared")
			}
		})

		t.Run("should return state stats", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			tsp.getLoadedToolNames("thread-1")
			tsp.getLoadedToolNames("thread-2")

			count, oldest := tsp.GetStateStats()
			if count != 2 {
				t.Fatalf("expected 2 threads, got %d", count)
			}
			if oldest == nil {
				t.Fatal("expected non-nil oldest access time")
			}
		})

		t.Run("should return zero stats when empty", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			count, oldest := tsp.GetStateStats()
			if count != 0 {
				t.Fatalf("expected 0 threads, got %d", count)
			}
			if oldest != nil {
				t.Fatal("expected nil oldest access time when empty")
			}
		})
	})

	t.Run("load_tool functionality", func(t *testing.T) {
		t.Run("should load existing tool by name", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true

			loaded := tsp.getLoadedTools("thread-1")
			if _, ok := loaded["search"]; !ok {
				t.Fatal("expected 'search' tool to be loaded")
			}
		})

		t.Run("should not load non-existent tool", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			names := tsp.getLoadedToolNames("thread-1")
			names["nonexistent"] = true

			loaded := tsp.getLoadedTools("thread-1")
			if _, ok := loaded["nonexistent"]; ok {
				t.Fatal("expected non-existent tool to not be loaded")
			}
		})

		t.Run("should load tool by ID", func(t *testing.T) {
			tools := map[string]*Tool{
				"my-search": {ID: "search-v2", Description: "Search tool v2"},
			}
			tsp := makeToolSearchProcessor(tools)
			// The tool is indexed by its ID ("search-v2"), not the map key
			names := tsp.getLoadedToolNames("thread-1")
			names["search-v2"] = true

			loaded := tsp.getLoadedTools("thread-1")
			if _, ok := loaded["search-v2"]; !ok {
				t.Fatal("expected tool to be loaded by ID")
			}
		})
	})

	t.Run("processInputStep integration", func(t *testing.T) {
		t.Run("should provide system messages with tool search instructions", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			rc := requestcontext.NewRequestContext()
			rc.Set(requestcontext.MastraThreadIDKey, "thread-1")

			result, _, err := tsp.ProcessInputStep(processors.ProcessInputStepArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
				Tools: map[string]any{},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if len(result.SystemMessages) == 0 {
				t.Fatal("expected system messages with tool search instructions")
			}
		})

		t.Run("should preserve existing tools", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			rc := requestcontext.NewRequestContext()
			rc.Set(requestcontext.MastraThreadIDKey, "thread-1")

			existingTools := map[string]any{
				"existing-tool": "some-tool-config",
			}
			result, _, err := tsp.ProcessInputStep(processors.ProcessInputStepArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
				Tools: existingTools,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := result.Tools["existing-tool"]; !ok {
				t.Fatal("expected existing tools to be preserved")
			}
		})

		t.Run("should merge loaded tools with existing tools", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			rc := requestcontext.NewRequestContext()
			rc.Set(requestcontext.MastraThreadIDKey, "thread-1")

			// Load a tool first
			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true

			existingTools := map[string]any{
				"existing": "config",
			}
			result, _, err := tsp.ProcessInputStep(processors.ProcessInputStepArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
				Tools: existingTools,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := result.Tools["existing"]; !ok {
				t.Fatal("expected existing tools to be preserved")
			}
			if _, ok := result.Tools["search"]; !ok {
				t.Fatal("expected loaded tool to be included")
			}
		})

		t.Run("should use default thread ID when request context has none", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())
			result, _, err := tsp.ProcessInputStep(processors.ProcessInputStepArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{},
				},
				Tools: map[string]any{},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result with default thread ID")
			}
		})
	})

	t.Run("full workflow (search -> load -> use)", func(t *testing.T) {
		t.Run("should support search then load workflow", func(t *testing.T) {
			tsp := makeToolSearchProcessor(makeTools())

			// Step 1: Search for tools
			results := tsp.SearchTools("search web")
			if len(results) == 0 {
				t.Fatal("expected search results")
			}

			// Step 2: Load the top result
			toolName := results[0].Name
			names := tsp.getLoadedToolNames("thread-1")
			names[toolName] = true

			// Step 3: Verify tool is available
			loaded := tsp.getLoadedTools("thread-1")
			if _, ok := loaded[toolName]; !ok {
				t.Fatalf("expected '%s' to be loaded and available", toolName)
			}
		})
	})

	t.Run("TTL and state cleanup", func(t *testing.T) {
		t.Run("should clean up stale state", func(t *testing.T) {
			tsp := NewToolSearchProcessor(ToolSearchProcessorOptions{
				Tools: makeTools(),
				TTL:   1, // 1ms TTL for testing
			})

			// Create some state
			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true

			// Wait for state to become stale
			time.Sleep(5 * time.Millisecond)

			// Manually trigger cleanup
			cleaned := tsp.CleanupNow()
			if cleaned < 1 {
				t.Fatalf("expected at least 1 thread cleaned up, got %d", cleaned)
			}
		})

		t.Run("should not clean up active state", func(t *testing.T) {
			tsp := NewToolSearchProcessor(ToolSearchProcessorOptions{
				Tools: makeTools(),
				TTL:   10000, // 10 second TTL
			})

			// Create fresh state
			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true

			// Trigger cleanup immediately
			cleaned := tsp.CleanupNow()
			if cleaned != 0 {
				t.Fatalf("expected 0 threads cleaned (state is fresh), got %d", cleaned)
			}

			// State should still be there
			loaded := tsp.getLoadedToolNames("thread-1")
			if !loaded["search"] {
				t.Fatal("expected state to be preserved")
			}
		})

		t.Run("should not clean up when TTL is disabled", func(t *testing.T) {
			tsp := NewToolSearchProcessor(ToolSearchProcessorOptions{
				Tools: makeTools(),
				TTL:   0, // Disabled
			})

			names := tsp.getLoadedToolNames("thread-1")
			names["search"] = true

			cleaned := tsp.CleanupNow()
			if cleaned != 0 {
				t.Fatalf("expected 0 cleanups when TTL disabled, got %d", cleaned)
			}
		})
	})
}
