package schedules

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/types"
)

// Module is the brainkit.Module form of persisted scheduling. It wires a
// Scheduler into the Kit's QuickJS bridges and registers the schedule.*
// bus commands at Init time.
type Module struct {
	cfg       Config
	scheduler *Scheduler
	kit       *brainkit.Kit
}

// NewModule builds the schedules module from config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) Name() string { return "schedules" }

func (m *Module) Init(k *brainkit.Kit) error {
	m.kit = k
	m.scheduler = newScheduler(
		k,
		m.cfg.Store,
		k.Logger(),
		k.HasCommand,
		k.IsDraining,
		func(err error) { k.ReportError(err, brainkit.ErrorContext{Operation: "schedules", Component: "module"}) },
	)
	k.SetScheduleHandler(m.scheduler)

	k.RegisterCommand(brainkit.Command(m.handleCreate))
	k.RegisterCommand(brainkit.Command(m.handleCancel))
	k.RegisterCommand(brainkit.Command(m.handleList))

	// Restore persisted schedules if a store is configured.
	m.scheduler.Restore()
	return nil
}

func (m *Module) Close() error {
	if m.scheduler != nil {
		_ = m.scheduler.Close()
	}
	if m.kit != nil {
		m.kit.SetScheduleHandler(nil)
	}
	return nil
}

// compile-time assertion that Scheduler satisfies the engine-side interface.
var _ types.ScheduleHandler = (*Scheduler)(nil)
