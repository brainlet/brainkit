// Ported from: packages/core/src/workspace/tools/file-stat.ts
package tools

import (
	"fmt"
)

// =============================================================================
// File Stat Tool
// =============================================================================

// FileStatInput holds the input for the file_stat tool.
type FileStatInput struct {
	// Path is the path to check.
	Path string `json:"path"`
}

// ExecuteFileStat executes the file_stat tool.
func ExecuteFileStat(input *FileStatInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	fs := result.Filesystem

	stat, err := fs.Stat(input.Path)
	if err != nil {
		// Check if it's a file-not-found error by message pattern
		return fmt.Sprintf("%s: not found", input.Path), nil
	}

	parts := []string{input.Path, fmt.Sprintf("Type: %s", stat.Type)}
	if stat.Size > 0 {
		parts = append(parts, fmt.Sprintf("Size: %d bytes", stat.Size))
	}
	if stat.ModifiedAt != "" {
		parts = append(parts, fmt.Sprintf("Modified: %s", stat.ModifiedAt))
	}

	output := ""
	for i, part := range parts {
		if i > 0 {
			output += " "
		}
		output += part
	}

	return output, nil
}
