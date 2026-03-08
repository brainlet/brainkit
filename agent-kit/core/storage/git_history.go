// Ported from: packages/core/src/storage/git-history.ts
//
// The implementation now lives in storage/fsutil to break import cycles between
// storage and storage/domains/*. This file re-exports the types so existing
// callers that reference storage.GitHistory continue to work.
package storage

import "github.com/brainlet/brainkit/agent-kit/core/storage/fsutil"

// GitCommit represents a single Git commit entry parsed from `git log` output.
type GitCommit = fsutil.GitCommit

// GitHistory is a read-only utility for reading Git history of
// filesystem-stored JSON files.
type GitHistory = fsutil.GitHistory

// NewGitHistory creates a new GitHistory instance.
var NewGitHistory = fsutil.NewGitHistory
