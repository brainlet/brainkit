// Ported from: packages/core/src/storage/filesystem-db.ts
//
// The implementation now lives in storage/fsutil to break import cycles between
// storage and storage/domains/*. This file re-exports the types so existing
// callers that reference storage.FilesystemDB continue to work.
package storage

import "github.com/brainlet/brainkit/agent-kit/core/storage/fsutil"

// FilesystemDB is a thin I/O layer for filesystem-based storage.
// See fsutil.FilesystemDB for full documentation.
type FilesystemDB = fsutil.FilesystemDB

// NewFilesystemDB creates a new FilesystemDB for the given directory.
var NewFilesystemDB = fsutil.NewFilesystemDB
