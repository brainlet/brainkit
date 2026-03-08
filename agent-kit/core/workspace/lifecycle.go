// Ported from: packages/core/src/workspace/lifecycle.ts
package workspace

// =============================================================================
// Status Types
// =============================================================================

// ProviderStatus represents common status values for stateful providers.
//
// Not all providers need status tracking — local/stateless providers
// may not use this. But providers with connection pools or cloud
// instances can use these states.
type ProviderStatus string

const (
	ProviderStatusPending      ProviderStatus = "pending"      // Created but not initialized
	ProviderStatusInitializing ProviderStatus = "initializing" // Running Init()
	ProviderStatusReady        ProviderStatus = "ready"        // Initialized, waiting to start (or stateless and ready)
	ProviderStatusStarting     ProviderStatus = "starting"     // Running Start()
	ProviderStatusRunning      ProviderStatus = "running"      // Active and accepting requests
	ProviderStatusStopping     ProviderStatus = "stopping"     // Running Stop()
	ProviderStatusStopped      ProviderStatus = "stopped"      // Stopped but can restart
	ProviderStatusDestroying   ProviderStatus = "destroying"   // Running Destroy()
	ProviderStatusDestroyed    ProviderStatus = "destroyed"    // Fully cleaned up
	ProviderStatusError        ProviderStatus = "error"        // Something went wrong
)

// =============================================================================
// Base Lifecycle Interface
// =============================================================================

// Lifecycle is the shared lifecycle base for workspace providers.
//
// Contains status tracking, destroy, readiness check, and info retrieval.
// Provider-specific lifecycle methods live in the extended interfaces:
//   - FilesystemLifecycle adds Init()
//   - SandboxLifecycle adds Start() / Stop()
type Lifecycle interface {
	// Status returns the current provider status.
	Status() ProviderStatus

	// StatusError returns the error message when status is "error".
	StatusError() string

	// Destroy cleans up all resources.
	// Called when the workspace is being permanently shut down.
	Destroy() error

	// IsReady checks if the provider is ready for operations.
	// Deprecated: Use Status() == ProviderStatusRunning instead.
	IsReady() (bool, error)

	// GetInfo returns status and metadata.
	GetInfo() (interface{}, error)
}

// =============================================================================
// Filesystem Lifecycle
// =============================================================================

// FilesystemLifecycle is the lifecycle interface for filesystem providers
// (two-phase: init -> destroy).
type FilesystemLifecycle interface {
	Lifecycle

	// Init performs one-time setup operations.
	// Called once when the workspace is first initialized.
	Init() error
}

// =============================================================================
// Sandbox Lifecycle
// =============================================================================

// SandboxLifecycle is the lifecycle interface for sandbox providers
// (three-phase: start -> stop -> destroy).
type SandboxLifecycle interface {
	Lifecycle

	// Start begins active operation.
	// Called to transition from initialized to running state.
	Start() error

	// Stop pauses operation, keeping state for potential restart.
	// Called to temporarily stop without full cleanup.
	Stop() error
}

// =============================================================================
// Lifecycle Helper
// =============================================================================

// LifecycleProvider is a provider that may have lifecycle methods.
// Used by CallLifecycle to dispatch to the correct method.
//
// The underscore-prefixed methods (_Init, _Start, etc.) are the wrapped
// versions that add status tracking and race-condition safety.
// CallLifecycle prefers the wrapped versions when available, falling back
// to the plain methods for interface-only implementations.
//
// The interface requires Status() as the minimum useful method since all
// lifecycle providers must be able to report their current state.
// CallLifecycle uses type assertions to check for specific lifecycle methods
// (_Init, _Start, _Stop, _Destroy, Init, Start, Stop, Destroy).
type LifecycleProvider interface {
	// Status returns the current provider status.
	Status() ProviderStatus
}

// lifecycleIniter is implemented by providers with an _Init method.
type lifecycleIniter interface {
	_Init() error
}

// lifecycleStarter is implemented by providers with a _Start method.
type lifecycleStarter interface {
	_Start() error
}

// lifecycleStopper is implemented by providers with a _Stop method.
type lifecycleStopper interface {
	_Stop() error
}

// lifecycleDestroyer is implemented by providers with a _Destroy method.
type lifecycleDestroyer interface {
	_Destroy() error
}

// plainIniter is implemented by providers with an Init method.
type plainIniter interface {
	Init() error
}

// plainStarter is implemented by providers with a Start method.
type plainStarter interface {
	Start() error
}

// plainStopper is implemented by providers with a Stop method.
type plainStopper interface {
	Stop() error
}

// plainDestroyer is implemented by providers with a Destroy method.
type plainDestroyer interface {
	Destroy() error
}

// CallLifecycle calls a lifecycle method on a provider, preferring the
// underscore-prefixed wrapper (which adds status tracking & race-condition
// safety) when available, falling back to the plain method for
// interface-only implementations.
//
// Example:
//
//	CallLifecycle(sandbox, "start")   // calls sandbox._Start() ?? sandbox.Start()
//	CallLifecycle(filesystem, "init") // calls filesystem._Init() ?? filesystem.Init()
func CallLifecycle(provider LifecycleProvider, method string) error {
	switch method {
	case "init":
		if p, ok := provider.(lifecycleIniter); ok {
			return p._Init()
		}
		if p, ok := provider.(plainIniter); ok {
			return p.Init()
		}
	case "start":
		if p, ok := provider.(lifecycleStarter); ok {
			return p._Start()
		}
		if p, ok := provider.(plainStarter); ok {
			return p.Start()
		}
	case "stop":
		if p, ok := provider.(lifecycleStopper); ok {
			return p._Stop()
		}
		if p, ok := provider.(plainStopper); ok {
			return p.Stop()
		}
	case "destroy":
		if p, ok := provider.(lifecycleDestroyer); ok {
			return p._Destroy()
		}
		if p, ok := provider.(plainDestroyer); ok {
			return p.Destroy()
		}
	}
	return nil
}
