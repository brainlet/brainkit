// Ported from: packages/core/src/workspace/tools/delete-file.ts
package tools

import (
	"fmt"
)

// =============================================================================
// Delete File Tool
// =============================================================================

// DeleteFileInput holds the input for the delete tool.
type DeleteFileInput struct {
	// Path is the path to the file or directory to delete.
	Path string `json:"path"`
	// Recursive deletes directories and their contents recursively (default: false).
	Recursive bool `json:"recursive,omitempty"`
}

// ExecuteDeleteFile executes the delete tool.
func ExecuteDeleteFile(input *DeleteFileInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	fs := result.Filesystem

	if fs.ReadOnly() {
		return "", fmt.Errorf("workspace is in read-only mode. Cannot perform: delete")
	}

	stat, err := fs.Stat(input.Path)
	if err != nil {
		return "", err
	}

	if stat.Type == "directory" {
		err = fs.Rmdir(input.Path, &RemoveOptions{
			Recursive: input.Recursive,
			Force:     input.Recursive,
		})
	} else {
		err = fs.DeleteFile(input.Path, nil)
	}

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Deleted %s", input.Path), nil
}
