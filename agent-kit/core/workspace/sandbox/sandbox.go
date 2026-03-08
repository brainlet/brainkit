// Ported from: packages/core/src/workspace/sandbox/sandbox.ts
package sandbox

// =============================================================================
// Sandbox Interface
// =============================================================================

// WorkspaceSandbox defines the contract for sandbox providers that can be used
// with Workspace. Users pass sandbox provider instances to the Workspace constructor.
//
// Sandboxes provide isolated environments for code and command execution.
// They may have their own filesystem that's separate from the workspace FS.
//
// Built-in providers (via ComputeSDK):
//   - E2B: Cloud sandboxes
//   - Modal: GPU-enabled sandboxes
//   - Docker: Container-based execution
//   - Local: Development-only local execution
type WorkspaceSandbox interface {
	// ID returns the unique identifier for this sandbox instance.
	ID() string

	// Name returns the human-readable name (e.g., 'LocalSandbox', 'E2B Sandbox').
	Name() string

	// Provider returns the provider type identifier.
	Provider() string

	// GetInstructions returns instructions describing how this sandbox works.
	// Used in tool descriptions to help agents understand execution context.
	GetInstructions() string

	// ---------------------------------------------------------------------------
	// Lifecycle
	// ---------------------------------------------------------------------------

	// Start begins active operation (spin up instance).
	Start() error

	// Stop pauses operation (pause instance).
	Stop() error

	// Destroy cleans up all resources (terminate instance).
	Destroy() error

	// IsReady checks if the sandbox is ready for operations.
	// Deprecated: Use Status() == ProviderStatusRunning instead.
	IsReady() bool

	// GetInfo returns status and metadata.
	GetInfo() (*SandboxInfo, error)

	// Status returns the current provider status.
	Status() ProviderStatus

	// EnsureRunning ensures the sandbox is running, starting it if needed.
	EnsureRunning() error

	// ---------------------------------------------------------------------------
	// Command Execution
	// ---------------------------------------------------------------------------

	// ExecuteCommand executes a shell command and waits for it to complete.
	// Returns nil if the sandbox doesn't support command execution.
	ExecuteCommand(command string, args []string, options *ExecuteCommandOptions) (*CommandResult, error)

	// ---------------------------------------------------------------------------
	// Process Management (Optional)
	// ---------------------------------------------------------------------------

	// Processes returns the process manager, or nil if not supported.
	Processes() ProcessManager

	// ---------------------------------------------------------------------------
	// Mounting Support (Optional)
	// ---------------------------------------------------------------------------

	// Mounts returns the mount manager, or nil if not supported.
	Mounts() *MountManager

	// Mount mounts a filesystem at a path in the sandbox.
	// Returns nil, nil if not supported.
	Mount(filesystem WorkspaceFilesystemRef, mountPath string) (*MountResult, error)

	// Unmount unmounts a filesystem from a path in the sandbox.
	Unmount(mountPath string) error
}

// ProcessManager is the interface for sandbox process managers.
// This is a forward declaration; the full interface is in processmanager/.
type ProcessManager interface {
	// Spawn spawns a new process.
	Spawn(command string, options *SpawnProcessOptions) (*ProcessHandle, error)
	// List returns info about all tracked processes.
	List() ([]ProcessInfo, error)
	// Get returns a handle to a process by PID.
	Get(pid int) (*ProcessHandle, error)
	// Kill kills a process by PID.
	Kill(pid int) (bool, error)
}

// SpawnProcessOptions holds options for spawning a process.
type SpawnProcessOptions = CommandOptions

// ProcessInfo holds information about a tracked process.
type ProcessInfo struct {
	// PID is the process ID.
	PID int
	// Command is the command that was executed (if available).
	Command string
	// Running indicates whether the process is still running.
	Running bool
	// ExitCode is the exit code if the process has finished.
	ExitCode *int
}
