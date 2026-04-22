package discovery

import (
	"fmt"
	"time"

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

// PeerYAML is a static peer entry (static mode or as a supplement to
// bus mode).
type PeerYAML struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Address   string            `yaml:"address"`
	Meta      map[string]string `yaml:"meta"`
}

// YAML is the config shape decoded by the registry factory.
//
// type: "static" | "bus" | "" (disabled — omit the section to
// disable; presence in YAML with type="" is still accepted but is a
// no-op and logs at boot).
type YAML struct {
	Type      string        `yaml:"type"`
	Name      string        `yaml:"name"`
	Heartbeat time.Duration `yaml:"heartbeat"`
	TTL       time.Duration `yaml:"ttl"`
	Peers     []PeerYAML    `yaml:"peers"`
}

// Factory is the registered ModuleFactory for discovery.
type Factory struct{}

// Build decodes YAML into a discovery.ModuleConfig and returns the
// module. The provider itself is wired in Init when PresenceTransport
// is live.
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := ModuleConfig{
		Type:      y.Type,
		Name:      y.Name,
		Heartbeat: y.Heartbeat,
		TTL:       y.TTL,
	}
	for _, p := range y.Peers {
		cfg.StaticPeers = append(cfg.StaticPeers, PeerConfig{
			Name:      p.Name,
			Namespace: p.Namespace,
			Address:   p.Address,
			Meta:      p.Meta,
		})
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "discovery",
		Status:  brainkit.ModuleStatusBeta,
		Summary: "Peer discovery: static list or bus-announced presence.",
	}
}

func init() { brainkit.RegisterModule("discovery", Factory{}) }
