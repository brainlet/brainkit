// Ported from: packages/core/src/workspace/glob.ts
package workspace

import (
	"path/filepath"
	"regexp"
	"strings"
)

// =============================================================================
// Glob Metacharacter Detection
// =============================================================================

// globCharsRe matches characters that indicate a glob pattern (not a plain path).
var globCharsRe = regexp.MustCompile(`[*?{}\[\]]`)

// IsGlobPattern checks if a string contains glob metacharacters.
//
// Examples:
//
//	IsGlobPattern("/docs")          // false
//	IsGlobPattern("/docs/**/*.md")  // true
//	IsGlobPattern("*.ts")           // true
//	IsGlobPattern("/src/{a,b}")     // true
func IsGlobPattern(input string) bool {
	return globCharsRe.MatchString(input)
}

// =============================================================================
// Glob Base Extraction
// =============================================================================

// ExtractGlobBase extracts the static directory prefix before the first glob
// metacharacter. Returns the deepest non-glob ancestor directory.
//
// Examples:
//
//	ExtractGlobBase("/docs/**/*.md")  // "/docs"
//	ExtractGlobBase("**/*.md")        // "/"
//	ExtractGlobBase("/src/*.ts")      // "/src"
//	ExtractGlobBase("/exact/path")    // "/exact/path"
func ExtractGlobBase(pattern string) string {
	// Find position of first glob metacharacter
	loc := globCharsRe.FindStringIndex(pattern)

	if loc == nil {
		// No glob chars — return the pattern as-is (it's a plain path)
		return pattern
	}

	firstMeta := loc[0]

	// Get the portion before the first metacharacter
	prefix := pattern[:firstMeta]

	// Walk back to the last directory separator
	lastSlash := strings.LastIndex(prefix, "/")

	if lastSlash <= 0 {
		// No slash or only root slash — base is root
		return "/"
	}

	return prefix[:lastSlash]
}

// =============================================================================
// Glob Matcher
// =============================================================================

// GlobMatcher is a compiled matcher function: returns true if a path matches.
type GlobMatcher func(path string) bool

// GlobMatcherOptions configures glob matching behavior.
type GlobMatcherOptions struct {
	// Dot enables matching dotfiles (default: false).
	Dot bool
}

// normalizeForMatch strips leading './' or '/' from a path for matching.
// filepath.Match does not match paths with these prefixes, so both
// patterns and test paths must be normalized before matching.
//
// This only affects matching — filesystem paths should keep their
// original form for correct resolution with contained/uncontained modes.
func normalizeForMatch(input string) string {
	if strings.HasPrefix(input, "./") {
		return input[2:]
	}
	if strings.HasPrefix(input, "/") {
		return input[1:]
	}
	return input
}

// CreateGlobMatcher compiles glob pattern(s) into a reusable matcher function.
// The matcher tests paths using workspace-style forward slashes.
//
// Automatically normalizes leading './' and '/' from both patterns
// and test paths.
//
// NOTE: The TypeScript version uses picomatch for full glob support
// (brace expansion, character classes, negation, **). This Go
// implementation uses filepath.Match for basic glob support and
// handles ** patterns manually.
// TODO: Replace with a full glob library (e.g., doublestar) for complete picomatch parity.
//
// Examples:
//
//	match := CreateGlobMatcher([]string{"**/*.ts"}, nil)
//	match("src/index.ts")  // true
//	match("src/style.css") // false
func CreateGlobMatcher(patterns []string, options *GlobMatcherOptions) GlobMatcher {
	dot := false
	if options != nil {
		dot = options.Dot
	}

	normalizedPatterns := make([]string, len(patterns))
	for i, p := range patterns {
		normalizedPatterns[i] = normalizeForMatch(p)
	}

	return func(path string) bool {
		normalized := normalizeForMatch(path)

		// Skip dotfiles unless dot option is set
		if !dot {
			parts := strings.Split(normalized, "/")
			for _, part := range parts {
				if strings.HasPrefix(part, ".") && part != "." && part != ".." {
					return false
				}
			}
		}

		for _, pattern := range normalizedPatterns {
			if matchDoublestar(normalized, pattern) {
				return true
			}
		}
		return false
	}
}

// CreateGlobMatcherSingle is a convenience for creating a matcher from a single pattern.
func CreateGlobMatcherSingle(pattern string, options *GlobMatcherOptions) GlobMatcher {
	return CreateGlobMatcher([]string{pattern}, options)
}

// MatchGlob is a one-off convenience: test if a path matches a glob pattern.
//
// For repeated matching against the same pattern, prefer CreateGlobMatcher()
// to compile once and reuse.
//
// Examples:
//
//	MatchGlob("src/index.ts", []string{"**/*.ts"}, nil)  // true
func MatchGlob(path string, patterns []string, options *GlobMatcherOptions) bool {
	return CreateGlobMatcher(patterns, options)(path)
}

// matchDoublestar matches a path against a pattern that may contain ** segments.
func matchDoublestar(path, pattern string) bool {
	// Handle ** at the start: **/rest
	if strings.HasPrefix(pattern, "**/") {
		rest := pattern[3:]
		// Try matching rest against every suffix of path
		parts := strings.Split(path, "/")
		for i := 0; i <= len(parts); i++ {
			sub := strings.Join(parts[i:], "/")
			if matchDoublestar(sub, rest) {
				return true
			}
		}
		return false
	}

	// Handle ** at the end: prefix/**
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		normalizedPath := normalizeForMatch(path)
		normalizedPrefix := normalizeForMatch(prefix)
		return normalizedPath == normalizedPrefix || strings.HasPrefix(normalizedPath, normalizedPrefix+"/")
	}

	// Handle ** in the middle: left/**/right
	// The ** matches zero or more directory segments between left and right.
	if idx := strings.Index(pattern, "/**/"); idx >= 0 {
		left := pattern[:idx]
		right := pattern[idx+4:]
		parts := strings.Split(path, "/")
		// The left part must match the beginning of the path.
		// Try each split point where left matches the prefix.
		for i := 0; i <= len(parts); i++ {
			leftPath := strings.Join(parts[:i], "/")
			if !matchDoublestar(leftPath, left) {
				continue
			}
			// The ** can consume zero or more segments after left.
			// Try matching right against each remaining suffix.
			for j := i; j <= len(parts); j++ {
				rightPath := strings.Join(parts[j:], "/")
				if matchDoublestar(rightPath, right) {
					return true
				}
			}
		}
		return false
	}

	// Handle pattern that is just **
	if pattern == "**" {
		return true
	}

	// No ** — use filepath.Match for standard glob
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false
	}
	return matched
}

// =============================================================================
// Path Pattern Resolution
// =============================================================================

// PathEntry is a filesystem entry returned by ResolvePathPattern.
type PathEntry struct {
	Path string
	Type string // "file" or "directory"
}

// ReaddirEntry is a minimal readdir entry — compatible with both FileEntry and SkillSourceEntry.
type ReaddirEntry struct {
	Name      string
	Type      string // "file" or "directory"
	IsSymlink bool
}

// ResolvePathOptions configures path resolution behavior.
type ResolvePathOptions struct {
	// Dot enables matching dotfiles (default: false).
	Dot bool
	// MaxDepth is the maximum directory depth to walk (default: 10).
	MaxDepth int
}

// ReaddirFunc is a function that reads a directory and returns its entries.
type ReaddirFunc func(dir string) ([]ReaddirEntry, error)

// walkAll walks a directory tree recursively, returning all entries
// (files and directories). Skips symlinked directories to prevent infinite loops.
func walkAll(readdir ReaddirFunc, dir string, depth, maxDepth int) []PathEntry {
	if depth >= maxDepth {
		return nil
	}

	entries, err := readdir(dir)
	if err != nil {
		return nil
	}

	var results []PathEntry
	for _, entry := range entries {
		if entry.Type == "directory" && entry.IsSymlink {
			continue
		}
		var fullPath string
		if dir == "/" {
			fullPath = "/" + entry.Name
		} else {
			fullPath = dir + "/" + entry.Name
		}
		results = append(results, PathEntry{Path: fullPath, Type: entry.Type})
		if entry.Type == "directory" {
			results = append(results, walkAll(readdir, fullPath, depth+1, maxDepth)...)
		}
	}
	return results
}

// ResolvePathPattern resolves a path pattern to matching filesystem entries.
//
// Handles both plain paths and glob patterns consistently:
//   - Plain paths: determines file vs directory via readdir probe, returns single entry
//   - Glob patterns: walks from the glob base, matches both files and directories
//
// Examples:
//
//	ResolvePathPattern("/docs", readdir, nil)            // [{Path: "/docs", Type: "directory"}]
//	ResolvePathPattern("/docs/readme.md", readdir, nil)  // [{Path: "/docs/readme.md", Type: "file"}]
//	ResolvePathPattern("/docs/**/*.md", readdir, nil)    // all .md files under /docs
func ResolvePathPattern(pattern string, readdir ReaddirFunc, options *ResolvePathOptions) ([]PathEntry, error) {
	maxDepth := 10
	dot := false
	if options != nil {
		if options.MaxDepth > 0 {
			maxDepth = options.MaxDepth
		}
		dot = options.Dot
	}

	// Strip trailing slash for consistent path handling (e.g. '/skills/' -> '/skills')
	normalized := pattern
	if len(normalized) > 1 && strings.HasSuffix(normalized, "/") {
		normalized = normalized[:len(normalized)-1]
	}

	if !IsGlobPattern(normalized) {
		// Plain path — probe with readdir to determine if it's a directory or file
		_, err := readdir(normalized)
		if err == nil {
			return []PathEntry{{Path: normalized, Type: "directory"}}, nil
		}
		// readdir failed — treat as a file path (consumer handles non-existence)
		return []PathEntry{{Path: normalized, Type: "file"}}, nil
	}

	// Glob pattern — walk from base, match all entries (files and directories)
	walkRoot := ExtractGlobBase(normalized)
	matcher := CreateGlobMatcher([]string{normalized}, &GlobMatcherOptions{Dot: dot})
	allEntries := walkAll(readdir, walkRoot, 0, maxDepth)

	var results []PathEntry
	for _, entry := range allEntries {
		if matcher(entry.Path) {
			results = append(results, entry)
		}
	}
	return results, nil
}
