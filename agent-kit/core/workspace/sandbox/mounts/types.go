// Ported from: packages/core/src/workspace/sandbox/mounts/types.ts
package mounts

import (
	"fmt"
)

// =============================================================================
// Shared Types for Local Mount Operations
// =============================================================================

// LogPrefix is the log prefix for local sandbox operations.
const LogPrefix = "[LocalSandbox]"

// RunResult holds the result of running a command.
type RunResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunFunc is a function that runs a command.
type RunFunc func(command string, args []string, timeout *int64) (*RunResult, error)

// LocalMountContext provides context for local mount operations.
type LocalMountContext struct {
	Run      RunFunc
	Platform string
	Logger   MountLogger
}

// MountLogger is a logger interface for mount operations.
type MountLogger interface {
	Debug(message string, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message string, args ...interface{})
}

// =============================================================================
// Mount Tool Not Found Error
// =============================================================================

// MountToolNotFoundError is returned when a required FUSE tool is not installed.
//
// Distinguished from general mount errors so LocalSandbox.Mount() can mark the
// mount as 'unavailable' (warning) rather than 'error'. The workspace still works
// via SDK filesystem methods — only sandbox process access to the mount path is affected.
type MountToolNotFoundError struct {
	Message string
}

func (e *MountToolNotFoundError) Error() string {
	return e.Message
}

// NewMountToolNotFoundError creates a new MountToolNotFoundError.
func NewMountToolNotFoundError(format string, args ...interface{}) *MountToolNotFoundError {
	return &MountToolNotFoundError{
		Message: fmt.Sprintf(format, args...),
	}
}
