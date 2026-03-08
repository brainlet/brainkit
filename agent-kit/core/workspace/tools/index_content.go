// Ported from: packages/core/src/workspace/tools/index-content.ts
package tools

import (
	"fmt"
)

// =============================================================================
// Index Content Tool
// =============================================================================

// IndexContentInput holds the input for the index tool.
type IndexContentInput struct {
	// Path is the document ID/path for search results.
	Path string `json:"path"`
	// Content is the text content to index.
	Content string `json:"content"`
	// Metadata is optional metadata to store with the document.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ExecuteIndexContent executes the index tool.
func ExecuteIndexContent(input *IndexContentInput, ctx *ToolContext) (string, error) {
	ws, err := RequireWorkspace(ctx)
	if err != nil {
		return "", err
	}

	var opts *IndexOptions
	if input.Metadata != nil {
		opts = &IndexOptions{Metadata: input.Metadata}
	}

	err = ws.Index(input.Path, input.Content, opts)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Indexed %s", input.Path), nil
}
