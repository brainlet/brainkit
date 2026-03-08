// Ported from: packages/core/src/workspace/tools/write-file.ts
package tools

import (
	"fmt"
)

// =============================================================================
// Write File Tool
// =============================================================================

// WriteFileInput holds the input for the write_file tool.
type WriteFileInput struct {
	// Path is the path where to write the file.
	Path string `json:"path"`
	// Content is the content to write to the file.
	Content string `json:"content"`
	// Overwrite controls whether to overwrite existing files (default: true).
	Overwrite *bool `json:"overwrite,omitempty"`
}

// ExecuteWriteFile executes the write_file tool.
func ExecuteWriteFile(input *WriteFileInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	fs := result.Filesystem

	if fs.ReadOnly() {
		return "", fmt.Errorf("workspace is in read-only mode. Cannot perform: write_file")
	}

	overwrite := true
	if input.Overwrite != nil {
		overwrite = *input.Overwrite
	}

	err = fs.WriteFile(input.Path, input.Content, &WriteOptions{Overwrite: &overwrite})
	if err != nil {
		return "", err
	}

	size := len([]byte(input.Content))
	output := fmt.Sprintf("Wrote %d bytes to %s", size, input.Path)
	output += GetEditDiagnosticsText(ws, input.Path, input.Content)
	return output, nil
}
