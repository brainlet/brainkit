// Ported from: packages/core/src/workspace/sandbox/native-sandbox/types.ts
package nativesandbox

// =============================================================================
// Isolation Backend Types
// =============================================================================

// IsolationBackend represents available isolation backends.
//   - "none": No sandboxing (direct execution on host)
//   - "seatbelt": macOS sandbox-exec (built-in)
//   - "bwrap": Linux bubblewrap (requires installation)
type IsolationBackend string

const (
	IsolationNone     IsolationBackend = "none"
	IsolationSeatbelt IsolationBackend = "seatbelt"
	IsolationBwrap    IsolationBackend = "bwrap"
)

// NativeSandboxConfig holds configuration for native sandboxing.
// These options control filesystem and network access within the sandbox.
type NativeSandboxConfig struct {
	// AllowNetwork allows network access from within the sandbox.
	// Default: false
	AllowNetwork bool

	// ReadOnlyPaths are additional paths to allow read-only access to.
	// These paths will be mounted/allowed in addition to system defaults.
	ReadOnlyPaths []string

	// ReadWritePaths are additional paths to allow read-write access to.
	// By default, only the workspace directory has write access.
	ReadWritePaths []string

	// AllowSystemBinaries allows executing system binaries (node, python, etc.)
	// When false, only binaries within the workspace can be executed.
	// Default: true (set to false explicitly to disable).
	AllowSystemBinaries *bool

	// SeatbeltProfilePath is the path to a custom seatbelt profile file (macOS only).
	// If the file exists, its contents are used as the sandbox profile.
	// If the file doesn't exist, a default profile is generated and written to this path.
	// Must contain valid SBPL (Sandbox Profile Language) if provided.
	SeatbeltProfilePath string

	// BwrapArgs are custom bwrap arguments (Linux only).
	// When provided, these completely replace the default bwrap arguments.
	// The command and its args are appended after these.
	BwrapArgs []string
}

// SandboxDetectionResult holds the result of sandbox backend detection.
type SandboxDetectionResult struct {
	// Backend is the detected/recommended backend.
	Backend IsolationBackend
	// Available indicates whether the backend is available and functional.
	Available bool
	// Message is a human-readable message about the detection result.
	Message string
}
