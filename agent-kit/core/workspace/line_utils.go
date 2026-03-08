// Ported from: packages/core/src/workspace/line-utils.ts
package workspace

import (
	"fmt"
	"strings"
)

// LineRange represents a line range where content was found.
type LineRange struct {
	// Start is the starting line number (1-indexed).
	Start int
	// End is the ending line number (1-indexed, inclusive).
	End int
}

// ExtractLinesResult holds the result of an ExtractLines call.
type ExtractLinesResult struct {
	// Content is the extracted text content.
	Content string
	// Lines contains the start and end line numbers of the extraction.
	Lines LineRange
	// TotalLines is the total number of lines in the original content.
	TotalLines int
}

// ExtractLines extracts lines from content by line range.
//
// startLine and endLine are 1-indexed. Pass 0 for startLine to default to 1,
// and 0 for endLine to default to the last line.
func ExtractLines(content string, startLine, endLine int) ExtractLinesResult {
	allLines := strings.Split(content, "\n")
	totalLines := len(allLines)

	// Default to full content
	start := startLine
	if start < 1 {
		start = 1
	}

	end := endLine
	if end <= 0 {
		end = totalLines
	}
	if end > totalLines {
		end = totalLines
	}

	// Extract the requested range (convert to 0-indexed)
	extractedLines := allLines[start-1 : end]

	return ExtractLinesResult{
		Content:    strings.Join(extractedLines, "\n"),
		Lines:      LineRange{Start: start, End: end},
		TotalLines: totalLines,
	}
}

// ExtractLinesWithLimit extracts lines using offset/limit style parameters (like Claude Code).
//
// offset is the line number to start from (1-indexed, default: 1).
// limit is the maximum number of lines to read (0 means all remaining).
func ExtractLinesWithLimit(content string, offset, limit int) ExtractLinesResult {
	startLine := offset
	if startLine < 1 {
		startLine = 1
	}

	endLine := 0
	if limit > 0 {
		endLine = startLine + limit - 1
	}

	return ExtractLines(content, startLine, endLine)
}

// FormatWithLineNumbers formats content with line number prefixes.
// Output format matches Claude Code: "     1->content here"
//
// startLineNumber is the line number of the first line (1-indexed).
func FormatWithLineNumbers(content string, startLineNumber int) string {
	if startLineNumber < 1 {
		startLineNumber = 1
	}

	lines := strings.Split(content, "\n")
	maxLineNum := startLineNumber + len(lines) - 1
	padWidth := len(fmt.Sprintf("%d", maxLineNum)) + 1
	if padWidth < 6 {
		padWidth = 6
	}

	var builder strings.Builder
	for i, line := range lines {
		if i > 0 {
			builder.WriteByte('\n')
		}
		lineNum := startLineNumber + i
		numStr := fmt.Sprintf("%d", lineNum)
		// Pad left
		for j := len(numStr); j < padWidth; j++ {
			builder.WriteByte(' ')
		}
		builder.WriteString(numStr)
		builder.WriteString("\u2192") // Unicode right arrow
		builder.WriteString(line)
	}

	return builder.String()
}

// CharIndexToLineNumber converts a character index to a line number.
// Useful for converting RAG chunk character offsets to line numbers.
//
// charIndex is 0-indexed. Returns the 1-indexed line number,
// or -1 if charIndex is out of bounds.
func CharIndexToLineNumber(content string, charIndex int) int {
	if charIndex < 0 || charIndex > len(content) {
		return -1
	}

	lineNumber := 1
	for i := 0; i < charIndex && i < len(content); i++ {
		if content[i] == '\n' {
			lineNumber++
		}
	}

	return lineNumber
}

// CharRangeToLineRange converts a character range to a line range.
// Useful for converting RAG chunk character offsets to line ranges.
//
// startCharIdx is 0-indexed. endCharIdx is 0-indexed and exclusive.
// Returns nil if indices are out of bounds.
func CharRangeToLineRange(content string, startCharIdx, endCharIdx int) *LineRange {
	startLine := CharIndexToLineNumber(content, startCharIdx)
	// For end, we want the line containing the last character (endCharIdx - 1)
	adjEnd := endCharIdx - 1
	if adjEnd < 0 {
		adjEnd = 0
	}
	endLine := CharIndexToLineNumber(content, adjEnd)

	if startLine == -1 || endLine == -1 {
		return nil
	}

	return &LineRange{Start: startLine, End: endLine}
}

// CountOccurrences counts occurrences of a string in content.
func CountOccurrences(content, searchString string) int {
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

// ReplaceStringResult holds the result of a ReplaceString call.
type ReplaceStringResult struct {
	// Content is the modified content.
	Content string
	// Replacements is the number of replacements made.
	Replacements int
}

// ReplaceString replaces a string in content, with validation for uniqueness.
//
// If replaceAll is false, the oldString must appear exactly once in content.
// Returns an error if oldString is not found or not unique (when replaceAll is false).
func ReplaceString(content, oldString, newString string, replaceAll bool) (ReplaceStringResult, error) {
	count := CountOccurrences(content, oldString)

	if count == 0 {
		return ReplaceStringResult{}, NewStringNotFoundError(oldString)
	}

	if !replaceAll && count > 1 {
		return ReplaceStringResult{}, NewStringNotUniqueError(oldString, count)
	}

	if replaceAll {
		result := strings.ReplaceAll(content, oldString, newString)
		return ReplaceStringResult{Content: result, Replacements: count}, nil
	}

	// Replace first (and only) occurrence
	result := strings.Replace(content, oldString, newString, 1)
	return ReplaceStringResult{Content: result, Replacements: 1}, nil
}

// StringNotFoundError is returned when a string is not found during replacement.
type StringNotFoundError struct {
	SearchString string
}

func (e *StringNotFoundError) Error() string {
	return "The specified text was not found. Make sure you use the exact text from the file."
}

// NewStringNotFoundError creates a new StringNotFoundError.
func NewStringNotFoundError(searchString string) *StringNotFoundError {
	return &StringNotFoundError{SearchString: searchString}
}

// StringNotUniqueError is returned when a string appears multiple times
// but a unique match was required.
type StringNotUniqueError struct {
	SearchString string
	Occurrences  int
}

func (e *StringNotUniqueError) Error() string {
	return fmt.Sprintf(
		"The specified text appears %d times. Provide more surrounding context to make the match unique, or use replace_all to replace all occurrences.",
		e.Occurrences,
	)
}

// NewStringNotUniqueError creates a new StringNotUniqueError.
func NewStringNotUniqueError(searchString string, occurrences int) *StringNotUniqueError {
	return &StringNotUniqueError{
		SearchString: searchString,
		Occurrences:  occurrences,
	}
}
