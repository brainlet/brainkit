// Ported from: packages/core/src/workspace/gitignore.ts
package workspace

import (
	"strings"

	"github.com/brainlet/brainkit/ignore"
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
// Uses the ported node-ignore library for full .gitignore specification
// compliance including negation patterns, directory-only patterns, and ** syntax.
func LoadGitignore(filesystem WorkspaceFilesystem) (IgnoreFilter, error) {
	raw, err := filesystem.ReadFile(".gitignore", nil)
	if err != nil {
		return nil, nil // No .gitignore or can't read — return nil filter (not an error)
	}

	content, ok := raw.(string)
	if !ok || strings.TrimSpace(content) == "" {
		return nil, nil
	}

	ig := ignore.New(ignore.Options{AllowRelativePaths: true})
	ig.Add(content)

	if len(ig.IgnoreRules()) == 0 {
		return nil, nil
	}

	return func(relativePath string) bool {
		// Normalize paths: strip leading './' or '/'
		normalized := strings.TrimPrefix(relativePath, "./")
		normalized = strings.TrimPrefix(normalized, "/")
		if normalized == "" {
			return false
		}
		return ig.Ignores(normalized)
	}, nil
}
