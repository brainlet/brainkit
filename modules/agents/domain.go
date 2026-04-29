package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/modules/agents/agentmsg"
	"github.com/brainlet/brainkit/sdk"
)

type AgentInfo = agentmsg.AgentInfo

// Domain owns the in-memory agent registry and agents.* command behavior.
type Domain struct {
	mu  syncx.RWMutex
	reg map[string]*AgentInfo
}

func NewDomain() *Domain {
	return &Domain{
		reg: make(map[string]*AgentInfo),
	}
}

// Register adds an agent to the registry.
func (d *Domain) Register(_ context.Context, info AgentInfo) error {
	if info.Name == "" {
		return &sdk.ValidationError{Field: "name", Message: "is required"}
	}
	if info.Status == "" {
		info.Status = "idle"
	}
	d.mu.Lock()
	d.reg[info.Name] = &info
	d.mu.Unlock()
	return nil
}

// Unregister removes an agent from the registry.
func (d *Domain) Unregister(_ context.Context, name string) error {
	if name == "" {
		return &sdk.ValidationError{Field: "name", Message: "is required"}
	}
	d.mu.Lock()
	_, ok := d.reg[name]
	if ok {
		delete(d.reg, name)
	}
	d.mu.Unlock()
	if !ok {
		return &sdk.NotFoundError{Resource: "agent", Name: name}
	}
	return nil
}

// ListAgents returns all registered agents matching an optional filter.
func (d *Domain) ListAgents(_ context.Context, req agentmsg.AgentListMsg) (*agentmsg.AgentListResp, error) {
	d.mu.RLock()
	var result []agentmsg.AgentInfo
	for _, info := range d.reg {
		if req.Filter != nil && !agentMatches(req.Filter, info) {
			continue
		}
		result = append(result, *info)
	}
	d.mu.RUnlock()
	if result == nil {
		result = []agentmsg.AgentInfo{}
	}
	return &agentmsg.AgentListResp{Agents: result}, nil
}

// DiscoverAgents finds agents matching criteria.
func (d *Domain) DiscoverAgents(ctx context.Context, req agentmsg.AgentDiscoverMsg) (*agentmsg.AgentDiscoverResp, error) {
	listResp, _ := d.ListAgents(ctx, agentmsg.AgentListMsg{
		Filter: &agentmsg.AgentFilter{
			Capability: req.Capability,
			Model:      req.Model,
			Status:     req.Status,
		},
	})
	return &agentmsg.AgentDiscoverResp{Agents: listResp.Agents}, nil
}

// GetAgentStatus returns the status of a named agent.
func (d *Domain) GetAgentStatus(_ context.Context, req agentmsg.AgentGetStatusMsg) (*agentmsg.AgentGetStatusResp, error) {
	if req.Name == "" {
		return nil, &sdk.ValidationError{Field: "name", Message: "is required"}
	}
	d.mu.RLock()
	info, ok := d.reg[req.Name]
	d.mu.RUnlock()
	if !ok {
		return nil, &sdk.NotFoundError{Resource: "agent", Name: req.Name}
	}
	return &agentmsg.AgentGetStatusResp{Name: info.Name, Status: info.Status}, nil
}

// SetAgentStatus updates the status of a named agent.
func (d *Domain) SetAgentStatus(_ context.Context, req agentmsg.AgentSetStatusMsg) (*agentmsg.AgentSetStatusResp, error) {
	if req.Name == "" {
		return nil, &sdk.ValidationError{Field: "name", Message: "is required"}
	}
	if req.Status == "" {
		return nil, &sdk.ValidationError{Field: "status", Message: "is required"}
	}
	switch req.Status {
	case "idle", "busy", "error":
	default:
		return nil, &sdk.ValidationError{Field: "status", Message: fmt.Sprintf("invalid value %q (must be idle|busy|error)", req.Status)}
	}
	d.mu.Lock()
	info, ok := d.reg[req.Name]
	if ok {
		info.Status = req.Status
	}
	d.mu.Unlock()
	if !ok {
		return nil, &sdk.NotFoundError{Resource: "agent", Name: req.Name}
	}
	return &agentmsg.AgentSetStatusResp{OK: true}, nil
}

// UnregisterAllForKit removes all agents registered by a specific Kit instance.
func (d *Domain) UnregisterAllForKit(kitID string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	count := 0
	for name, info := range d.reg {
		if info.Kit == kitID {
			delete(d.reg, name)
			count++
		}
	}
	return count
}

// Get returns agent info by name, or nil if not found.
func (d *Domain) Get(name string) *AgentInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	info, ok := d.reg[name]
	if !ok {
		return nil
	}
	cp := *info
	return &cp
}

func agentMatches(filter *agentmsg.AgentFilter, info *AgentInfo) bool {
	if filter.Status != "" && info.Status != filter.Status {
		return false
	}
	if filter.Model != "" && info.Model != filter.Model {
		return false
	}
	if filter.Capability != "" {
		for _, cap := range info.Capabilities {
			if strings.EqualFold(cap, filter.Capability) {
				return true
			}
		}
		return false
	}
	return true
}
