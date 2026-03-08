// Ported from: packages/core/src/workspace/tools/grep.ts
package tools

import (
	"fmt"
	"regexp"
	"strings"
)

// =============================================================================
// Grep Tool
// =============================================================================

// GrepInput holds the input for the grep tool.
type GrepInput struct {
	// Pattern is the regex pattern to search for.
	Pattern string `json:"pattern"`
	// Path is the file, directory, or glob pattern to search within (default: "./").
	Path string `json:"path,omitempty"`
	// ContextLines is the number of lines of context before and after each match (default: 0).
	ContextLines int `json:"contextLines,omitempty"`
	// MaxCount is the maximum matches per file.
	MaxCount *int `json:"maxCount,omitempty"`
	// CaseSensitive controls case sensitivity (default: true).
	CaseSensitive *bool `json:"caseSensitive,omitempty"`
	// IncludeHidden includes hidden files/directories (default: false).
	IncludeHidden bool `json:"includeHidden,omitempty"`
}

// ExecuteGrep executes the grep tool.
func ExecuteGrep(input *GrepInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	fs := result.Filesystem

	inputPath := input.Path
	if inputPath == "" {
		inputPath = "./"
	}

	caseSensitive := true
	if input.CaseSensitive != nil {
		caseSensitive = *input.CaseSensitive
	}

	// Guard against excessively long patterns
	const maxPatternLength = 1000
	if len(input.Pattern) > maxPatternLength {
		return fmt.Sprintf("Error: Pattern too long (%d chars, max %d). Use a shorter pattern.",
			len(input.Pattern), maxPatternLength), nil
	}

	// Compile regex
	flags := ""
	if !caseSensitive {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + input.Pattern)
	if err != nil {
		return fmt.Sprintf("Error: Invalid regex pattern: %s", err.Error()), nil
	}

	// Collect files to search
	var filePaths []string

	stat, statErr := fs.Stat(inputPath)
	if statErr != nil {
		// Path doesn't exist
		filePaths = []string{}
	} else if stat.Type == "file" {
		filePaths = []string{inputPath}
	} else {
		// Walk directory recursively
		filePaths = collectFilesForGrep(fs, inputPath, input.IncludeHidden)
	}

	var outputLines []string
	filesWithMatches := make(map[string]bool)
	totalMatchCount := 0
	truncated := false
	const maxLineLength = 500
	const globalCap = 1000

	for _, filePath := range filePaths {
		if truncated {
			break
		}

		raw, readErr := fs.ReadFile(filePath, &ReadOptions{Encoding: "utf-8"})
		if readErr != nil {
			continue
		}
		content, ok := raw.(string)
		if !ok {
			continue
		}

		lines := strings.Split(content, "\n")
		fileMatchCount := 0

		for i, currentLine := range lines {
			loc := re.FindStringIndex(currentLine)
			if loc == nil {
				continue
			}

			filesWithMatches[filePath] = true

			lineContent := currentLine
			if len(lineContent) > maxLineLength {
				lineContent = lineContent[:maxLineLength] + "..."
			}

			// Context lines before the match
			if input.ContextLines > 0 {
				beforeStart := i - input.ContextLines
				if beforeStart < 0 {
					beforeStart = 0
				}
				for b := beforeStart; b < i; b++ {
					outputLines = append(outputLines, fmt.Sprintf("%s:%d- %s", filePath, b+1, lines[b]))
				}
			}

			// The matching line
			outputLines = append(outputLines, fmt.Sprintf("%s:%d:%d: %s",
				filePath, i+1, loc[0]+1, lineContent))

			// Context lines after the match
			if input.ContextLines > 0 {
				afterEnd := i + input.ContextLines
				if afterEnd >= len(lines) {
					afterEnd = len(lines) - 1
				}
				for a := i + 1; a <= afterEnd; a++ {
					outputLines = append(outputLines, fmt.Sprintf("%s:%d- %s", filePath, a+1, lines[a]))
				}
				outputLines = append(outputLines, "--")
			}

			totalMatchCount++
			fileMatchCount++

			// Per-file limit
			if input.MaxCount != nil && fileMatchCount >= *input.MaxCount {
				break
			}

			// Global cap
			if totalMatchCount >= globalCap {
				truncated = true
				break
			}
		}
	}

	// Summary line at the top
	matchWord := "matches"
	if totalMatchCount == 1 {
		matchWord = "match"
	}
	fileWord := "files"
	if len(filesWithMatches) == 1 {
		fileWord = "file"
	}
	summary := fmt.Sprintf("%d %s across %d %s", totalMatchCount, matchWord, len(filesWithMatches), fileWord)
	if truncated {
		summary += fmt.Sprintf(" (truncated at %d)", globalCap)
	}

	// Insert summary at the top
	final := append([]string{summary, "---"}, outputLines...)

	// Get token limit from config
	var tokenLimit *int
	toolsConfig := ws.GetToolsConfig()
	if toolsConfig != nil {
		tc := toolsConfig.GetToolConfig("mastra_workspace_grep")
		if tc != nil {
			tokenLimit = tc.MaxOutputTokens
		}
	}

	return ApplyTokenLimit(strings.Join(final, "\n"), tokenLimit, "end"), nil
}

// collectFilesForGrep recursively collects text files from a directory.
func collectFilesForGrep(fs FilesystemAccessor, dir string, includeHidden bool) []string {
	entries, err := fs.Readdir(dir, nil)
	if err != nil {
		return nil
	}

	var files []string
	for _, entry := range entries {
		// Skip hidden files/dirs unless includeHidden
		if !includeHidden && strings.HasPrefix(entry.Name, ".") {
			continue
		}

		fullPath := joinPath(dir, entry.Name)

		if entry.Type == "file" {
			// Skip non-text files
			if !isTextFileByName(entry.Name) {
				continue
			}
			files = append(files, fullPath)
		} else if entry.Type == "directory" && !entry.IsSymlink {
			files = append(files, collectFilesForGrep(fs, fullPath, includeHidden)...)
		}
	}

	return files
}

// isTextFileByName checks if a file is likely text based on its name/extension.
// Simplified version matching the TS isTextFile function.
func isTextFileByName(filename string) bool {
	// Common text file extensions
	textExts := map[string]bool{
		".txt": true, ".md": true, ".json": true, ".yaml": true, ".yml": true,
		".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".mjs": true, ".cjs": true,
		".py": true, ".rb": true, ".go": true, ".rs": true, ".java": true,
		".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".cs": true, ".swift": true, ".kt": true,
		".php": true, ".sh": true, ".bash": true,
		".html": true, ".htm": true, ".css": true, ".scss": true,
		".xml": true, ".svg": true, ".toml": true, ".ini": true, ".cfg": true,
		".sql": true, ".graphql": true, ".proto": true,
		".vue": true, ".svelte": true, ".astro": true,
		".env": true, ".csv": true, ".log": true,
		".lock": true, ".diff": true, ".patch": true,
		".mdx": true, ".rst": true, ".tex": true,
	}

	ext := strings.ToLower(getFileExt(filename))
	if ext == "" {
		return true // No extension = treat as text
	}
	return textExts[ext]
}

// getFileExt returns the file extension including the dot.
func getFileExt(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return filename[i:]
		}
		if filename[i] == '/' || filename[i] == '\\' {
			break
		}
	}
	return ""
}
