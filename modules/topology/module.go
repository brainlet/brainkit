package topology

import (
	"context"
	"fmt"
	"sync"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/discovery"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Peer is a known cross-kit endpoint. Shared shape with
// discovery.Peer — when the topology is fed by a discovery.Provider,
// the two slices contain the same values.
type Peer = discovery.Peer

// ProviderSource is the narrow surface topology needs from a
// presence-aware subsystem. modules/discovery.Module satisfies it
// (via its exported Provider() accessor) — topology calls this lazily
// at request time, so wiring works regardless of module mount order
// when Config.Discovery is supplied directly.
type ProviderSource interface {
	Provider() discovery.Provider
}

// Config builds a topology Module.
type Config struct {
	// Peers is the static peer list used when Discovery is nil. When
	// Discovery is set, Peers supplements the discovered list (static
	// entries override the dynamic lookup for the same name).
	Peers []Peer

	// Discovery, when non-nil, lets the topology module read live
	// peers from a presence-aware source. Typically a
	// *discovery.Module returned by discovery.NewModule(...); pass
	// the module directly (topology calls Provider() lazily).
	Discovery ProviderSource

	// UseDiscovery, when true, tells Mount to resolve the discovery
	// module via k.Module("discovery") and use it as the provider
	// source. Required for the registry path, where a sibling module
	// isn't available at Build time.
	UseDiscovery bool
}

// Module is the brainkit.Module form of cross-kit topology. It owns
// the peers.list / peers.resolve bus commands and provides Resolve()
// for WithCallTo routing. Discovery is optional — without it the
// module works from the static Peers slice only.
type Module struct {
	cfg Config

	mu     sync.RWMutex
	static map[string]Peer
}

// NewModule builds the topology Module from Config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg Config) *Module {
	m := &Module{cfg: cfg, static: make(map[string]Peer, len(cfg.Peers))}
	for _, p := range cfg.Peers {
		if p.Name != "" {
			m.static[p.Name] = p
		}
	}
	return m
}

func (m *Module) ID() string              { return "topology" }
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusBeta }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	if m.cfg.UseDiscovery && m.cfg.Discovery == nil {
		value, err := host.Capabilities().Require("discovery.provider")
		if err != nil {
			return fmt.Errorf("topology: use_discovery is true but no discovery.provider capability is mounted")
		}
		switch src := value.(type) {
		case ProviderSource:
			m.cfg.Discovery = src
		case discovery.Provider:
			m.cfg.Discovery = providerSourceFunc(func() discovery.Provider { return src })
		default:
			return fmt.Errorf("topology: discovery.provider capability is %T, want topology.ProviderSource or discovery.Provider", value)
		}
	}

	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req PeersListMsg) (*PeersListResp, error) {
		return m.handleList(ctx, req)
	})); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req PeersResolveMsg) (*PeersResolveResp, error) {
		return m.handleResolve(ctx, req)
	})); err != nil {
		return err
	}
	return nil
}

func (m *Module) Close() error { return nil }

type providerSourceFunc func() discovery.Provider

func (f providerSourceFunc) Provider() discovery.Provider { return f() }

// provider returns the live discovery.Provider the module reads from
// (nil when no source is wired or the source hasn't initialized yet).
func (m *Module) provider() discovery.Provider {
	if m.cfg.Discovery == nil {
		return nil
	}
	return m.cfg.Discovery.Provider()
}

// Resolve maps a peer Name to its Namespace. Unknown names return a
// NotFoundError so callers (WithCallTo) can surface a clear message
// instead of silently routing to an unrouted namespace.
func (m *Module) Resolve(name string) (string, error) {
	m.mu.RLock()
	p, ok := m.static[name]
	m.mu.RUnlock()
	if ok {
		return p.Namespace, nil
	}

	if p := m.provider(); p != nil {
		if ns, err := p.Resolve(name); err == nil {
			return ns, nil
		}
	}
	return "", &sdkerrors.NotFoundError{Resource: "peer", Name: name}
}

// Peers returns every known peer — static entries plus whatever the
// discovery provider currently sees. Static entries take precedence
// on name collisions.
func (m *Module) Peers() []Peer {
	m.mu.RLock()
	out := make([]Peer, 0, len(m.static))
	seen := make(map[string]struct{}, len(m.static))
	for _, p := range m.static {
		out = append(out, p)
		seen[p.Name] = struct{}{}
	}
	m.mu.RUnlock()

	if p := m.provider(); p != nil {
		if peers, err := p.Browse(); err == nil {
			for _, pr := range peers {
				if _, dup := seen[pr.Name]; dup {
					continue
				}
				out = append(out, pr)
			}
		}
	}
	return out
}

// Namespaces returns every unique namespace across static + discovered
// peers. Useful for broadcast-style routing.
func (m *Module) Namespaces() []string {
	seen := make(map[string]struct{})
	for _, p := range m.Peers() {
		if p.Namespace != "" {
			seen[p.Namespace] = struct{}{}
		}
	}
	if p := m.provider(); p != nil {
		if nss, err := p.BrowseNamespaces(); err == nil {
			for _, ns := range nss {
				seen[ns] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for ns := range seen {
		out = append(out, ns)
	}
	return out
}

func (m *Module) handleList(_ context.Context, _ PeersListMsg) (*PeersListResp, error) {
	peers := m.Peers()
	infos := make([]PeerInfo, len(peers))
	for i, p := range peers {
		infos[i] = PeerInfo{Name: p.Name, Namespace: p.Namespace, Address: p.Address, Meta: p.Meta}
	}
	return &PeersListResp{Peers: infos, Namespaces: m.Namespaces()}, nil
}

func (m *Module) handleResolve(_ context.Context, req PeersResolveMsg) (*PeersResolveResp, error) {
	ns, err := m.Resolve(req.Name)
	if err != nil {
		return nil, fmt.Errorf("topology.resolve %q: %w", req.Name, err)
	}
	return &PeersResolveResp{Namespace: ns}, nil
}

// PeerYAML is a single static peer entry.
type PeerYAML struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Address   string            `yaml:"address"`
	Meta      map[string]string `yaml:"meta"`
}

// YAML is the config shape decoded by the registry factory.
//
// `use_discovery: true` wires the discovery module (required in YAML
// alongside) as the live peer source.
type YAML struct {
	Peers        []PeerYAML `yaml:"peers"`
	UseDiscovery bool       `yaml:"use_discovery"`
}

// Factory is the registered ModuleFactory for topology.
type Factory struct{}

// Build decodes YAML into a topology.Config. Discovery wiring (when
// requested) is deferred to Mount, where the Kit can resolve the
// sibling module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := Config{UseDiscovery: y.UseDiscovery}
	for _, p := range y.Peers {
		cfg.Peers = append(cfg.Peers, Peer{
			Name:      p.Name,
			Namespace: p.Namespace,
			Address:   p.Address,
			Meta:      p.Meta,
		})
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "topology",
		Status:  bkmodule.StatusBeta,
		Summary: "Cross-kit routing — static peers + optional discovery feed.",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[PeersListMsg, PeersListResp](),
			bkmodule.CommandMessage[PeersResolveMsg, PeersResolveResp](),
		},
	}
}

func init() { bkmodule.Register("topology", Factory{}) }
