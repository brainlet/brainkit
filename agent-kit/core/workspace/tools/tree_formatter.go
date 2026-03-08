// Ported from: packages/core/src/workspace/tools/tree-formatter.ts
package tools

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// =============================================================================
// Types
// =============================================================================

// TreeOptions configures tree formatting behavior.
type TreeOptions struct {
	// MaxDepth is the maximum depth to descend (default: 2).
	MaxDepth int
	// ShowHidden shows hidden files starting with "." (default: false).
	ShowHidden bool
	// DirsOnly lists directories only, no files (default: false).
	DirsOnly bool
	// Exclude is a pattern to exclude (similar to tree -I flag).
	Exclude string
	// Extension filters by file extension (e.g., ".ts").
	Extension string
	// Pattern is one or more glob patterns to filter files.
	Pattern interface{} // string or []string
	// RespectGitignore respects .gitignore patterns (default: true).
	RespectGitignore bool
	// IgnoreFilter is a gitignore-style filter function.
	IgnoreFilter func(path string) bool
}

// TreeResult holds the result of a tree formatting operation.
type TreeResult struct {
	// Tree is the formatted tree string.
	Tree string
	// Summary is a human-readable summary of the tree contents.
	Summary string
	// Stats holds counts of directories and files.
	Stats TreeStats
}

// TreeStats holds statistics about the tree.
type TreeStats struct {
	// Dirs is the number of directories found.
	Dirs int
	// Files is the number of files found.
	Files int
	// TruncatedDirs is the number of directories that hit the depth limit.
	TruncatedDirs int
}

// =============================================================================
// Glob Matching (simple)
// =============================================================================

// simpleGlobMatch performs basic glob matching against a path.
// Supports: *, **, ?, {a,b,c}, and character classes [abc].
func simpleGlobMatch(pattern, name string) bool {
	matched, _ := filepath.Match(pattern, name)
	if matched {
		return true
	}
	// For ** patterns, try matching against the basename
	if strings.Contains(pattern, "**") {
		// Strip ** prefix and try matching the rest against the basename
		rest := strings.TrimPrefix(pattern, "**/")
		if rest != pattern {
			base := filepath.Base(name)
			matched, _ = filepath.Match(rest, base)
			if matched {
				return true
			}
			// Also try matching against the full path
			matched, _ = filepath.Match(rest, name)
			return matched
		}
	}
	return false
}

// matchesAnyPattern checks if a name matches any of the given glob patterns.
func matchesAnyPattern(name string, patterns []string) bool {
	for _, p := range patterns {
		if simpleGlobMatch(p, name) {
			return true
		}
	}
	return false
}

// =============================================================================
// Tree Formatter
// =============================================================================

// FormatAsTree walks a filesystem directory and produces a tree-style listing.
func FormatAsTree(fs FilesystemAccessor, rootPath string, opts *TreeOptions) (*TreeResult, error) {
	if opts == nil {
		opts = &TreeOptions{}
	}

	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 2
	}

	// Parse glob patterns
	var patterns []string
	if opts.Pattern != nil {
		switch v := opts.Pattern.(type) {
		case string:
			if v != "" {
				patterns = append(patterns, v)
			}
		case []string:
			patterns = v
		}
	}

	stats := &TreeStats{}
	lines := []string{rootPath}

	err := formatTreeRecursive(fs, rootPath, "", 0, maxDepth, opts, patterns, stats, &lines)
	if err != nil {
		return nil, err
	}

	// Build summary
	summaryParts := []string{
		fmt.Sprintf("%d director%s", stats.Dirs, pluralize(stats.Dirs, "y", "ies")),
	}
	if !opts.DirsOnly {
		summaryParts = append(summaryParts, fmt.Sprintf("%d file%s", stats.Files, pluralize(stats.Files, "", "s")))
	}
	if stats.TruncatedDirs > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d at depth limit", stats.TruncatedDirs))
	}

	return &TreeResult{
		Tree:    strings.Join(lines, "\n"),
		Summary: strings.Join(summaryParts, ", "),
		Stats:   *stats,
	}, nil
}

func formatTreeRecursive(
	fs FilesystemAccessor,
	dirPath string,
	prefix string,
	depth int,
	maxDepth int,
	opts *TreeOptions,
	patterns []string,
	stats *TreeStats,
	lines *[]string,
) error {
	if depth >= maxDepth {
		stats.TruncatedDirs++
		return nil
	}

	entries, err := fs.Readdir(dirPath, nil)
	if err != nil {
		return nil // Silently skip unreadable directories
	}

	// Filter entries
	var filtered []FileEntry
	for _, entry := range entries {
		// Skip hidden files unless showHidden
		if !opts.ShowHidden && strings.HasPrefix(entry.Name, ".") {
			continue
		}

		// Skip excluded pattern
		if opts.Exclude != "" && matchesExclude(entry.Name, opts.Exclude) {
			continue
		}

		// Apply gitignore filter
		if opts.IgnoreFilter != nil {
			relativePath := joinPath(dirPath, entry.Name)
			checkPath := relativePath
			if entry.Type == "directory" {
				checkPath = relativePath + "/"
			}
			if opts.IgnoreFilter(checkPath) {
				continue
			}
		}

		// DirsOnly mode — skip files
		if opts.DirsOnly && entry.Type != "directory" {
			continue
		}

		// Extension filter (files only)
		if opts.Extension != "" && entry.Type == "file" {
			ext := filepath.Ext(entry.Name)
			if !strings.EqualFold(ext, opts.Extension) {
				continue
			}
		}

		// Glob pattern filter (files only; directories always pass through)
		if len(patterns) > 0 && entry.Type == "file" {
			fullPath := joinPath(dirPath, entry.Name)
			if !matchesAnyPattern(fullPath, patterns) && !matchesAnyPattern(entry.Name, patterns) {
				continue
			}
		}

		filtered = append(filtered, entry)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Type != filtered[j].Type {
			if filtered[i].Type == "directory" {
				return true
			}
			if filtered[j].Type == "directory" {
				return false
			}
		}
		return strings.ToLower(filtered[i].Name) < strings.ToLower(filtered[j].Name)
	})

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "\u251c\u2500\u2500 " // ├──
		if isLast {
			connector = "\u2514\u2500\u2500 " // └──
		}

		displayName := entry.Name
		if entry.Type == "directory" {
			displayName += "/"
			stats.Dirs++
		} else {
			stats.Files++
		}

		*lines = append(*lines, prefix+connector+displayName)

		// Recurse into directories
		if entry.Type == "directory" && !entry.IsSymlink {
			childPrefix := prefix + "\u2502   " // │
			if isLast {
				childPrefix = prefix + "    "
			}
			childPath := joinPath(dirPath, entry.Name)
			_ = formatTreeRecursive(fs, childPath, childPrefix, depth+1, maxDepth, opts, patterns, stats, lines)
		}
	}

	return nil
}

// matchesExclude checks if a name matches an exclude pattern.
// Supports simple substring matching and basic glob.
func matchesExclude(name, pattern string) bool {
	// Simple substring match
	if strings.Contains(name, pattern) {
		return true
	}
	// Try glob match
	matched, _ := filepath.Match(pattern, name)
	return matched
}

// joinPath joins path components, handling the "./" root.
func joinPath(dir, name string) string {
	if dir == "./" || dir == "." {
		return name
	}
	dir = strings.TrimRight(dir, "/")
	return dir + "/" + name
}

// pluralize returns singular or plural suffix based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

// FormatEntriesAsTree formats a flat list of file entries as a tree.
// This is a convenience function for when entries have already been collected.
func FormatEntriesAsTree(entries []FileEntry, rootName string) string {
	if rootName == "" {
		rootName = "."
	}

	var lines []string
	lines = append(lines, rootName)

	for i, entry := range entries {
		isLast := i == len(entries)-1
		connector := "\u251c\u2500\u2500 " // ├──
		if isLast {
			connector = "\u2514\u2500\u2500 " // └──
		}

		displayName := entry.Name
		if entry.Type == "directory" {
			displayName += "/"
		}

		lines = append(lines, connector+displayName)
	}

	return strings.Join(lines, "\n")
}
