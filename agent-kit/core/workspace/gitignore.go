// Ported from: packages/core/src/workspace/gitignore.ts
package workspace

import (
	"strings"
)

// IgnoreFilter is a function that takes a path relative to the workspace root
// and returns true if the path is ignored (should be skipped).
type IgnoreFilter func(relativePath string) bool

// LoadGitignore loads .gitignore from the workspace root and returns a filter function.
//
// The returned function takes a path relative to the workspace root and
// returns true if the path is ignored (should be skipped).
//
// Returns nil if no .gitignore exists or it can't be read.
//
// NOTE: The TypeScript version uses the `ignore` npm package for full
// .gitignore specification compliance. This Go implementation provides
// a simplified pattern matching that covers common cases but does not
// implement the full .gitignore specification (negation patterns, etc.).
// TODO: Replace with a full gitignore implementation (e.g., go-gitignore) for complete spec compliance.
func LoadGitignore(filesystem WorkspaceFilesystem) (IgnoreFilter, error) {
	raw, err := filesystem.ReadFile(".gitignore", nil)
	if err != nil {
		return nil, nil // No .gitignore or can't read — return nil filter (not an error)
	}

	content, ok := raw.(string)
	if !ok || strings.TrimSpace(content) == "" {
		return nil, nil
	}

	patterns := parseGitignorePatterns(content)
	if len(patterns) == 0 {
		return nil, nil
	}

	return func(relativePath string) bool {
		// The `ignore` package expects paths without leading './' or '/'
		normalized := strings.TrimPrefix(relativePath, "./")
		normalized = strings.TrimPrefix(normalized, "/")
		if normalized == "" {
			return false
		}
		return matchesGitignore(normalized, patterns)
	}, nil
}

// gitignorePattern represents a parsed gitignore pattern.
type gitignorePattern struct {
	pattern    string
	isNegation bool
	isDir      bool
}

// parseGitignorePatterns parses .gitignore content into patterns.
func parseGitignorePatterns(content string) []gitignorePattern {
	lines := strings.Split(content, "\n")
	var patterns []gitignorePattern

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		p := gitignorePattern{}

		// Handle negation
		if strings.HasPrefix(line, "!") {
			p.isNegation = true
			line = line[1:]
		}

		// Handle directory-only patterns
		if strings.HasSuffix(line, "/") {
			p.isDir = true
			line = strings.TrimSuffix(line, "/")
		}

		p.pattern = line
		patterns = append(patterns, p)
	}

	return patterns
}

// matchesGitignore checks if a path matches any gitignore pattern.
// This is a simplified implementation that covers common patterns.
func matchesGitignore(path string, patterns []gitignorePattern) bool {
	ignored := false

	for _, p := range patterns {
		if matchesPattern(path, p.pattern) {
			if p.isNegation {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

// matchesPattern checks if a path matches a single gitignore pattern.
// Supports: *, **, ?, and plain string prefix/suffix matching.
func matchesPattern(path, pattern string) bool {
	// Handle ** patterns
	if pattern == "**" {
		return true
	}

	// Handle **/name patterns (match anywhere)
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// Match at root level or any subdirectory
		if matchSimple(path, suffix) {
			return true
		}
		parts := strings.Split(path, "/")
		for i := range parts {
			subpath := strings.Join(parts[i:], "/")
			if matchSimple(subpath, suffix) {
				return true
			}
		}
		return false
	}

	// Handle name/** patterns (match everything under)
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}

	// Handle patterns without slashes (match basename anywhere)
	if !strings.Contains(pattern, "/") {
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if matchSimple(part, pattern) {
				return true
			}
		}
		return false
	}

	// Handle patterns with slashes (match from root)
	pattern = strings.TrimPrefix(pattern, "/")
	return matchSimple(path, pattern)
}

// matchSimple matches a path against a simple glob pattern (supports * and ?).
func matchSimple(path, pattern string) bool {
	return matchGlobSimple(path, pattern)
}

// matchGlobSimple implements simple glob matching with * and ? support.
func matchGlobSimple(str, pattern string) bool {
	si, pi := 0, 0
	starSi, starPi := -1, -1

	for si < len(str) {
		if pi < len(pattern) && (pattern[pi] == '?' || pattern[pi] == str[si]) {
			si++
			pi++
		} else if pi < len(pattern) && pattern[pi] == '*' {
			starSi = si
			starPi = pi
			pi++
		} else if starPi >= 0 {
			starSi++
			si = starSi
			pi = starPi + 1
		} else {
			return false
		}
	}

	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}

	return pi == len(pattern)
}
