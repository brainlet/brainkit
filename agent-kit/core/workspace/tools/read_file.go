// Ported from: packages/core/src/workspace/tools/read-file.ts
package tools

import (
	"fmt"
)

// =============================================================================
// Read File Tool
// =============================================================================

// ReadFileInput holds the input for the read_file tool.
type ReadFileInput struct {
	// Path is the path to the file to read.
	Path string `json:"path"`
	// Encoding is the encoding to use (default: "utf-8").
	Encoding string `json:"encoding,omitempty"`
	// Offset is the line number to start reading from (1-indexed).
	Offset *int `json:"offset,omitempty"`
	// Limit is the maximum number of lines to read.
	Limit *int `json:"limit,omitempty"`
	// ShowLineNumbers controls whether to prefix each line with its line number (default: true).
	ShowLineNumbers *bool `json:"showLineNumbers,omitempty"`
}

// ExecuteReadFile executes the read_file tool.
func ExecuteReadFile(input *ReadFileInput, ctx *ToolContext) (string, error) {
	result, err := RequireFilesystem(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	fs := result.Filesystem

	effectiveEncoding := input.Encoding
	if effectiveEncoding == "" {
		effectiveEncoding = "utf-8"
	}

	fullContent, err := fs.ReadFile(input.Path, &ReadOptions{Encoding: effectiveEncoding})
	if err != nil {
		return "", err
	}

	stat, err := fs.Stat(input.Path)
	if err != nil {
		return "", err
	}

	isTextEncoding := input.Encoding == "" || input.Encoding == "utf-8" || input.Encoding == "utf8"

	// Get token limit from config
	var tokenLimit *int
	toolsConfig := ws.GetToolsConfig()
	if toolsConfig != nil {
		tc := toolsConfig.GetToolConfig("mastra_workspace_read_file")
		if tc != nil {
			tokenLimit = tc.MaxOutputTokens
		}
	}

	// Non-text encoding — return raw content
	if !isTextEncoding {
		contentStr := fmt.Sprintf("%v", fullContent)
		header := fmt.Sprintf("%s (%d bytes, %s)\n%s", stat.Path, stat.Size, effectiveEncoding, contentStr)
		return ApplyTokenLimit(header, tokenLimit, "end"), nil
	}

	// Content must be a string for text processing
	contentStr, ok := fullContent.(string)
	if !ok {
		// Binary content — return as base64 description
		header := fmt.Sprintf("%s (%d bytes, binary)", stat.Path, stat.Size)
		return ApplyTokenLimit(header, tokenLimit, "end"), nil
	}

	// Apply line range extraction
	hasLineRange := input.Offset != nil || input.Limit != nil
	offset := 0
	limit := 0
	if input.Offset != nil {
		offset = *input.Offset
	}
	if input.Limit != nil {
		limit = *input.Limit
	}

	extractResult := ExtractLinesWithLimit(contentStr, offset, limit)

	// Format with line numbers
	showLineNumbers := true
	if input.ShowLineNumbers != nil {
		showLineNumbers = *input.ShowLineNumbers
	}

	formattedContent := extractResult.Content
	if showLineNumbers {
		formattedContent = FormatWithLineNumbers(extractResult.Content, extractResult.Lines.Start)
	}

	// Build header
	var header string
	if hasLineRange {
		header = fmt.Sprintf("%s (lines %d-%d of %d, %d bytes)",
			stat.Path, extractResult.Lines.Start, extractResult.Lines.End,
			extractResult.TotalLines, stat.Size)
	} else {
		header = fmt.Sprintf("%s (%d bytes)", stat.Path, stat.Size)
	}

	return ApplyTokenLimit(header+"\n"+formattedContent, tokenLimit, "end"), nil
}

// =============================================================================
// Line Utilities (inlined from workspace/line_utils.go for package isolation)
// =============================================================================

// ExtractLinesResult holds the result of line extraction.
type ExtractLinesResult struct {
	Content    string
	Lines      ExtractedLineRange
	TotalLines int
}

// ExtractedLineRange holds the start and end line numbers.
type ExtractedLineRange struct {
	Start int
	End   int
}

// ExtractLinesWithLimit extracts lines using offset/limit style parameters.
func ExtractLinesWithLimit(content string, offset, limit int) ExtractLinesResult {
	return ExtractLines(content, offset, limit)
}

// ExtractLines extracts lines from content by offset and limit.
func ExtractLines(content string, offset, limit int) ExtractLinesResult {
	lines := splitLines(content)
	totalLines := len(lines)

	start := offset
	if start < 1 {
		start = 1
	}

	end := totalLines
	if limit > 0 {
		end = start + limit - 1
	}
	if end > totalLines {
		end = totalLines
	}

	extracted := lines[start-1 : end]

	return ExtractLinesResult{
		Content:    joinLines(extracted),
		Lines:      ExtractedLineRange{Start: start, End: end},
		TotalLines: totalLines,
	}
}

// FormatWithLineNumbers formats content with line number prefixes.
func FormatWithLineNumbers(content string, startLineNumber int) string {
	if startLineNumber < 1 {
		startLineNumber = 1
	}

	lines := splitLines(content)
	maxLineNum := startLineNumber + len(lines) - 1
	padWidth := len(fmt.Sprintf("%d", maxLineNum)) + 1
	if padWidth < 6 {
		padWidth = 6
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		lineNum := startLineNumber + i
		numStr := fmt.Sprintf("%d", lineNum)
		padding := ""
		for j := len(numStr); j < padWidth; j++ {
			padding += " "
		}
		result[i] = padding + numStr + "\u2192" + line
	}

	return joinLines(result)
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	// Use manual split to match JS behavior exactly
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
