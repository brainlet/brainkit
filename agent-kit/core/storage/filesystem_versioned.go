// Ported from: packages/core/src/storage/filesystem-versioned.ts
//
// The implementation now lives in storage/fsutil to break import cycles between
// storage and storage/domains/*. This file re-exports the types so existing
// callers that reference storage.FilesystemVersionedHelpers continue to work.
package storage

import "github.com/brainlet/brainkit/agent-kit/core/storage/fsutil"

// GIT_VERSION_PREFIX is the prefix for version IDs that come from git history.
const GIT_VERSION_PREFIX = fsutil.GIT_VERSION_PREFIX

// FilesystemVersionedConfig configures a filesystem-backed versioned storage domain.
type FilesystemVersionedConfig = fsutil.FilesystemVersionedConfig

// FilesystemVersionedHelpers provides generic helpers for filesystem-backed
// versioned storage domains.
type FilesystemVersionedHelpers = fsutil.FilesystemVersionedHelpers

// NewFilesystemVersionedHelpers creates a new FilesystemVersionedHelpers instance.
var NewFilesystemVersionedHelpers = fsutil.NewFilesystemVersionedHelpers

// IsGitVersion checks if a version ID represents a git-based version.
var IsGitVersion = fsutil.IsGitVersion
