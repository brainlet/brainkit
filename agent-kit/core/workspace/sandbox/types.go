// Ported from: packages/core/src/workspace/sandbox/types.ts
package sandbox

import (
	"time"
)

// =============================================================================
// Mount State Types
// =============================================================================

// MountState represents the state of a mount in the sandbox.
type MountState string

const (
	MountStatePending     MountState = "pending"
	MountStateMounting    MountState = "mounting"
	MountStateMounted     MountState = "mounted"
	MountStateError       MountState = "error"
	MountStateUnsupported MountState = "unsupported"
	MountStateUnavailable MountState = "unavailable"
)

// MountEntry represents a mount entry tracked by the mount manager.
type MountEntry struct {
	// Filesystem is the filesystem to mount.
	Filesystem WorkspaceFilesystemRef
	// State is the current state of the mount.
	State MountState
	// Error is the error message if state is 'error' or 'unavailable'.
	Error string
	// Config is the resolved mount config from filesystem.GetMountConfig().
	Config *FilesystemMountConfig
	// ConfigHash is the hash of config for quick comparison.
	ConfigHash string
}

// WorkspaceFilesystemRef is a reference to a workspace filesystem.
// This avoids importing the full filesystem package (circular dependency prevention).
type WorkspaceFilesystemRef interface {
	ID() string
	Name() string
	Provider() string
	ReadOnly() bool
	DisplayName() string
	Description() string
	GetMountConfig() *FilesystemMountConfig
}

// FilesystemMountConfig holds mount configuration for a filesystem.
type FilesystemMountConfig struct {
	// Type is the mount type: "local", "s3", "gcs", etc.
	Type string `json:"type"`
	// BasePath is the local base path (for type "local").
	BasePath string `json:"basePath,omitempty"`
	// Bucket is the bucket name (for type "s3" or "gcs").
	Bucket string `json:"bucket,omitempty"`
	// Region is the AWS region (for type "s3").
	Region string `json:"region,omitempty"`
	// Prefix is the key prefix (for type "s3" or "gcs").
	Prefix string `json:"prefix,omitempty"`
}

// MountResult holds the result of a mount operation.
type MountResult struct {
	// Success indicates whether the mount succeeded.
	Success bool
	// MountPath is the path where the filesystem was mounted.
	MountPath string
	// Error is the error message if the mount failed.
	Error string
	// Unavailable indicates the mount tool is not installed (warning, not error).
	Unavailable bool
}

// =============================================================================
// Execution Types
// =============================================================================

// ExecutionResult holds the result of a command execution.
type ExecutionResult struct {
	// Success indicates whether execution completed successfully (exitCode === 0).
	Success bool
	// ExitCode is the exit code (0 = success).
	ExitCode int
	// Stdout is the standard output.
	Stdout string
	// Stderr is the standard error.
	Stderr string
	// ExecutionTimeMs is the execution time in milliseconds.
	ExecutionTimeMs int64
	// TimedOut indicates whether execution timed out.
	TimedOut bool
	// Killed indicates whether execution was killed.
	Killed bool
}

// CommandResult extends ExecutionResult with command metadata.
type CommandResult struct {
	ExecutionResult
	// Command is the command that was executed.
	Command string
	// Args are the arguments passed to the command.
	Args []string
}

// =============================================================================
// Command Options
// =============================================================================

// CommandOptions holds shared options for running commands in a sandbox.
type CommandOptions struct {
	// Timeout in milliseconds. Kills the process if exceeded.
	Timeout int64
	// Env holds environment variables.
	Env map[string]string
	// Cwd is the working directory.
	Cwd string
	// OnStdout is a callback for stdout chunks (enables streaming).
	OnStdout func(data string)
	// OnStderr is a callback for stderr chunks (enables streaming).
	OnStderr func(data string)
	// AbortSignal is used to cancel the command.
	AbortSignal <-chan struct{}
}

// ExecuteCommandOptions holds options for executeCommand.
type ExecuteCommandOptions = CommandOptions

// =============================================================================
// Sandbox Info
// =============================================================================

// SandboxInfo holds information about a sandbox provider's state.
type SandboxInfo struct {
	// ID is the sandbox identifier.
	ID string
	// Name is the human-readable name.
	Name string
	// Provider is the provider type.
	Provider string
	// Status is the current provider status.
	Status ProviderStatus
	// CreatedAt is when the sandbox was created.
	CreatedAt time.Time
	// LastUsedAt is when the sandbox was last used.
	LastUsedAt *time.Time
	// TimeoutAt is the time until auto-shutdown (if applicable).
	TimeoutAt *time.Time
	// Mounts are the current mounts in the sandbox.
	Mounts []SandboxMountInfo
	// Resources holds resource info (if available).
	Resources *SandboxResources
	// Metadata holds provider-specific metadata.
	Metadata map[string]interface{}
}

// SandboxMountInfo holds information about a sandbox mount.
type SandboxMountInfo struct {
	Path       string
	Filesystem string
}

// SandboxResources holds sandbox resource information.
type SandboxResources struct {
	MemoryMB     *float64
	MemoryUsedMB *float64
	CPUCores     *float64
	CPUPercent   *float64
	DiskMB       *float64
	DiskUsedMB   *float64
}

// =============================================================================
// Error Types
// =============================================================================

// SandboxOperation represents sandbox operation types for timeout errors.
type SandboxOperation string

const (
	SandboxOperationCommand SandboxOperation = "command"
	SandboxOperationSync    SandboxOperation = "sync"
	SandboxOperationInstall SandboxOperation = "install"
)

// =============================================================================
// Provider Status (re-export from lifecycle)
// =============================================================================

// ProviderStatus represents common status values for stateful providers.
type ProviderStatus string

const (
	ProviderStatusPending    ProviderStatus = "pending"
	ProviderStatusStarting   ProviderStatus = "starting"
	ProviderStatusRunning    ProviderStatus = "running"
	ProviderStatusStopping   ProviderStatus = "stopping"
	ProviderStatusStopped    ProviderStatus = "stopped"
	ProviderStatusDestroying ProviderStatus = "destroying"
	ProviderStatusDestroyed  ProviderStatus = "destroyed"
	ProviderStatusError      ProviderStatus = "error"
)
