// Ported from: packages/core/src/workspace/errors.ts
package workspace

import "fmt"

// =============================================================================
// Base Error
// =============================================================================

// WorkspaceError is the base error type for workspace operations.
type WorkspaceError struct {
	Message     string
	Code        string
	WorkspaceID string
}

func (e *WorkspaceError) Error() string {
	return e.Message
}

// NewWorkspaceError creates a new WorkspaceError.
func NewWorkspaceError(message, code string, workspaceID ...string) *WorkspaceError {
	wsID := ""
	if len(workspaceID) > 0 {
		wsID = workspaceID[0]
	}
	return &WorkspaceError{
		Message:     message,
		Code:        code,
		WorkspaceID: wsID,
	}
}

// =============================================================================
// Availability Errors
// =============================================================================

// WorkspaceNotAvailableError is returned when a workspace is not configured.
type WorkspaceNotAvailableError struct {
	WorkspaceError
}

// NewWorkspaceNotAvailableError creates a new WorkspaceNotAvailableError.
func NewWorkspaceNotAvailableError() *WorkspaceNotAvailableError {
	return &WorkspaceNotAvailableError{
		WorkspaceError: WorkspaceError{
			Message: "Workspace not available. Ensure the agent has a workspace configured.",
			Code:    "NO_WORKSPACE",
		},
	}
}

// FilesystemNotAvailableError is returned when a workspace has no filesystem configured.
type FilesystemNotAvailableError struct {
	WorkspaceError
}

// NewFilesystemNotAvailableError creates a new FilesystemNotAvailableError.
func NewFilesystemNotAvailableError() *FilesystemNotAvailableError {
	return &FilesystemNotAvailableError{
		WorkspaceError: WorkspaceError{
			Message: "Workspace does not have a filesystem configured",
			Code:    "NO_FILESYSTEM",
		},
	}
}

// SandboxNotAvailableError is returned when a workspace has no sandbox configured.
type SandboxNotAvailableError struct {
	WorkspaceError
}

// NewSandboxNotAvailableError creates a new SandboxNotAvailableError.
func NewSandboxNotAvailableError(message ...string) *SandboxNotAvailableError {
	msg := "Workspace does not have a sandbox configured"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}
	return &SandboxNotAvailableError{
		WorkspaceError: WorkspaceError{
			Message: msg,
			Code:    "NO_SANDBOX",
		},
	}
}

// SandboxFeature represents a sandbox feature that may not be supported.
type SandboxFeature string

const (
	SandboxFeatureExecuteCommand SandboxFeature = "executeCommand"
	SandboxFeatureInstallPackage SandboxFeature = "installPackage"
	SandboxFeatureProcesses      SandboxFeature = "processes"
)

// SandboxFeatureNotSupportedError is returned when a sandbox feature is not supported.
type SandboxFeatureNotSupportedError struct {
	WorkspaceError
	Feature SandboxFeature
}

// NewSandboxFeatureNotSupportedError creates a new SandboxFeatureNotSupportedError.
func NewSandboxFeatureNotSupportedError(feature SandboxFeature) *SandboxFeatureNotSupportedError {
	return &SandboxFeatureNotSupportedError{
		WorkspaceError: WorkspaceError{
			Message: fmt.Sprintf("Sandbox does not support %s", feature),
			Code:    "FEATURE_NOT_SUPPORTED",
		},
		Feature: feature,
	}
}

// SearchNotAvailableError is returned when search is not configured on a workspace.
type SearchNotAvailableError struct {
	WorkspaceError
}

// NewSearchNotAvailableError creates a new SearchNotAvailableError.
func NewSearchNotAvailableError() *SearchNotAvailableError {
	return &SearchNotAvailableError{
		WorkspaceError: WorkspaceError{
			Message: "Workspace does not have search configured (enable bm25 or provide vectorStore + embedder)",
			Code:    "NO_SEARCH",
		},
	}
}

// =============================================================================
// State Errors
// =============================================================================

// WorkspaceNotReadyError is returned when a workspace is not in a ready state.
type WorkspaceNotReadyError struct {
	WorkspaceError
}

// NewWorkspaceNotReadyError creates a new WorkspaceNotReadyError.
func NewWorkspaceNotReadyError(workspaceID string, status WorkspaceStatus) *WorkspaceNotReadyError {
	return &WorkspaceNotReadyError{
		WorkspaceError: WorkspaceError{
			Message:     fmt.Sprintf("Workspace is not ready (status: %s)", status),
			Code:        "NOT_READY",
			WorkspaceID: workspaceID,
		},
	}
}

// WorkspaceReadOnlyError is returned when a write operation is attempted on a read-only workspace.
type WorkspaceReadOnlyError struct {
	WorkspaceError
	Operation string
}

// NewWorkspaceReadOnlyError creates a new WorkspaceReadOnlyError.
func NewWorkspaceReadOnlyError(operation string) *WorkspaceReadOnlyError {
	return &WorkspaceReadOnlyError{
		WorkspaceError: WorkspaceError{
			Message: fmt.Sprintf("Workspace is in read-only mode. Cannot perform: %s", operation),
			Code:    "READ_ONLY",
		},
		Operation: operation,
	}
}

// =============================================================================
// Filesystem Errors
// =============================================================================

// FilesystemError is the base error type for filesystem operations.
type FilesystemError struct {
	Message string
	Code    string
	Path    string
}

func (e *FilesystemError) Error() string {
	return e.Message
}

// NewFilesystemError creates a new FilesystemError.
func NewFilesystemError(message, code, path string) *FilesystemError {
	return &FilesystemError{
		Message: message,
		Code:    code,
		Path:    path,
	}
}

// FileNotFoundError is returned when a file is not found.
type FileNotFoundError struct {
	FilesystemError
}

// NewFileNotFoundError creates a new FileNotFoundError.
func NewFileNotFoundError(path string) *FileNotFoundError {
	return &FileNotFoundError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("File not found: %s", path),
			Code:    "ENOENT",
			Path:    path,
		},
	}
}

// DirectoryNotFoundError is returned when a directory is not found.
type DirectoryNotFoundError struct {
	FilesystemError
}

// NewDirectoryNotFoundError creates a new DirectoryNotFoundError.
func NewDirectoryNotFoundError(path string) *DirectoryNotFoundError {
	return &DirectoryNotFoundError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("Directory not found: %s", path),
			Code:    "ENOENT",
			Path:    path,
		},
	}
}

// FileExistsError is returned when a file already exists.
type FileExistsError struct {
	FilesystemError
}

// NewFileExistsError creates a new FileExistsError.
func NewFileExistsError(path string) *FileExistsError {
	return &FileExistsError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("File already exists: %s", path),
			Code:    "EEXIST",
			Path:    path,
		},
	}
}

// IsDirectoryError is returned when a path is a directory but a file was expected.
type IsDirectoryError struct {
	FilesystemError
}

// NewIsDirectoryError creates a new IsDirectoryError.
func NewIsDirectoryError(path string) *IsDirectoryError {
	return &IsDirectoryError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("Path is a directory: %s", path),
			Code:    "EISDIR",
			Path:    path,
		},
	}
}

// NotDirectoryError is returned when a path is not a directory.
type NotDirectoryError struct {
	FilesystemError
}

// NewNotDirectoryError creates a new NotDirectoryError.
func NewNotDirectoryError(path string) *NotDirectoryError {
	return &NotDirectoryError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("Path is not a directory: %s", path),
			Code:    "ENOTDIR",
			Path:    path,
		},
	}
}

// DirectoryNotEmptyError is returned when a directory is not empty.
type DirectoryNotEmptyError struct {
	FilesystemError
}

// NewDirectoryNotEmptyError creates a new DirectoryNotEmptyError.
func NewDirectoryNotEmptyError(path string) *DirectoryNotEmptyError {
	return &DirectoryNotEmptyError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("Directory not empty: %s", path),
			Code:    "ENOTEMPTY",
			Path:    path,
		},
	}
}

// PermissionError is returned when a filesystem operation is denied.
type PermissionError struct {
	FilesystemError
	Operation string
}

// NewPermissionError creates a new PermissionError.
func NewPermissionError(path, operation string) *PermissionError {
	return &PermissionError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("Permission denied: %s on %s", operation, path),
			Code:    "EACCES",
			Path:    path,
		},
		Operation: operation,
	}
}

// FileReadRequiredError is returned when a file must be read before writing.
type FileReadRequiredError struct {
	FilesystemError
}

// NewFileReadRequiredError creates a new FileReadRequiredError.
func NewFileReadRequiredError(path, reason string) *FileReadRequiredError {
	return &FileReadRequiredError{
		FilesystemError: FilesystemError{
			Message: reason,
			Code:    "EREAD_REQUIRED",
			Path:    path,
		},
	}
}

// FilesystemNotReadyError is returned when a filesystem operation is
// attempted before initialization.
type FilesystemNotReadyError struct {
	FilesystemError
}

// NewFilesystemNotReadyError creates a new FilesystemNotReadyError.
func NewFilesystemNotReadyError(id string) *FilesystemNotReadyError {
	return &FilesystemNotReadyError{
		FilesystemError: FilesystemError{
			Message: fmt.Sprintf("Filesystem %q is not ready. Call Init() first or use EnsureReady().", id),
			Code:    "ENOTREADY",
			Path:    id,
		},
	}
}
