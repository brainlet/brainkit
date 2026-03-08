// Ported from: packages/core/src/workspace/filesystem/mastra-filesystem.ts
package filesystem

import (
	"sync"
)

// =============================================================================
// MastraFilesystem Base
// =============================================================================

// Logger is a minimal logger interface for filesystem providers.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// noopLogger is a logger that discards all output.
type noopLogger struct{}

func (l *noopLogger) Debug(_ string, _ ...interface{}) {}
func (l *noopLogger) Info(_ string, _ ...interface{})  {}
func (l *noopLogger) Warn(_ string, _ ...interface{})  {}
func (l *noopLogger) Error(_ string, _ ...interface{}) {}

// MastraFilesystemOptions holds options for the MastraFilesystem base.
type MastraFilesystemOptions struct {
	// Name is the component name for logging.
	Name string
	// Logger is an optional logger instance.
	Logger Logger
	// OnInit is called after the filesystem is initialized.
	OnInit func() error
	// OnDestroy is called before the filesystem is destroyed.
	OnDestroy func() error
}

// MastraFilesystem is the abstract base for filesystem providers with logger
// integration and lifecycle management.
//
// Subclasses override OnInit() and OnDestroy() for their setup/teardown.
// Callers use Init() and Destroy() which add status tracking and
// race-condition safety.
//
// In Go, this is implemented as a composable struct rather than an abstract class.
type MastraFilesystem struct {
	mu            sync.Mutex
	status        ProviderStatus
	logger        Logger
	onInitHook    func() error
	onDestroyHook func() error
	initOnce      sync.Once
	destroyOnce   sync.Once
}

// NewMastraFilesystem creates a new MastraFilesystem base.
func NewMastraFilesystem(opts MastraFilesystemOptions) MastraFilesystem {
	l := Logger(&noopLogger{})
	if opts.Logger != nil {
		l = opts.Logger
	}
	return MastraFilesystem{
		status:        ProviderStatusPending,
		logger:        l,
		onInitHook:    opts.OnInit,
		onDestroyHook: opts.OnDestroy,
	}
}

// Status returns the current provider status.
func (mf *MastraFilesystem) GetStatus() ProviderStatus {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	return mf.status
}

// SetLogger updates the logger instance.
func (mf *MastraFilesystem) SetLogger(logger Logger) {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	mf.logger = logger
}

// GetLogger returns the current logger.
func (mf *MastraFilesystem) GetLogger() Logger {
	mf.mu.Lock()
	defer mf.mu.Unlock()
	return mf.logger
}

// Init initializes the filesystem with status tracking.
// Race-condition safe — concurrent calls are serialized.
func (mf *MastraFilesystem) Init() error {
	mf.mu.Lock()
	if mf.status == ProviderStatusReady {
		mf.mu.Unlock()
		return nil
	}
	mf.status = ProviderStatusInitializing
	mf.mu.Unlock()

	var initErr error
	mf.initOnce.Do(func() {
		if mf.onInitHook != nil {
			initErr = mf.onInitHook()
		}
	})

	mf.mu.Lock()
	defer mf.mu.Unlock()
	if initErr != nil {
		mf.status = ProviderStatusError
		return initErr
	}
	mf.status = ProviderStatusReady
	return nil
}

// Destroy destroys the filesystem with status tracking.
// Race-condition safe — concurrent calls are serialized.
func (mf *MastraFilesystem) Destroy() error {
	mf.mu.Lock()
	if mf.status == ProviderStatusDestroyed {
		mf.mu.Unlock()
		return nil
	}
	mf.status = ProviderStatusDestroying
	mf.mu.Unlock()

	var destroyErr error
	mf.destroyOnce.Do(func() {
		if mf.onDestroyHook != nil {
			destroyErr = mf.onDestroyHook()
		}
	})

	mf.mu.Lock()
	defer mf.mu.Unlock()
	if destroyErr != nil {
		mf.status = ProviderStatusError
		return destroyErr
	}
	mf.status = ProviderStatusDestroyed
	return nil
}

// EnsureReady checks that the filesystem is ready. If pending, calls Init().
func (mf *MastraFilesystem) EnsureReady() error {
	mf.mu.Lock()
	s := mf.status
	mf.mu.Unlock()

	switch s {
	case ProviderStatusReady:
		return nil
	case ProviderStatusPending:
		return mf.Init()
	case ProviderStatusDestroyed:
		return &filesystemNotReadyError{id: "filesystem"}
	default:
		return &filesystemNotReadyError{id: "filesystem"}
	}
}

type filesystemNotReadyError struct {
	id string
}

func (e *filesystemNotReadyError) Error() string {
	return "Filesystem " + e.id + " is not ready. Call Init() first or use EnsureReady()."
}
