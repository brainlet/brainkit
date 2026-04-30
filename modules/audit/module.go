package audit

import (
	"context"
	"fmt"
	"path/filepath"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/audit/auditmsg"
	"github.com/brainlet/brainkit/modules/audit/stores"
)

// Module is the brainkit.Module form of the audit log. Mount attaches a
// store to the core Recorder (so every subsystem's Record calls start
// persisting) and registers the audit.query / audit.stats / audit.prune
// bus commands.
type Module struct {
	cfg    Config
	domain *domain
}

// NewModule builds the audit module from config.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) ID() string { return "audit" }

func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	setStore, err := bkmodule.RequireCapability[func(Store)](host, bkmodule.CapabilitySetAuditStore)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	setVerbosity, err := bkmodule.RequireCapability[func(Verbosity)](host, bkmodule.CapabilitySetAuditVerbosity)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	host.Scope().Defer(func(context.Context) error {
		setStore(nil)
		if m.cfg.Verbose {
			setVerbosity(VerbosityNormal)
		}
		return m.closeStore()
	})
	m.attach(auditCoreFuncs{setStore: setStore, setVerbosity: setVerbosity})
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

type auditCore interface {
	SetAuditStore(Store)
	SetAuditVerbosity(Verbosity)
}

type auditCoreFuncs struct {
	setStore     func(Store)
	setVerbosity func(Verbosity)
}

func (f auditCoreFuncs) SetAuditStore(store Store) { f.setStore(store) }

func (f auditCoreFuncs) SetAuditVerbosity(verbosity Verbosity) { f.setVerbosity(verbosity) }

func (m *Module) attach(core auditCore) {
	m.domain = newDomain(m.cfg.Store)

	// Attach the store to core's Recorder so writes start persisting.
	core.SetAuditStore(m.cfg.Store)
	if m.cfg.Verbose {
		core.SetAuditVerbosity(VerbosityVerbose)
	}
}

func (m *Module) Close() error {
	return m.closeStore()
}

func (m *Module) closeStore() error {
	if m.cfg.OwnStore && m.cfg.Store != nil {
		err := m.cfg.Store.Close()
		m.cfg.Store = nil
		return err
	}
	return nil
}

// YAML is the config shape decoded by the registry factory. Empty
// Path falls back to `<FSRoot>/audit.db`. Other backends (postgres,
// in-memory) can be selected via Type.
type YAML struct {
	Type             string `yaml:"type"`
	Path             string `yaml:"path"`
	ConnectionString string `yaml:"connection_string"`
	Verbose          bool   `yaml:"verbose"`
}

// Factory is the registered ModuleFactory for audit.
type Factory struct{}

// Build opens the audit store and returns the module. OwnStore is
// always true — the factory opened it, the factory's module closes it.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	store, err := openAuditStore(ctx, y)
	if err != nil {
		return nil, err
	}
	return NewModule(Config{Store: store, Verbose: y.Verbose, OwnStore: true}), nil
}

func openAuditStore(ctx bkmodule.BuildContext, y YAML) (Store, error) {
	switch y.Type {
	case "", "sqlite":
		path := y.Path
		if path == "" {
			path = filepath.Join(ctx.FSRoot, "audit.db")
		}
		s, err := stores.NewSQLite(path)
		if err != nil {
			return nil, fmt.Errorf("audit: open sqlite %q: %w", path, err)
		}
		return s, nil
	case "postgres":
		s, err := stores.NewPostgres(y.ConnectionString)
		if err != nil {
			return nil, fmt.Errorf("audit: open postgres: %w", err)
		}
		return s, nil
	default:
		return nil, fmt.Errorf("audit: unknown store type %q (want sqlite or postgres)", y.Type)
	}
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "audit",
		Status:  bkmodule.StatusStable,
		Summary: "Persistent audit log with query/stats/prune bus commands.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[auditmsg.AuditPruneMsg, auditmsg.AuditPruneResp](),
			bkmodule.CommandMessage[auditmsg.AuditQueryMsg, auditmsg.AuditQueryResp](),
			bkmodule.CommandMessage[auditmsg.AuditStatsMsg, auditmsg.AuditStatsResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[func(Store)](bkmodule.CapabilitySetAuditStore),
			bkmodule.RequiredCapabilityOf[func(Verbosity)](bkmodule.CapabilitySetAuditVerbosity),
		},
	}
}

func init() { bkmodule.Register("audit", Factory{}) }
