package audit

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/brainlet/brainkit/internal/closejob"
	bkmodule "github.com/brainlet/brainkit/module"
)

// Module is the bkmodule.Module form of the audit log. Mount attaches a
// store to the core Recorder (so every subsystem's Record calls start
// persisting) and registers the audit.query / audit.stats / audit.prune
// bus commands.
type Module struct {
	closeMu         sync.Mutex
	mu              sync.RWMutex
	cfg             Config
	domain          *domain
	storeLease      bkmodule.Handle
	verbosityLease  bkmodule.Handle
	storeAttached   atomic.Bool
	verboseAttached atomic.Bool
	closing         atomic.Bool
	storeClosing    atomic.Bool
	storeCloseJob   closejob.Job
}

type contextCloseableStore interface {
	CloseContext(context.Context) error
}

// NewModule builds the audit module from config.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) ID() string { return "audit" }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	leaseStore, err := bkmodule.RequireCapability[bkmodule.LeaseFunc[Store]](host, bkmodule.CapabilityAuditStoreLease)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	leaseVerbosity, err := bkmodule.RequireCapability[bkmodule.LeaseFunc[Verbosity]](host, bkmodule.CapabilityAuditVerbosityLease)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	m.domain = newDomain(m.cfg.Store)

	if m.cfg.Store != nil {
		handle, err := leaseStore(ctx, m.cfg.Store)
		if err != nil {
			return fmt.Errorf("audit: %w", err)
		}
		m.mu.Lock()
		m.storeLease = handle
		m.storeAttached.Store(true)
		m.mu.Unlock()
	}
	if m.cfg.Verbose {
		handle, err := leaseVerbosity(ctx, VerbosityVerbose)
		if err != nil {
			return errors.Join(fmt.Errorf("audit: %w", err), m.CloseContext(ctx))
		}
		m.mu.Lock()
		m.verbosityLease = handle
		m.verboseAttached.Store(true)
		m.mu.Unlock()
	}
	host.Scope().Defer(func(closeCtx context.Context) error { return m.CloseContext(closeCtx) })
	if m.cfg.Store != nil {
		host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindStore, "audit.store", "Attached persistent audit event store."))
	}
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "audit", func() any {
			return m.DebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("audit: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.domain.Query)); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.domain.Stats)); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(m.domain.Prune)); err != nil {
		return err
	}
	return nil
}

func (m *Module) Close() error {
	return m.CloseContext(context.Background())
}

func (m *Module) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.closeMu.Lock()
	defer m.closeMu.Unlock()
	m.closing.Store(true)
	defer m.closing.Store(false)
	var err error

	m.mu.RLock()
	verbosityLease := m.verbosityLease
	storeLease := m.storeLease
	ownedStore := m.cfg.Store
	ownsStore := m.cfg.OwnStore
	m.mu.RUnlock()

	if verbosityLease != nil {
		if closeErr := verbosityLease.Close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
		} else {
			m.mu.Lock()
			m.verbosityLease = nil
			m.verboseAttached.Store(false)
			m.mu.Unlock()
		}
	}
	if storeLease != nil {
		if closeErr := storeLease.Close(ctx); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		m.mu.Lock()
		m.storeLease = nil
		m.storeAttached.Store(false)
		ownedStore = m.cfg.Store
		ownsStore = m.cfg.OwnStore
		m.mu.Unlock()
	}
	if ownsStore && ownedStore != nil {
		done := m.storeCloseJob.Start(func() error {
			m.storeClosing.Store(true)
			defer m.storeClosing.Store(false)
			closeErr := closeAuditStore(ctx, ownedStore)
			if closeErr == nil {
				m.mu.Lock()
				if m.cfg.Store == ownedStore {
					m.cfg.Store = nil
					m.cfg.OwnStore = false
				}
				m.mu.Unlock()
			}
			return closeErr
		})
		closeErr := m.storeCloseJob.Wait(ctx, done)
		if closeErr != nil {
			return errors.Join(err, closeErr)
		}
	}
	return err
}

func closeAuditStore(ctx context.Context, store Store) error {
	if closer, ok := store.(contextCloseableStore); ok {
		return closer.CloseContext(ctx)
	}
	return store.Close()
}
