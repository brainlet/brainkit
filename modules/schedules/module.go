package schedules

import (
	"fmt"

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

	// Registry-path factories build without a Store. Fall back to
	// the shared KitStore so schedules survive restart by default.
	// Callers that want ephemeral scheduling can pass Config{Store:
	// nil} via brainkit.Config.Modules directly.
	if m.cfg.Store == nil {
		if ks := k.Store(); ks != nil {
			m.cfg.Store = ks
		}
	}

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
// cfg.Store nil so Init falls back to the shared KitStore.
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := Config{}
	if y.Path != "" {
		store, err := brainkit.NewSQLiteStore(y.Path)
		if err != nil {
			return nil, fmt.Errorf("schedules: open store %q: %w", y.Path, err)
		}
		cfg.Store = store
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "schedules",
		Status:  brainkit.ModuleStatusBeta,
		Summary: "Persisted cron + one-shot scheduling with multi-replica claim.",
	}
}

func init() { brainkit.RegisterModule("schedules", Factory{}) }
