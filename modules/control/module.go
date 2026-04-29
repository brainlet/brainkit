// Package control owns runtime control bus commands as a hot-mountable Kit
// module.
package control

import (
	"context"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
)

// Module exposes kit.set-draining and cluster.peers. Construct via New and
// include in brainkit.Config.Modules when the runtime should expose these
// control-plane commands over the bus.
type Module struct {
	control bkmodule.RuntimeControl
}

// New creates the control module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "control" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers runtime control command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	control, err := bkmodule.RequireCapability[bkmodule.RuntimeControl](host, bkmodule.CapabilityRuntimeControl)
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	m.control = control
	host.Scope().Defer(func(context.Context) error {
		m.control = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.SetDraining),
		bkmodule.Command(m.ClusterPeers),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close detaches the module from Kit capabilities. Command handles are owned by
// the module scope, so unmounting unregisters them.
func (m *Module) Close() error {
	m.control = nil
	return nil
}

// Factory is the registered ModuleFactory for control.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the control module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "control",
		Status:  bkmodule.StatusStable,
		Summary: "Runtime control bus commands (kit.set-draining, cluster.peers).",
	}
}

func init() { bkmodule.Register("control", Factory{}) }

// SetDraining handles kit.set-draining.
func (m *Module) SetDraining(_ context.Context, req KitSetDrainingMsg) (*KitSetDrainingResp, error) {
	m.control.SetDraining(req.Draining)
	return &KitSetDrainingResp{Draining: req.Draining}, nil
}

// ClusterPeers handles cluster.peers.
func (m *Module) ClusterPeers(ctx context.Context, _ ClusterPeersMsg) (*ClusterPeersResp, error) {
	peers, err := m.control.ClusterPeers(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]ClusterPeerInfo, 0, len(peers))
	for _, peer := range peers {
		resp = append(resp, ClusterPeerInfo{
			ClusterID: peer.ClusterID,
			RuntimeID: peer.RuntimeID,
			Namespace: peer.Namespace,
			CallerID:  peer.CallerID,
			StartedAt: peer.StartedAt,
		})
	}
	return &ClusterPeersResp{Peers: resp}, nil
}
