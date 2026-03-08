// Ported from: packages/core/src/workspace/sandbox/process-manager/process-manager.ts
package sandbox

import (
	"sync"
)

// =============================================================================
// Base Process Manager
// =============================================================================

// BaseProcessManager is the base implementation for sandbox process managers.
//
// Wraps subclass overrides of Spawn(), List(), and Get() with
// EnsureRunning() so the sandbox is lazily started before
// any process operation.
//
// Subclasses implement the actual platform-specific logic for all methods.
type BaseProcessManager struct {
	// Sandbox is the sandbox this process manager belongs to.
	// Set automatically by MastraSandbox when processes are passed into the constructor.
	Sandbox WorkspaceSandbox

	// Env holds environment variables for the process manager.
	Env map[string]string

	// Tracked holds process handles keyed by PID.
	Tracked map[int]*ProcessHandle

	// Dismissed holds PIDs that have been read after exit.
	Dismissed map[int]bool

	mu sync.RWMutex

	// SpawnImpl is the implementation-specific spawn function.
	SpawnImpl func(command string, options *SpawnProcessOptions) (*ProcessHandle, error)
	// ListImpl is the implementation-specific list function.
	ListImpl func() ([]ProcessInfo, error)
	// GetImpl is the implementation-specific get function.
	// If nil, defaults to looking up in Tracked map.
	GetImpl func(pid int) (*ProcessHandle, error)
}

// NewBaseProcessManager creates a new BaseProcessManager.
func NewBaseProcessManager(env map[string]string) *BaseProcessManager {
	if env == nil {
		env = make(map[string]string)
	}
	return &BaseProcessManager{
		Env:       env,
		Tracked:   make(map[int]*ProcessHandle),
		Dismissed: make(map[int]bool),
	}
}

// Spawn spawns a process with sandbox ensureRunning() wrapper.
func (pm *BaseProcessManager) Spawn(command string, options *SpawnProcessOptions) (*ProcessHandle, error) {
	if pm.Sandbox != nil {
		if err := pm.Sandbox.EnsureRunning(); err != nil {
			return nil, err
		}
	}

	if pm.SpawnImpl == nil {
		return nil, &SandboxError{Message: "SpawnImpl not set", Code: "NOT_IMPLEMENTED"}
	}

	handle, err := pm.SpawnImpl(command, options)
	if err != nil {
		return nil, err
	}

	handle.Command = command

	// Wire abort signal to handle.Kill()
	if options != nil && options.AbortSignal != nil {
		abortCh := options.AbortSignal
		go func() {
			select {
			case <-abortCh:
				_, _ = handle.Kill()
			}
		}()
	}

	return handle, nil
}

// List lists all tracked processes with sandbox ensureRunning() wrapper.
func (pm *BaseProcessManager) List() ([]ProcessInfo, error) {
	if pm.Sandbox != nil {
		if err := pm.Sandbox.EnsureRunning(); err != nil {
			return nil, err
		}
	}

	if pm.ListImpl != nil {
		return pm.ListImpl()
	}

	// Default: build list from tracked handles
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var result []ProcessInfo
	for _, handle := range pm.Tracked {
		info := ProcessInfo{
			PID:     handle.PID,
			Command: handle.Command,
			Running: handle.ExitCode == nil,
		}
		if handle.ExitCode != nil {
			code := *handle.ExitCode
			info.ExitCode = &code
		}
		result = append(result, info)
	}
	return result, nil
}

// Get returns a handle to a process by PID with sandbox ensureRunning() wrapper.
// Prunes exited processes when their output is read.
func (pm *BaseProcessManager) Get(pid int) (*ProcessHandle, error) {
	if pm.Sandbox != nil {
		if err := pm.Sandbox.EnsureRunning(); err != nil {
			return nil, err
		}
	}

	pm.mu.RLock()
	if pm.Dismissed[pid] {
		pm.mu.RUnlock()
		return nil, nil
	}
	pm.mu.RUnlock()

	var handle *ProcessHandle
	var err error

	if pm.GetImpl != nil {
		handle, err = pm.GetImpl(pid)
	} else {
		pm.mu.RLock()
		handle = pm.Tracked[pid]
		pm.mu.RUnlock()
	}

	if err != nil {
		return nil, err
	}

	// Prune exited processes when their output is read
	if handle != nil && handle.ExitCode != nil {
		pm.mu.Lock()
		delete(pm.Tracked, handle.PID)
		pm.Dismissed[handle.PID] = true
		pm.mu.Unlock()
	}

	return handle, nil
}

// Kill kills a process by PID. Returns true if killed, false if not found.
func (pm *BaseProcessManager) Kill(pid int) (bool, error) {
	handle, err := pm.Get(pid)
	if err != nil {
		return false, err
	}
	if handle == nil {
		return false, nil
	}

	killed, err := handle.Kill()
	if err != nil {
		return false, err
	}

	if killed {
		// Wait for termination so handle.ExitCode is populated before returning.
		_, _ = handle.Wait(nil)
	}

	// Release tracked handle to free accumulated output buffers.
	pm.mu.Lock()
	delete(pm.Tracked, pid)
	pm.Dismissed[pid] = true
	pm.mu.Unlock()

	return killed, nil
}

// TrackHandle adds a process handle to the tracked map.
func (pm *BaseProcessManager) TrackHandle(handle *ProcessHandle) {
	pm.mu.Lock()
	pm.Tracked[handle.PID] = handle
	pm.mu.Unlock()
}
