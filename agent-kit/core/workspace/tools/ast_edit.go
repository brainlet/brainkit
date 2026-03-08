// Ported from: packages/core/src/workspace/tools/ast-edit.ts
package tools

import (
	"fmt"
	"strings"
)

// =============================================================================
// AST Edit Types
// =============================================================================

// ASTReplacement represents a text replacement with start and end byte offsets.
type ASTReplacement struct {
	Start int
	End   int
	Text  string
}

// ASTTransformResult holds the result of an AST transformation.
type ASTTransformResult struct {
	Content string
	Count   int
	Error   string
}

// ImportSpec specifies an import to add.
type ImportSpec struct {
	// Module is the module to import from.
	Module string `json:"module"`
	// Names are the names to import.
	Names []string `json:"names"`
	// IsDefault indicates the first name is a default import.
	IsDefault bool `json:"isDefault,omitempty"`
}

// =============================================================================
// AST Edit Tool Input
// =============================================================================

// ASTEditInput holds the input for the ast_edit tool.
type ASTEditInput struct {
	// Path is the path to the file to edit.
	Path string `json:"path"`
	// Pattern is an AST pattern to search for (supports $VARIABLE placeholders).
	Pattern string `json:"pattern,omitempty"`
	// Replacement is the replacement pattern (can use captured $VARIABLES).
	Replacement *string `json:"replacement,omitempty"`
	// Transform is the structured transformation to apply.
	Transform string `json:"transform,omitempty"` // "add-import", "remove-import", "rename"
	// TargetName is required for remove-import and rename transforms.
	TargetName string `json:"targetName,omitempty"`
	// NewName is required for rename transform.
	NewName string `json:"newName,omitempty"`
	// ImportSpec is required for add-import transform.
	ImportSpec *ImportSpec `json:"importSpec,omitempty"`
}

// =============================================================================
// Language Detection
// =============================================================================

// astGrepLanguages maps file extensions to language identifiers.
var astGrepLanguages = map[string]string{
	"ts":   "TypeScript",
	"tsx":  "Tsx",
	"jsx":  "Tsx",
	"js":   "JavaScript",
	"html": "Html",
	"css":  "Css",
}

// GetLanguageFromPath returns the AST language for a file path.
func GetLanguageFromPath(filePath string) string {
	parts := strings.Split(filePath, ".")
	if len(parts) < 2 {
		return ""
	}
	ext := strings.ToLower(parts[len(parts)-1])
	return astGrepLanguages[ext]
}

// =============================================================================
// AST Grep Availability
// =============================================================================

// NOTE: In the TypeScript version, ast-grep/napi is an optional peer dependency
// loaded dynamically. In Go, there is no equivalent AST-grep library. This
// tool is ported as a stub that returns "not available" unless a Go AST-grep
// implementation is provided. The types and helper functions are preserved
// for completeness and future integration.

// astGrepAvailable tracks whether an AST grep backend is available.
var astGrepAvailable = false

// IsASTGrepAvailable checks if AST-grep is available.
func IsASTGrepAvailable() bool {
	return astGrepAvailable
}

// =============================================================================
// Execute AST Edit
// =============================================================================

// ExecuteASTEdit executes the ast_edit tool.
func ExecuteASTEdit(input *ASTEditInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	fs := result.Filesystem

	if fs.ReadOnly() {
		return "", fmt.Errorf("workspace is in read-only mode. Cannot perform: ast_edit")
	}

	// Check AST-grep availability
	if !IsASTGrepAvailable() {
		return "AST-grep is not available in Go. Use the edit_file tool for text-based edits.", nil
	}

	// Read current content
	raw, err := fs.ReadFile(input.Path, &ReadOptions{Encoding: "utf-8"})
	if err != nil {
		return fmt.Sprintf("File not found: %s. Use the write file tool to create it first.", input.Path), nil
	}

	content, ok := raw.(string)
	if !ok {
		return "Cannot perform AST edits on binary files. Use the write file tool instead.", nil
	}

	// Check language support
	lang := GetLanguageFromPath(input.Path)
	if lang == "" {
		return fmt.Sprintf("Unsupported file type for AST editing: %s", input.Path), nil
	}

	modifiedContent := content
	var changes []string

	if input.Transform != "" {
		switch input.Transform {
		case "add-import":
			if input.ImportSpec == nil {
				return "Error: importSpec is required for add-import transform", nil
			}
			changes = append(changes, fmt.Sprintf("Added import from '%s'", input.ImportSpec.Module))

		case "remove-import":
			if input.TargetName == "" {
				return "Error: targetName is required for remove-import transform", nil
			}
			changes = append(changes, fmt.Sprintf("Removed import '%s'", input.TargetName))

		case "rename":
			if input.TargetName == "" || input.NewName == "" {
				return "Error: targetName and newName are required for rename transform", nil
			}
			changes = append(changes, fmt.Sprintf("Renamed '%s' to '%s'", input.TargetName, input.NewName))
		}
	} else if input.Pattern != "" && input.Replacement != nil {
		changes = append(changes, "Replaced occurrences of pattern")
	} else if input.Pattern != "" && input.Replacement == nil {
		return "Error: replacement is required when pattern is provided", nil
	} else if input.Pattern == "" && input.Replacement != nil {
		return "Error: pattern is required when replacement is provided", nil
	} else {
		return "Error: Must provide either transform or pattern/replacement", nil
	}

	// Check if modified
	wasModified := modifiedContent != content
	if !wasModified {
		return fmt.Sprintf("No changes made to %s (%s)", input.Path, strings.Join(changes, "; ")), nil
	}

	overwrite := true
	err = fs.WriteFile(input.Path, modifiedContent, &WriteOptions{Overwrite: &overwrite})
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("%s: %s", input.Path, strings.Join(changes, "; "))
	output += GetEditDiagnosticsText(ws, input.Path, modifiedContent)
	return output, nil
}
