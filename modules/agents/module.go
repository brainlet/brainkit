// Package agents owns the agents.* registry commands as a hot-mountable Kit
// module.
package agents

import (
	"context"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/agents/agentmsg"
)

// Module exposes agents.list/discover/get-status/set-status. Construct via New
// and include in brainkit.Config.Modules when the runtime should expose the
// agent registry over the bus.
type Module struct {
	registry agentRegistry
}

type agentRegistry interface {
	ListAgents(context.Context, agentmsg.AgentListMsg) (*agentmsg.AgentListResp, error)
	DiscoverAgents(context.Context, agentmsg.AgentDiscoverMsg) (*agentmsg.AgentDiscoverResp, error)
	GetAgentStatus(context.Context, agentmsg.AgentGetStatusMsg) (*agentmsg.AgentGetStatusResp, error)
	SetAgentStatus(context.Context, agentmsg.AgentSetStatusMsg) (*agentmsg.AgentSetStatusResp, error)
}

// New creates the agents module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "agents" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers agents.* command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	registry, err := bkmodule.RequireCapability[agentRegistry](host, bkmodule.CapabilityAgentRegistry)
	if err != nil {
		return fmt.Errorf("agents: %w", err)
	}
	m.registry = registry
	host.Scope().Defer(func(context.Context) error {
		m.registry = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.List),
		bkmodule.Command(m.Discover),
		bkmodule.Command(m.GetStatus),
		bkmodule.Command(m.SetStatus),
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
	m.registry = nil
	return nil
}

// Factory is the registered ModuleFactory for agents.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the agents module.
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
		Name:    "agents",
		Status:  bkmodule.StatusStable,
		Summary: "Agent registry bus commands (agents.list, discover, get-status, set-status).",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[agentmsg.AgentDiscoverMsg, agentmsg.AgentDiscoverResp](),
			bkmodule.CommandMessage[agentmsg.AgentGetStatusMsg, agentmsg.AgentGetStatusResp](),
			bkmodule.CommandMessage[agentmsg.AgentListMsg, agentmsg.AgentListResp](),
			bkmodule.CommandMessage[agentmsg.AgentSetStatusMsg, agentmsg.AgentSetStatusResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[agentRegistry](bkmodule.CapabilityAgentRegistry),
		},
	}
}

func init() { bkmodule.Register("agents", Factory{}) }

// List handles agents.list.
func (m *Module) List(ctx context.Context, req agentmsg.AgentListMsg) (*agentmsg.AgentListResp, error) {
	return m.registry.ListAgents(ctx, req)
}

// Discover handles agents.discover.
func (m *Module) Discover(ctx context.Context, req agentmsg.AgentDiscoverMsg) (*agentmsg.AgentDiscoverResp, error) {
	return m.registry.DiscoverAgents(ctx, req)
}

// GetStatus handles agents.get-status.
func (m *Module) GetStatus(ctx context.Context, req agentmsg.AgentGetStatusMsg) (*agentmsg.AgentGetStatusResp, error) {
	return m.registry.GetAgentStatus(ctx, req)
}

// SetStatus handles agents.set-status.
func (m *Module) SetStatus(ctx context.Context, req agentmsg.AgentSetStatusMsg) (*agentmsg.AgentSetStatusResp, error) {
	return m.registry.SetAgentStatus(ctx, req)
}
