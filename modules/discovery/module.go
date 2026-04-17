package discovery

import (
	"context"
	"fmt"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/google/uuid"
)

// Module is the brainkit.Module form of peer discovery. It owns a Provider
// (bus or static), self-registers on Kit.Init, and exposes peers.list /
// peers.resolve bus commands.
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
	if err := m.provider.Register(Peer{Name: name, Namespace: k.Namespace()}); err != nil {
		return err
	}

	k.RegisterCommand(brainkit.Command(m.handleList))
	k.RegisterCommand(brainkit.Command(m.handleResolve))
	return nil
}

func (m *Module) Close() error {
	if m.provider == nil {
		return nil
	}
	return m.provider.Close()
}

func (m *Module) handleList(ctx context.Context, req sdk.PeersListMsg) (*sdk.PeersListResp, error) {
	if m.provider == nil {
		return &sdk.PeersListResp{Peers: []sdk.PeerInfo{}}, nil
	}
	peers, err := m.provider.Browse()
	if err != nil {
		return nil, err
	}
	infos := make([]sdk.PeerInfo, len(peers))
	for i, p := range peers {
		infos[i] = sdk.PeerInfo{Name: p.Name, Namespace: p.Namespace, Address: p.Address, Meta: p.Meta}
	}
	namespaces, _ := m.provider.BrowseNamespaces()
	return &sdk.PeersListResp{Peers: infos, Namespaces: namespaces}, nil
}

func (m *Module) handleResolve(ctx context.Context, req sdk.PeersResolveMsg) (*sdk.PeersResolveResp, error) {
	if m.provider == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "discovery"}
	}
	addr, err := m.provider.Resolve(req.Name)
	if err != nil {
		return nil, err
	}
	return &sdk.PeersResolveResp{Namespace: addr}, nil
}
