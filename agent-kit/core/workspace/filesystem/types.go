// Ported from: packages/core/src/workspace/filesystem/filesystem.ts
package filesystem

import (
	"time"
)

// =============================================================================
// File System Types
// =============================================================================

// FileStat represents file/directory metadata returned by Stat().
type FileStat struct {
	// Name is the basename of the file or directory.
	Name string
	// Path is the full path within the filesystem.
	Path string
	// Type is "file" or "directory".
	Type string
	// Size in bytes.
	Size int64
	// CreatedAt is the creation time.
	CreatedAt time.Time
	// ModifiedAt is the last modification time.
	ModifiedAt time.Time
	// MimeType is the MIME type (optional, for files).
	MimeType string
}

// FileEntry represents a directory entry returned by Readdir().
type FileEntry struct {
	// Name is the basename of the entry.
	Name string
	// Type is "file" or "directory".
	Type string
	// Size in bytes (optional).
	Size int64
	// IsSymlink indicates if the entry is a symbolic link.
	IsSymlink bool
	// SymlinkTarget is the target of the symlink (optional).
	SymlinkTarget string
	// Mount holds mount point metadata if this entry is a mount point.
	Mount *FileEntryMount
}

// FileEntryMount holds mount point metadata for a directory entry.
type FileEntryMount struct {
	Provider    string
	Icon        *FilesystemIcon
	DisplayName string
	Description string
	Status      *ProviderStatus
	Error       string
}

// ProviderStatus represents the status of a provider.
type ProviderStatus string

const (
	ProviderStatusPending      ProviderStatus = "pending"
	ProviderStatusInitializing ProviderStatus = "initializing"
	ProviderStatusReady        ProviderStatus = "ready"
	ProviderStatusError        ProviderStatus = "error"
	ProviderStatusDestroying   ProviderStatus = "destroying"
	ProviderStatusDestroyed    ProviderStatus = "destroyed"
	ProviderStatusStarting     ProviderStatus = "starting"
	ProviderStatusRunning      ProviderStatus = "running"
	ProviderStatusStopping     ProviderStatus = "stopping"
	ProviderStatusStopped      ProviderStatus = "stopped"
)

// =============================================================================
// Options Types
// =============================================================================

// ReadOptions configures file read behavior.
type ReadOptions struct {
	// Encoding specifies the text encoding (e.g., "utf-8", "base64").
	Encoding string
}

// WriteOptions configures file write behavior.
type WriteOptions struct {
	// Recursive creates parent directories if they don't exist.
	Recursive bool
	// Overwrite allows overwriting existing files.
	Overwrite *bool
	// MimeType sets the MIME type for the written file.
	MimeType string
}

// ListOptions configures directory listing behavior.
type ListOptions struct {
	// Recursive lists entries recursively.
	Recursive bool
	// Extension filters by file extension.
	Extension []string
	// MaxDepth limits recursion depth.
	MaxDepth int
}

// RemoveOptions configures file/directory removal behavior.
type RemoveOptions struct {
	// Recursive removes directories and their contents.
	Recursive bool
	// Force ignores errors if the path doesn't exist.
	Force bool
}

// CopyOptions configures file copy/move behavior.
type CopyOptions struct {
	// Overwrite allows overwriting existing files.
	Overwrite bool
	// Recursive copies directories recursively.
	Recursive bool
}

// MkdirOptions configures directory creation behavior.
type MkdirOptions struct {
	// Recursive creates parent directories if needed.
	Recursive bool
}

// =============================================================================
// Filesystem Info
// =============================================================================

// FilesystemInfo holds information about a filesystem provider's state.
type FilesystemInfo struct {
	ID       string
	Name     string
	Provider string
	Status   *ProviderStatus
	Error    string
	ReadOnly bool
	Icon     *FilesystemIcon
	Metadata map[string]interface{}
}

// FilesystemAuditEntry records a filesystem operation for auditing.
type FilesystemAuditEntry struct {
	// Operation is the type of operation (e.g., "read", "write", "delete").
	Operation string
	// Path is the affected file path.
	Path string
	// Timestamp is when the operation occurred.
	Timestamp time.Time
	// Success indicates if the operation succeeded.
	Success bool
	// Error holds the error message if the operation failed.
	Error string
	// Details holds additional operation-specific details.
	Details map[string]interface{}
}

// =============================================================================
// Workspace Filesystem Interface
// =============================================================================

// WorkspaceFilesystem is the abstract filesystem interface for workspace storage.
//
// All paths are workspace-relative (virtual paths starting with "/").
// Implementations handle path resolution to their backing store.
type WorkspaceFilesystem interface {
	// Identity
	ID() string
	Name() string
	Provider() string
	ReadOnly() bool
	BasePath() string
	Icon() *FilesystemIcon
	DisplayName() string
	Description() string

	// Instructions
	GetInstructions(opts *InstructionsOpts) string

	// Mount config
	GetMountConfig() *FilesystemMountConfig

	// File operations
	ReadFile(path string, options *ReadOptions) (interface{}, error)
	WriteFile(path string, content interface{}, options *WriteOptions) error
	AppendFile(path string, content interface{}) error
	DeleteFile(path string, options *RemoveOptions) error
	CopyFile(src, dest string, options *CopyOptions) error
	MoveFile(src, dest string, options *CopyOptions) error

	// Directory operations
	Mkdir(path string, options *MkdirOptions) error
	Rmdir(path string, options *RemoveOptions) error
	Readdir(path string, options *ListOptions) ([]FileEntry, error)

	// Path operations
	ResolveAbsolutePath(path string) string
	Exists(path string) (bool, error)
	Stat(path string) (*FileStat, error)

	// Lifecycle
	Init() error
	Destroy() error
	GetInfo() (*FilesystemInfo, error)

	// Status
	Status() ProviderStatus
}

// InstructionsOpts holds options for GetInstructions calls.
type InstructionsOpts struct {
	// RequestContext is the request context for dynamic instructions.
	RequestContext interface{}
}
