package discovery

import (
	"fmt"

	"github.com/brainlet/brainkit"
	"github.com/google/uuid"
)

// Module is the brainkit.Module form of peer discovery. It owns a Provider
// (bus or static) and self-registers the local peer on Kit.Init so other
// kits on the same cluster can find it. The bus surface (peers.list /
// peers.resolve) now lives in modules/topology — callers that want the
// bus commands must pair discovery with topology (passing this module's
// Provider() as topology.Config.Discovery).
type Module struct {
	cfg      ModuleConfig
	provider Provider
}

// NewModule builds the discovery Module from config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg ModuleConfig) *Module { return &Module{cfg: cfg} }

func (m *Module) Name() string { return "discovery" }

func (m *Module) Init(k *brainkit.Kit) error {
	switch m.cfg.Type {
	case "":
		return nil
	case "static":
		m.provider = NewStaticFromConfig(m.cfg.StaticPeers)
	case "bus":
		m.provider = NewBus(BusConfig{
			Transport: k.PresenceTransport(),
			Heartbeat: m.cfg.Heartbeat,
			TTL:       m.cfg.TTL,
		})
	default:
		return fmt.Errorf("discovery: unknown type %q (want \"static\" or \"bus\")", m.cfg.Type)
	}

	// Name must be unique across the presence cluster — two kits sharing a
	// name make Bus.handleMessage's self-skip treat each other's announcements
	// as their own. When the caller doesn't pin a Name, generate a per-
	// instance UUID (matches the pre-module wiring, which used Node.nodeID).
	name := m.cfg.Name
	if name == "" {
		name = uuid.NewString()
	}
	return m.provider.Register(Peer{Name: name, Namespace: k.Namespace()})
}

func (m *Module) Close() error {
	if m.provider == nil {
		return nil
	}
	return m.provider.Close()
}

// Provider exposes the underlying presence provider so topology (or any
// other downstream consumer) can read the live peer set. Returns nil
// when Type is empty.
func (m *Module) Provider() Provider { return m.provider }
