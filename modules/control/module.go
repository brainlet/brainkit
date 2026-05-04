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
	control           bkmodule.RuntimeControl
	mountedModules    func() []bkmodule.Descriptor
	lifecycle         bkmodule.ModuleLifecycle
	lifecycleSnapshot func() bkmodule.LifecycleDebugSnapshot
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
	mountedModules, err := bkmodule.RequireCapability[func() []bkmodule.Descriptor](host, bkmodule.CapabilityMountedModules)
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	m.mountedModules = mountedModules
	lifecycle, err := bkmodule.RequireCapability[bkmodule.ModuleLifecycle](host, bkmodule.CapabilityModuleLifecycle)
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	m.lifecycle = lifecycle
	lifecycleSnapshot, err := bkmodule.RequireCapability[func() bkmodule.LifecycleDebugSnapshot](host, bkmodule.CapabilityLifecycleDebugSnapshot)
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	m.lifecycleSnapshot = lifecycleSnapshot
	host.Scope().Defer(func(context.Context) error {
		m.control = nil
		m.mountedModules = nil
		m.lifecycle = nil
		m.lifecycleSnapshot = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.SetDraining),
		bkmodule.Command(m.ClusterPeers),
		bkmodule.Command(m.Modules),
		bkmodule.Command(m.Lifecycle),
		bkmodule.Command(m.ModuleMount),
		bkmodule.Command(m.ModuleUnmount),
		bkmodule.Command(m.ModuleDescribe),
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
	m.mountedModules = nil
	m.lifecycle = nil
	m.lifecycleSnapshot = nil
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

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "control",
		Status:  bkmodule.StatusStable,
		Summary: "Runtime control bus commands (kit.set-draining, kit.modules, module lifecycle, cluster.peers).",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[ClusterPeersMsg, ClusterPeersResp](),
			bkmodule.CommandMessage[KitModuleDescribeMsg, KitModuleDescribeResp](),
			bkmodule.CommandMessage[KitModuleMountMsg, KitModuleMountResp](),
			bkmodule.CommandMessage[KitModuleUnmountMsg, KitModuleUnmountResp](),
			bkmodule.CommandMessage[KitLifecycleMsg, KitLifecycleResp](),
			bkmodule.CommandMessage[KitModulesMsg, KitModulesResp](),
			bkmodule.CommandMessage[KitSetDrainingMsg, KitSetDrainingResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[func() bkmodule.LifecycleDebugSnapshot](bkmodule.CapabilityLifecycleDebugSnapshot),
			bkmodule.RequiredCapabilityOf[bkmodule.ModuleLifecycle](bkmodule.CapabilityModuleLifecycle),
			bkmodule.RequiredCapabilityOf[func() []bkmodule.Descriptor](bkmodule.CapabilityMountedModules),
			bkmodule.RequiredCapabilityOf[bkmodule.RuntimeControl](bkmodule.CapabilityRuntimeControl),
		},
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

// Modules handles kit.modules.
func (m *Module) Modules(ctx context.Context, _ KitModulesMsg) (*KitModulesResp, error) {
	if m.mountedModules == nil {
		return nil, fmt.Errorf("control: mounted module catalog is not configured")
	}
	modules := m.mountedModules()
	preflights := make(map[string]bkmodule.ModulePreflight, len(modules))
	if m.lifecycle != nil {
		for _, desc := range modules {
			if desc.Name == "" {
				continue
			}
			preflight, err := m.lifecycle.PreflightModule(ctx, desc.Name)
			if err != nil {
				preflight = bkmodule.ModulePreflight{
					Ready:  false,
					Errors: []string{err.Error()},
				}
			}
			preflights[desc.Name] = preflight
		}
	}
	return &KitModulesResp{Modules: modules, Preflights: preflights}, nil
}

// Lifecycle handles kit.lifecycle.
func (m *Module) Lifecycle(context.Context, KitLifecycleMsg) (*KitLifecycleResp, error) {
	if m.lifecycleSnapshot == nil {
		return nil, fmt.Errorf("control: lifecycle snapshot is not configured")
	}
	return &KitLifecycleResp{Lifecycle: m.lifecycleSnapshot()}, nil
}

// ModuleMount handles kit.module.mount.
func (m *Module) ModuleMount(ctx context.Context, req KitModuleMountMsg) (*KitModuleMountResp, error) {
	if m.lifecycle == nil {
		return nil, fmt.Errorf("control: module lifecycle is not configured")
	}
	desc, err := m.lifecycle.MountModule(ctx, req.ID, bkmodule.ModuleBuildConfig{
		JSON: req.Config,
		YAML: req.ConfigYAML,
	})
	if err != nil {
		return nil, err
	}
	preflight, err := m.lifecycle.PreflightModule(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &KitModuleMountResp{Module: desc, Preflight: preflight}, nil
}

// ModuleUnmount handles kit.module.unmount.
func (m *Module) ModuleUnmount(ctx context.Context, req KitModuleUnmountMsg) (*KitModuleUnmountResp, error) {
	if m.lifecycle == nil {
		return nil, fmt.Errorf("control: module lifecycle is not configured")
	}
	if req.ID == "control" {
		return nil, fmt.Errorf("control: refusing to unmount control module through its own command")
	}
	desc, err := m.lifecycle.UnmountModule(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &KitModuleUnmountResp{Module: desc}, nil
}

// ModuleDescribe handles kit.module.describe.
func (m *Module) ModuleDescribe(ctx context.Context, req KitModuleDescribeMsg) (*KitModuleDescribeResp, error) {
	if m.lifecycle == nil {
		return nil, fmt.Errorf("control: module lifecycle is not configured")
	}
	desc, mounted, err := m.lifecycle.DescribeModule(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	preflight, err := m.lifecycle.PreflightModule(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &KitModuleDescribeResp{Module: desc, Mounted: mounted, Preflight: preflight}, nil
}
