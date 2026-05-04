package schedules

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/closejob"
	internalstore "github.com/brainlet/brainkit/internal/store"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
)

// Module is the bkmodule.Module form of persisted scheduling. It wires a
// Scheduler into the Kit's QuickJS bridges and registers the schedules.*
// bus commands at mount time.
type Module struct {
	mu      sync.RWMutex
	closeMu sync.Mutex

	cfg       Config
	scheduler *Scheduler

	scheduleHandlerLease       bkmodule.Handle
	scheduleHandlerLeaseActive atomic.Bool
	closing                    atomic.Bool
	storeClosing               atomic.Bool
	storeCloseJob              closejob.Job
}

type closeableStore interface {
	Close() error
}

type contextCloseableStore interface {
	CloseContext(context.Context) error
}

// NewModule builds the schedules module from config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) ID() string { return "schedules" }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	m.mu.Lock()
	if m.cfg.Store == nil {
		if store, ok := bkmodule.Capability[Store](host, bkmodule.CapabilityKitStore); ok {
			m.cfg.Store = store
		}
	}
	store := m.cfg.Store
	m.mu.Unlock()

	isDraining := func() bool {
		if control, ok := bkmodule.Capability[bkmodule.RuntimeControl](host, bkmodule.CapabilityRuntimeControl); ok {
			return control.IsDraining()
		}
		return false
	}
	scheduler := newScheduler(
		host.Messages(),
		store,
		host.Logger(),
		host.Commands().Has,
		isDraining,
		func(err error) { host.Logger().Error("schedules error", "error", err) },
	)
	m.mu.Lock()
	m.scheduler = scheduler
	m.mu.Unlock()
	host.Scope().Defer(func(closeCtx context.Context) error { return m.CloseContext(closeCtx) })
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindScheduler, "schedules.scheduler", "Persisted cron and one-shot scheduler."))

	leaseScheduleHandler, err := bkmodule.RequireCapability[bkmodule.LeaseFunc[types.ScheduleHandler]](host, bkmodule.CapabilityScheduleHandlerLease)
	if err != nil {
		return fmt.Errorf("schedules: %w", err)
	}
	scheduleHandlerLease, err := leaseScheduleHandler(ctx, scheduler)
	if err != nil {
		return fmt.Errorf("schedules: %w", err)
	}
	m.mu.Lock()
	m.scheduleHandlerLease = scheduleHandlerLease
	m.scheduleHandlerLeaseActive.Store(true)
	m.mu.Unlock()
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindHook, "schedule-handler", "Core schedule dispatch hook."))

	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "schedules", func() any {
			return m.DebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("schedules: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}

	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req schedulemsg.ScheduleCreateMsg) (*schedulemsg.ScheduleCreateResp, error) {
		return m.handleCreate(ctx, req)
	})); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req schedulemsg.ScheduleCancelMsg) (*schedulemsg.ScheduleCancelResp, error) {
		return m.handleCancel(ctx, req)
	})); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req schedulemsg.ScheduleListMsg) (*schedulemsg.ScheduleListResp, error) {
		return m.handleList(ctx, req)
	})); err != nil {
		return err
	}

	scheduler.Restore()
	return nil
}

func (m *Module) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.CloseContext(ctx)
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
	lease := m.scheduleHandlerLease
	scheduler := m.scheduler
	store := m.cfg.Store
	ownsStore := m.cfg.ownsStore
	m.mu.RUnlock()

	leaseClosed := lease == nil
	if lease != nil {
		if closeErr := lease.Close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
		} else {
			leaseClosed = true
			m.mu.Lock()
			m.scheduleHandlerLease = nil
			m.scheduleHandlerLeaseActive.Store(false)
			store = m.cfg.Store
			ownsStore = m.cfg.ownsStore
			m.mu.Unlock()
		}
	} else {
		m.scheduleHandlerLeaseActive.Store(false)
	}

	schedulerClosed := true
	if scheduler != nil {
		if closeErr := scheduler.CloseContext(ctx); closeErr != nil {
			schedulerClosed = false
			err = errors.Join(err, closeErr)
		}
	}

	if ownsStore && store != nil && leaseClosed && schedulerClosed {
		done := m.storeCloseJob.Start(func() error {
			m.storeClosing.Store(true)
			defer m.storeClosing.Store(false)
			closeErr := closeScheduleStore(ctx, store)
			if closeErr == nil {
				m.mu.Lock()
				if m.cfg.Store == store {
					m.cfg.Store = nil
					m.cfg.ownsStore = false
				}
				m.mu.Unlock()
			}
			return closeErr
		})
		if closeErr := m.storeCloseJob.Wait(ctx, done); closeErr != nil {
			return errors.Join(err, closeErr)
		}
	}
	return err
}

func closeScheduleStore(ctx context.Context, store Store) error {
	if closer, ok := store.(contextCloseableStore); ok {
		return closer.CloseContext(ctx)
	}
	if closer, ok := store.(closeableStore); ok {
		return closer.Close()
	}
	return nil
}

// compile-time assertion that Scheduler satisfies the engine-side interface.
var _ types.ScheduleHandler = (*Scheduler)(nil)

// YAML is the config shape decoded by the registry factory.
//
// `path`, when set, opens a dedicated SQLite database at that path.
// When empty, schedules share the Kit's main store (kit.db).
type YAML struct {
	Path string `yaml:"path"`
}

// Factory is the registered ModuleFactory for schedules.
type Factory struct{}

// Build opens the dedicated store when Path is set, otherwise leaves
// cfg.Store nil so Mount falls back to the shared KitStore.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := Config{}
	if y.Path != "" {
		store, err := internalstore.NewSQLiteKitStore(y.Path)
		if err != nil {
			return nil, fmt.Errorf("schedules: open store %q: %w", y.Path, err)
		}
		cfg.Store = store
		cfg.ownsStore = true
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "schedules",
		Status:  bkmodule.StatusBeta,
		Summary: "Persisted cron + one-shot scheduling with multi-replica claim.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[schedulemsg.ScheduleCancelMsg, schedulemsg.ScheduleCancelResp](),
			bkmodule.CommandMessage[schedulemsg.ScheduleCreateMsg, schedulemsg.ScheduleCreateResp](),
			bkmodule.CommandMessage[schedulemsg.ScheduleListMsg, schedulemsg.ScheduleListResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.OptionalCapabilityOf[Store](bkmodule.CapabilityKitStore),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
			bkmodule.OptionalCapabilityOf[bkmodule.RuntimeControl](bkmodule.CapabilityRuntimeControl),
			bkmodule.RequiredCapabilityOf[bkmodule.LeaseFunc[types.ScheduleHandler]](bkmodule.CapabilityScheduleHandlerLease),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindScheduler, "schedules.scheduler", "Persisted cron and one-shot scheduler."),
			bkmodule.Resource(bkmodule.ResourceKindHook, "schedule-handler", "Core schedule dispatch hook."),
		},
	}
}

func init() { bkmodule.Register("schedules", Factory{}) }
