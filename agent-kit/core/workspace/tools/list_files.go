// Ported from: packages/core/src/workspace/tools/list-files.ts
package tools

import (
	"fmt"
)

// =============================================================================
// List Files Tool
// =============================================================================

// ListFilesInput holds the input for the list_files tool.
type ListFilesInput struct {
	// Path is the directory path to list (default: "./").
	Path string `json:"path,omitempty"`
	// MaxDepth is the maximum depth to descend (default: 2).
	MaxDepth int `json:"maxDepth,omitempty"`
	// ShowHidden shows hidden files starting with "." (default: false).
	ShowHidden bool `json:"showHidden,omitempty"`
	// DirsOnly lists directories only (default: false).
	DirsOnly bool `json:"dirsOnly,omitempty"`
	// Exclude is a pattern to exclude (e.g., "node_modules").
	Exclude string `json:"exclude,omitempty"`
	// Extension filters by file extension (e.g., ".ts").
	Extension string `json:"extension,omitempty"`
	// Pattern is one or more glob patterns to filter files.
	Pattern interface{} `json:"pattern,omitempty"`
	// RespectGitignore respects .gitignore (default: true).
	RespectGitignore *bool `json:"respectGitignore,omitempty"`
}

// ExecuteListFiles executes the list_files tool.
func ExecuteListFiles(input *ListFilesInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	fs := result.Filesystem

	path := input.Path
	if path == "" {
		path = "./"
	}

	maxDepth := input.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	respectGitignore := true
	if input.RespectGitignore != nil {
		respectGitignore = *input.RespectGitignore
	}
	_ = respectGitignore // Used for gitignore filter if available

	treeResult, err := FormatAsTree(fs, path, &TreeOptions{
		MaxDepth:   maxDepth,
		ShowHidden: input.ShowHidden,
		DirsOnly:   input.DirsOnly,
		Exclude:    input.Exclude,
		Extension:  input.Extension,
		Pattern:    input.Pattern,
	})
	if err != nil {
		return "", err
	}

	output := fmt.Sprintf("%s\n\n%s", treeResult.Tree, treeResult.Summary)

	// Get token limit from config (default: 1000)
	var tokenLimit *int
	defaultLimit := 1000
	tokenLimit = &defaultLimit
	toolsConfig := ws.GetToolsConfig()
	if toolsConfig != nil {
		tc := toolsConfig.GetToolConfig("mastra_workspace_list_files")
		if tc != nil && tc.MaxOutputTokens != nil {
			tokenLimit = tc.MaxOutputTokens
		}
	}

	return ApplyTokenLimit(output, tokenLimit, "end"), nil
}
