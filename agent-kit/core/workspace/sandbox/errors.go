// Ported from: packages/core/src/workspace/sandbox/errors.ts
package sandbox

import (
	"fmt"
)

// =============================================================================
// Base Error
// =============================================================================

// SandboxError is the base error type for sandbox operations.
type SandboxError struct {
	Message string
	Code    string
	Details map[string]interface{}
}

func (e *SandboxError) Error() string {
	return e.Message
}

// NewSandboxError creates a new SandboxError.
func NewSandboxError(message, code string, details map[string]interface{}) *SandboxError {
	return &SandboxError{
		Message: message,
		Code:    code,
		Details: details,
	}
}

// =============================================================================
// Execution Errors
// =============================================================================

// SandboxExecutionError is returned when command execution fails.
type SandboxExecutionError struct {
	SandboxError
	ExitCode int
	Stdout   string
	Stderr   string
}

// NewSandboxExecutionError creates a new SandboxExecutionError.
func NewSandboxExecutionError(message string, exitCode int, stdout, stderr string) *SandboxExecutionError {
	return &SandboxExecutionError{
		SandboxError: SandboxError{
			Message: message,
			Code:    "EXECUTION_FAILED",
			Details: map[string]interface{}{
				"exitCode": exitCode,
				"stdout":   stdout,
				"stderr":   stderr,
			},
		},
		ExitCode: exitCode,
		Stdout:   stdout,
		Stderr:   stderr,
	}
}

// SandboxTimeoutError is returned when a sandbox operation times out.
type SandboxTimeoutError struct {
	SandboxError
	TimeoutMs int64
	Operation SandboxOperation
}

// NewSandboxTimeoutError creates a new SandboxTimeoutError.
func NewSandboxTimeoutError(timeoutMs int64, operation SandboxOperation) *SandboxTimeoutError {
	return &SandboxTimeoutError{
		SandboxError: SandboxError{
			Message: fmt.Sprintf("Execution timed out after %dms", timeoutMs),
			Code:    "TIMEOUT",
			Details: map[string]interface{}{
				"timeoutMs": timeoutMs,
				"operation": string(operation),
			},
		},
		TimeoutMs: timeoutMs,
		Operation: operation,
	}
}

// SandboxNotReadyError is returned when the sandbox is not ready.
type SandboxNotReadyError struct {
	SandboxError
}

// NewSandboxNotReadyError creates a new SandboxNotReadyError.
func NewSandboxNotReadyError(idOrStatus string) *SandboxNotReadyError {
	return &SandboxNotReadyError{
		SandboxError: SandboxError{
			Message: fmt.Sprintf("Sandbox is not ready: %s", idOrStatus),
			Code:    "NOT_READY",
			Details: map[string]interface{}{
				"id": idOrStatus,
			},
		},
	}
}

// IsolationUnavailableError is returned when a sandbox isolation backend is not available.
type IsolationUnavailableError struct {
	SandboxError
	Backend string
	Reason  string
}

// NewIsolationUnavailableError creates a new IsolationUnavailableError.
func NewIsolationUnavailableError(backend, reason string) *IsolationUnavailableError {
	return &IsolationUnavailableError{
		SandboxError: SandboxError{
			Message: fmt.Sprintf("Isolation backend '%s' is not available: %s", backend, reason),
			Code:    "ISOLATION_UNAVAILABLE",
			Details: map[string]interface{}{
				"backend": backend,
				"reason":  reason,
			},
		},
		Backend: backend,
		Reason:  reason,
	}
}

// =============================================================================
// Mount Errors
// =============================================================================

// MountError is the base error for mount operations.
type MountError struct {
	SandboxError
	MountPath string
}

// NewMountError creates a new MountError.
func NewMountError(message, mountPath string, details map[string]interface{}) *MountError {
	merged := map[string]interface{}{
		"mountPath": mountPath,
	}
	for k, v := range details {
		merged[k] = v
	}
	return &MountError{
		SandboxError: SandboxError{
			Message: message,
			Code:    "MOUNT_ERROR",
			Details: merged,
		},
		MountPath: mountPath,
	}
}

// MountNotSupportedError is returned when the sandbox doesn't support mounting.
type MountNotSupportedError struct {
	SandboxError
}

// NewMountNotSupportedError creates a new MountNotSupportedError.
func NewMountNotSupportedError(sandboxProvider string) *MountNotSupportedError {
	return &MountNotSupportedError{
		SandboxError: SandboxError{
			Message: fmt.Sprintf("Sandbox provider '%s' does not support mounting", sandboxProvider),
			Code:    "MOUNT_NOT_SUPPORTED",
			Details: map[string]interface{}{
				"sandboxProvider": sandboxProvider,
			},
		},
	}
}

// FilesystemNotMountableError is returned when a filesystem cannot be mounted.
type FilesystemNotMountableError struct {
	SandboxError
}

// NewFilesystemNotMountableError creates a new FilesystemNotMountableError.
func NewFilesystemNotMountableError(filesystemProvider string, reason string) *FilesystemNotMountableError {
	message := fmt.Sprintf("Filesystem '%s' does not support mounting", filesystemProvider)
	if reason != "" {
		message = fmt.Sprintf("Filesystem '%s' cannot be mounted: %s", filesystemProvider, reason)
	}
	return &FilesystemNotMountableError{
		SandboxError: SandboxError{
			Message: message,
			Code:    "FILESYSTEM_NOT_MOUNTABLE",
			Details: map[string]interface{}{
				"filesystemProvider": filesystemProvider,
				"reason":             reason,
			},
		},
	}
}
