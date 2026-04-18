package audit

import (
	"github.com/brainlet/brainkit"
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
