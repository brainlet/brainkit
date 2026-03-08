// Ported from: packages/core/src/workspace/tools/mkdir.ts
package tools

import (
	"fmt"
)

// =============================================================================
// Mkdir Tool
// =============================================================================

// MkdirInput holds the input for the mkdir tool.
type MkdirInput struct {
	// Path is the path of the directory to create.
	Path string `json:"path"`
	// Recursive creates parent directories if they do not exist (default: true).
	Recursive *bool `json:"recursive,omitempty"`
}

// ExecuteMkdir executes the mkdir tool.
func ExecuteMkdir(input *MkdirInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	fs := result.Filesystem

	if fs.ReadOnly() {
		return "", fmt.Errorf("workspace is in read-only mode. Cannot perform: mkdir")
	}

	recursive := true
	if input.Recursive != nil {
		recursive = *input.Recursive
	}

	err = fs.Mkdir(input.Path, &MkdirOptions{Recursive: recursive})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Created directory %s", input.Path), nil
}
