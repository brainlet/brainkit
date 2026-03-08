// Ported from: packages/core/src/workspace/tools/search.ts
package tools

import (
	"fmt"
	"strings"
)

// =============================================================================
// Search Tool
// =============================================================================

// SearchInput holds the input for the search tool.
type SearchInput struct {
	// Query is the search query string.
	Query string `json:"query"`
	// TopK is the maximum number of results to return (default: 5).
	TopK int `json:"topK,omitempty"`
	// Mode is the search mode: "bm25", "vector", or "hybrid".
	Mode string `json:"mode,omitempty"`
	// MinScore is the minimum score threshold.
	MinScore *float64 `json:"minScore,omitempty"`
}

// ExecuteSearch executes the search tool.
func ExecuteSearch(input *SearchInput, ctx *ToolContext) (string, error) {
	ws, err := RequireWorkspace(ctx)
	if err != nil {
		return "", err
	}

	topK := input.TopK
	if topK <= 0 {
		topK = 5
	}

	results, err := ws.Search(input.Query, &SearchOptions{
		TopK:     topK,
		Mode:     input.Mode,
		MinScore: input.MinScore,
	})
	if err != nil {
		return "", err
	}

	// Determine effective mode
	effectiveMode := input.Mode
	if effectiveMode == "" {
		if ws.CanHybrid() {
			effectiveMode = "hybrid"
		} else if ws.CanVector() {
			effectiveMode = "vector"
		} else {
			effectiveMode = "bm25"
		}
	}

	var lines []string
	for _, r := range results {
		lineInfo := ""
		if r.LineRange != nil {
			lineInfo = fmt.Sprintf(":%d-%d", r.LineRange.Start, r.LineRange.End)
		}
		lines = append(lines, fmt.Sprintf("%s%s: %s", r.ID, lineInfo, r.Content))
	}

	lines = append(lines, "---")
	resultWord := "results"
	if len(results) == 1 {
		resultWord = "result"
	}
	lines = append(lines, fmt.Sprintf("%d %s (%s search)", len(results), resultWord, effectiveMode))

	return strings.Join(lines, "\n"), nil
}
