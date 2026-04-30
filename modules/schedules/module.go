package schedules

import (
	"context"
	"fmt"

	internalstore "github.com/brainlet/brainkit/internal/store"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
)

// Module is the brainkit.Module form of persisted scheduling. It wires a
// Scheduler into the Kit's QuickJS bridges and registers the schedule.*
// bus commands at mount time.
type Module struct {
	cfg       Config
	scheduler *Scheduler
}

// NewModule builds the schedules module from config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) ID() string { return "schedules" }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	if m.cfg.Store == nil {
		if store, ok := host.Store().(Store); ok {
			m.cfg.Store = store
		}
	}

	isDraining := func() bool {
		if draining, ok := host.Runtime().(interface{ IsDraining() bool }); ok {
			return draining.IsDraining()
		}
		return false
	}
	m.scheduler = newScheduler(
		host.Runtime(),
		m.cfg.Store,
		host.Logger(),
		host.Commands().Has,
		isDraining,
		func(err error) { host.Logger().Error("schedules error", "error", err) },
	)
	host.Scope().Defer(func(context.Context) error { return m.scheduler.Close() })

	setScheduleHandler, err := bkmodule.RequireCapability[func(types.ScheduleHandler)](host, bkmodule.CapabilitySetScheduleHandler)
	if err != nil {
		return fmt.Errorf("schedules: %w", err)
	}
	setScheduleHandler(m.scheduler)
	host.Scope().Defer(func(context.Context) error {
		setScheduleHandler(nil)
		return nil
	})

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

	m.scheduler.Restore()
	return nil
}

func (m *Module) Close() error {
	if m.scheduler != nil {
		_ = m.scheduler.Close()
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
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
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
			bkmodule.RequiredCapabilityOf[func(types.ScheduleHandler)](bkmodule.CapabilitySetScheduleHandler),
		},
	}
}

func init() { bkmodule.Register("schedules", Factory{}) }
