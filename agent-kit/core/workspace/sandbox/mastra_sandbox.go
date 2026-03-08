// Ported from: packages/core/src/workspace/sandbox/mastra-sandbox.ts
package sandbox

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

// =============================================================================
// Lifecycle Hook
// =============================================================================

// SandboxLifecycleHook is a lifecycle hook that fires during sandbox state transitions.
type SandboxLifecycleHook func(sandbox WorkspaceSandbox) error

// =============================================================================
// MastraSandbox Options
// =============================================================================

// MastraSandboxOptions holds options for the MastraSandbox base.
type MastraSandboxOptions struct {
	// OnStart is called after the sandbox reaches 'running' status.
	OnStart SandboxLifecycleHook
	// OnStop is called before the sandbox stops.
	OnStop SandboxLifecycleHook
	// OnDestroy is called before the sandbox is destroyed.
	OnDestroy SandboxLifecycleHook
	// Processes is the process manager for this sandbox.
	Processes *BaseProcessManager
}

// =============================================================================
// MastraSandbox Base
// =============================================================================

// MastraSandboxBase provides the base implementation for sandbox providers
// with automatic logger integration and lifecycle management.
//
// MountManager is automatically created if the subclass implements Mount().
//
// ## Lifecycle Management
//
// The base class provides race-condition-safe lifecycle wrappers:
//   - WrappedStart() - Handles concurrent calls, status management, and mount processing
//   - WrappedStop() - Handles concurrent calls and status management
//   - WrappedDestroy() - Handles concurrent calls and status management
//
// Subclasses override the plain Start(), Stop(), and Destroy() methods
// to provide their implementation.
type MastraSandboxBase struct {
	// StatusValue is the current provider status.
	StatusValue ProviderStatus

	// ProcessManager is the process manager (if configured).
	ProcessManager *BaseProcessManager

	// MountMgr is the mount manager (if mount support is available).
	MountMgr *MountManager

	// Logger is the logger for this sandbox.
	Logger SandboxLogger

	// StartImpl is the subclass's start implementation.
	StartImpl func() error
	// StopImpl is the subclass's stop implementation.
	StopImpl func() error
	// DestroyImpl is the subclass's destroy implementation.
	DestroyImpl func() error
	// MountImpl is the subclass's mount implementation (optional).
	MountImpl func(filesystem WorkspaceFilesystemRef, mountPath string) (*MountResult, error)

	onStart  SandboxLifecycleHook
	onStop   SandboxLifecycleHook
	onDestroy SandboxLifecycleHook

	startPromise  *sandboxPromise
	stopPromise   *sandboxPromise
	destroyPromise *sandboxPromise
	mu            sync.Mutex
}

// SandboxLogger is the logger interface for sandboxes.
type SandboxLogger interface {
	Debug(message string, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message string, args ...interface{})
}

// defaultLogger is a basic logger that writes to standard log.
type defaultLogger struct{}

func (l *defaultLogger) Debug(message string, _ ...interface{}) { log.Println("[DEBUG]", message) }
func (l *defaultLogger) Info(message string, _ ...interface{})  { log.Println("[INFO]", message) }
func (l *defaultLogger) Warn(message string, _ ...interface{})  { log.Println("[WARN]", message) }
func (l *defaultLogger) Error(message string, _ ...interface{}) { log.Println("[ERROR]", message) }

// InitMastraSandboxBase initializes the MastraSandboxBase fields.
// Call this from the subclass constructor.
func (b *MastraSandboxBase) InitMastraSandboxBase(options MastraSandboxOptions) {
	b.StatusValue = ProviderStatusPending
	b.Logger = &defaultLogger{}

	b.onStart = options.OnStart
	b.onStop = options.OnStop
	b.onDestroy = options.OnDestroy

	// Wire up process manager if provided
	if options.Processes != nil {
		b.ProcessManager = options.Processes
	}

	// MountManager is created by the subclass if it supports mounting
}

// InitMountManager creates a MountManager for sandboxes that support mounting.
// Call this from the subclass constructor if the subclass supports Mount().
func (b *MastraSandboxBase) InitMountManager(self WorkspaceSandbox) {
	if b.MountImpl != nil {
		b.MountMgr = NewMountManager(MountManagerConfig{
			Mount:  b.MountImpl,
			Logger: b.Logger,
		})
	}
	// Wire up process manager sandbox reference
	if b.ProcessManager != nil {
		b.ProcessManager.Sandbox = self
	}
}

// SetupExecuteCommand creates a default ExecuteCommand implementation using
// the process manager (spawn + wait). Returns the function to be used.
func (b *MastraSandboxBase) SetupExecuteCommand(name string) func(command string, args []string, opts *ExecuteCommandOptions) (*CommandResult, error) {
	if b.ProcessManager == nil {
		return nil
	}
	pm := b.ProcessManager
	logger := b.Logger
	return func(command string, args []string, opts *ExecuteCommandOptions) (*CommandResult, error) {
		fullCommand := command
		if len(args) > 0 {
			quotedArgs := make([]string, len(args))
			for i, a := range args {
				quotedArgs[i] = ShellQuote(a)
			}
			fullCommand = command + " " + strings.Join(quotedArgs, " ")
		}
		logger.Debug(fmt.Sprintf("[%s] Executing: %s", name, fullCommand))

		spawnOpts := &SpawnProcessOptions{}
		if opts != nil {
			spawnOpts = opts
		}

		handle, err := pm.Spawn(fullCommand, spawnOpts)
		if err != nil {
			return nil, err
		}

		result, err := handle.Wait(nil)
		if err != nil {
			return nil, err
		}

		logger.Debug(fmt.Sprintf("[%s] Exit code: %d (%dms)", name, result.ExitCode, result.ExecutionTimeMs))

		result.Command = fullCommand
		return result, nil
	}
}

// =============================================================================
// Lifecycle Wrappers (race-condition-safe)
// =============================================================================

// sandboxPromise tracks an in-flight lifecycle operation.
type sandboxPromise struct {
	done chan struct{}
	err  error
}

// WrappedStart starts the sandbox with status management and race-condition safety.
func (b *MastraSandboxBase) WrappedStart() error {
	b.mu.Lock()

	// Already running
	if b.StatusValue == ProviderStatusRunning {
		b.mu.Unlock()
		return nil
	}

	// Wait for in-flight stop/destroy before starting
	if b.stopPromise != nil {
		p := b.stopPromise
		b.mu.Unlock()
		<-p.done
		b.mu.Lock()
	}
	if b.destroyPromise != nil {
		p := b.destroyPromise
		b.mu.Unlock()
		<-p.done
		b.mu.Lock()
	}

	// Cannot start a destroyed sandbox
	if b.StatusValue == ProviderStatusDestroyed {
		b.mu.Unlock()
		return fmt.Errorf("cannot start a destroyed sandbox")
	}

	// Start already in progress - wait for it
	if b.startPromise != nil {
		p := b.startPromise
		b.mu.Unlock()
		<-p.done
		return p.err
	}

	// Create and store the start promise
	p := &sandboxPromise{done: make(chan struct{})}
	b.startPromise = p
	b.mu.Unlock()

	err := b.executeStart()
	p.err = err
	close(p.done)

	b.mu.Lock()
	b.startPromise = nil
	b.mu.Unlock()

	return err
}

func (b *MastraSandboxBase) executeStart() error {
	b.StatusValue = ProviderStatusStarting

	if b.StartImpl != nil {
		if err := b.StartImpl(); err != nil {
			b.StatusValue = ProviderStatusError
			return err
		}
	}

	b.StatusValue = ProviderStatusRunning

	// Fire onStart callback (non-fatal)
	if b.onStart != nil {
		// The sandbox reference is set by the subclass through InitMountManager
		if err := b.onStart(nil); err != nil {
			b.Logger.Warn(fmt.Sprintf("onStart callback failed: %v", err))
		}
	}

	// Process any pending mounts
	if b.MountMgr != nil {
		if err := b.MountMgr.ProcessPending(); err != nil {
			b.Logger.Warn(fmt.Sprintf("Unexpected error processing pending mounts: %v", err))
		}
	}

	return nil
}

// EnsureRunning ensures the sandbox is running, starting it if needed.
func (b *MastraSandboxBase) EnsureRunning() error {
	if b.StatusValue == ProviderStatusDestroyed {
		return NewSandboxNotReadyError("destroyed")
	}
	// During teardown, allow operations to proceed
	if b.StatusValue == ProviderStatusDestroying || b.StatusValue == ProviderStatusStopping {
		return nil
	}
	if b.StatusValue != ProviderStatusRunning {
		if err := b.WrappedStart(); err != nil {
			return err
		}
	}
	if b.StatusValue != ProviderStatusRunning {
		return NewSandboxNotReadyError(string(b.StatusValue))
	}
	return nil
}

// WrappedStop stops the sandbox with status management and race-condition safety.
func (b *MastraSandboxBase) WrappedStop() error {
	b.mu.Lock()

	// Already stopped
	if b.StatusValue == ProviderStatusStopped {
		b.mu.Unlock()
		return nil
	}

	// Wait for in-flight start
	if b.startPromise != nil {
		p := b.startPromise
		b.mu.Unlock()
		<-p.done
		b.mu.Lock()
	}

	// Stop already in progress - wait for it
	if b.stopPromise != nil {
		p := b.stopPromise
		b.mu.Unlock()
		<-p.done
		return p.err
	}

	// Create and store the stop promise
	p := &sandboxPromise{done: make(chan struct{})}
	b.stopPromise = p
	b.mu.Unlock()

	err := b.executeStop()
	p.err = err
	close(p.done)

	b.mu.Lock()
	b.stopPromise = nil
	b.mu.Unlock()

	return err
}

func (b *MastraSandboxBase) executeStop() error {
	b.StatusValue = ProviderStatusStopping

	// Fire onStop callback
	if b.onStop != nil {
		if err := b.onStop(nil); err != nil {
			b.Logger.Warn(fmt.Sprintf("onStop callback failed: %v", err))
		}
	}

	if b.StopImpl != nil {
		if err := b.StopImpl(); err != nil {
			b.StatusValue = ProviderStatusError
			return err
		}
	}

	b.StatusValue = ProviderStatusStopped
	return nil
}

// WrappedDestroy destroys the sandbox with status management and race-condition safety.
func (b *MastraSandboxBase) WrappedDestroy() error {
	b.mu.Lock()

	// Already destroyed
	if b.StatusValue == ProviderStatusDestroyed {
		b.mu.Unlock()
		return nil
	}

	// Never started — nothing to clean up
	if b.StatusValue == ProviderStatusPending {
		b.StatusValue = ProviderStatusDestroyed
		b.mu.Unlock()
		return nil
	}

	// Wait for in-flight start/stop
	if b.startPromise != nil {
		p := b.startPromise
		b.mu.Unlock()
		<-p.done
		b.mu.Lock()
	}
	if b.stopPromise != nil {
		p := b.stopPromise
		b.mu.Unlock()
		<-p.done
		b.mu.Lock()
	}

	// Destroy already in progress - wait for it
	if b.destroyPromise != nil {
		p := b.destroyPromise
		b.mu.Unlock()
		<-p.done
		return p.err
	}

	// Create and store the destroy promise
	p := &sandboxPromise{done: make(chan struct{})}
	b.destroyPromise = p
	b.mu.Unlock()

	err := b.executeDestroy()
	p.err = err
	close(p.done)

	b.mu.Lock()
	b.destroyPromise = nil
	b.mu.Unlock()

	return err
}

func (b *MastraSandboxBase) executeDestroy() error {
	b.StatusValue = ProviderStatusDestroying

	// Fire onDestroy callback
	if b.onDestroy != nil {
		if err := b.onDestroy(nil); err != nil {
			b.Logger.Warn(fmt.Sprintf("onDestroy callback failed: %v", err))
		}
	}

	if b.DestroyImpl != nil {
		if err := b.DestroyImpl(); err != nil {
			b.StatusValue = ProviderStatusError
			return err
		}
	}

	b.StatusValue = ProviderStatusDestroyed
	return nil
}
