// Ported from: packages/core/src/workspace/tools/edit-file.ts
package tools

import (
	"fmt"
	"strings"
)

// =============================================================================
// Edit File Tool
// =============================================================================

// EditFileInput holds the input for the edit_file tool.
type EditFileInput struct {
	// Path is the path to the file to edit.
	Path string `json:"path"`
	// OldString is the exact text to find and replace.
	OldString string `json:"old_string"`
	// NewString is the text to replace OldString with.
	NewString string `json:"new_string"`
	// ReplaceAll replaces all occurrences if true (default: false).
	ReplaceAll bool `json:"replace_all,omitempty"`
}

// ExecuteEditFile executes the edit_file tool.
func ExecuteEditFile(input *EditFileInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	fs := result.Filesystem

	if fs.ReadOnly() {
		return "", fmt.Errorf("workspace is in read-only mode. Cannot perform: edit_file")
	}

	raw, err := fs.ReadFile(input.Path, &ReadOptions{Encoding: "utf-8"})
	if err != nil {
		return "", err
	}

	content, ok := raw.(string)
	if !ok {
		return "Cannot edit binary files. Use the write file tool instead.", nil
	}

	replaceResult, err := ReplaceString(content, input.OldString, input.NewString, input.ReplaceAll)
	if err != nil {
		// StringNotFoundError and StringNotUniqueError are user-facing
		return err.Error(), nil
	}

	overwrite := true
	err = fs.WriteFile(input.Path, replaceResult.Content, &WriteOptions{Overwrite: &overwrite})
	if err != nil {
		return "", err
	}

	suffix := ""
	if replaceResult.Replacements != 1 {
		suffix = "s"
	}
	output := fmt.Sprintf("Replaced %d occurrence%s in %s", replaceResult.Replacements, suffix, input.Path)
	output += GetEditDiagnosticsText(ws, input.Path, replaceResult.Content)
	return output, nil
}

// =============================================================================
// String Replacement (inlined from workspace/line_utils.go for package isolation)
// =============================================================================

// ReplaceStringResult holds the result of a string replacement.
type ReplaceStringResult struct {
	Content      string
	Replacements int
}

// ReplaceString replaces a string in content, with validation for uniqueness.
func ReplaceString(content, oldString, newString string, replaceAll bool) (*ReplaceStringResult, error) {
	count := countOccurrences(content, oldString)

	if count == 0 {
		return nil, fmt.Errorf("The specified text was not found. Make sure you use the exact text from the file.")
	}

	if !replaceAll && count > 1 {
		return nil, fmt.Errorf(
			"The specified text appears %d times. Provide more surrounding context to make the match unique, or use replace_all to replace all occurrences.",
			count,
		)
	}

	if replaceAll {
		result := strings.ReplaceAll(content, oldString, newString)
		return &ReplaceStringResult{Content: result, Replacements: count}, nil
	}

	result := strings.Replace(content, oldString, newString, 1)
	return &ReplaceStringResult{Content: result, Replacements: 1}, nil
}

// countOccurrences counts occurrences of a string in content.
func countOccurrences(content, searchString string) int {
	if searchString == "" {
		return 0
	}
	count := 0
	position := 0
	for {
		idx := strings.Index(content[position:], searchString)
		if idx == -1 {
			break
		}
		count++
		position += idx + len(searchString)
	}
	return count
}
