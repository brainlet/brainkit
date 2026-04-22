package audit

import (
	"fmt"
	"path/filepath"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/audit/stores"
)

// Module is the brainkit.Module form of the audit log. Init attaches a
// store to the core Recorder (so every subsystem's Record calls start
// persisting) and registers the audit.query / audit.stats / audit.prune
// bus commands.
type Module struct {
	cfg    Config
	kit    *brainkit.Kit
	domain *domain
}

// NewModule builds the audit module from config.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) Name() string { return "audit" }

func (m *Module) Init(k *brainkit.Kit) error {
	m.kit = k
	m.domain = newDomain(m.cfg.Store)

	// Attach the store to core's Recorder so writes start persisting.
	k.SetAuditStore(m.cfg.Store)
	if m.cfg.Verbose {
		k.SetAuditVerbosity(VerbosityVerbose)
	}

	k.RegisterCommand(brainkit.Command(m.domain.Query))
	k.RegisterCommand(brainkit.Command(m.domain.Stats))
	k.RegisterCommand(brainkit.Command(m.domain.Prune))
	return nil
}

func (m *Module) Close() error {
	if m.kit != nil {
		m.kit.SetAuditStore(nil)
	}
	if m.cfg.OwnStore && m.cfg.Store != nil {
		return m.cfg.Store.Close()
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
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
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

func openAuditStore(ctx brainkit.ModuleContext, y YAML) (Store, error) {
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
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "audit",
		Status:  brainkit.ModuleStatusStable,
		Summary: "Persistent audit log with query/stats/prune bus commands.",
	}
}

func init() { brainkit.RegisterModule("audit", Factory{}) }
