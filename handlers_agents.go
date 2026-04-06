package brainkit

import (
	"context"
	"fmt"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

// AgentInfo describes a registered agent.
type AgentInfo struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Model        string   `json:"model"`
	Status       string   `json:"status"` // "idle" | "busy" | "error"
	Kit          string   `json:"kit"`
}

type agentFilter struct {
	Capability string `json:"capability,omitempty"`
	Model      string `json:"model,omitempty"`
	Status     string `json:"status,omitempty"`
}

func (f *agentFilter) matches(info *AgentInfo) bool {
	if f.Status != "" && info.Status != f.Status {
		return false
	}
	if f.Model != "" && info.Model != f.Model {
		return false
	}
	if f.Capability != "" {
		found := false
		for _, cap := range info.Capabilities {
			if strings.EqualFold(cap, f.Capability) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// AgentsDomain handles agent lifecycle. Self-contained — no Kernel dependency.
type AgentsDomain struct {
	mu  syncx.RWMutex
	reg map[string]*AgentInfo
}

func newAgentsDomain() *AgentsDomain {
	return &AgentsDomain{
		reg: make(map[string]*AgentInfo),
	}
}

// Register adds an agent to the registry.
func (d *AgentsDomain) Register(_ context.Context, info AgentInfo) error {
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
func (d *AgentsDomain) Unregister(_ context.Context, name string) error {
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

// List returns all registered agents matching an optional filter.
func (d *AgentsDomain) List(_ context.Context, filter *agentFilter) (*messages.AgentListResp, error) {
	d.mu.RLock()
	var result []messages.AgentInfo
	for _, info := range d.reg {
		if filter != nil && !filter.matches(info) {
			continue
		}
		result = append(result, messages.AgentInfo{
			Name:         info.Name,
			Capabilities: info.Capabilities,
			Model:        info.Model,
			Status:       info.Status,
			Kit:          info.Kit,
		})
	}
	d.mu.RUnlock()
	if result == nil {
		result = []messages.AgentInfo{}
	}
	return &messages.AgentListResp{Agents: result}, nil
}

// Discover finds agents matching criteria.
func (d *AgentsDomain) Discover(_ context.Context, req messages.AgentDiscoverMsg) (*messages.AgentDiscoverResp, error) {
	filter := &agentFilter{
		Capability: req.Capability,
		Model:      req.Model,
		Status:     req.Status,
	}
	listResp, _ := d.List(context.Background(), filter)
	return &messages.AgentDiscoverResp{Agents: listResp.Agents}, nil
}

// GetStatus returns the status of a named agent.
func (d *AgentsDomain) GetStatus(_ context.Context, req messages.AgentGetStatusMsg) (*messages.AgentGetStatusResp, error) {
	if req.Name == "" {
		return nil, &sdk.ValidationError{Field: "name", Message: "is required"}
	}
	d.mu.RLock()
	info, ok := d.reg[req.Name]
	d.mu.RUnlock()
	if !ok {
		return nil, &sdk.NotFoundError{Resource: "agent", Name: req.Name}
	}
	return &messages.AgentGetStatusResp{Name: info.Name, Status: info.Status}, nil
}

// SetStatus updates the status of a named agent.
func (d *AgentsDomain) SetStatus(_ context.Context, req messages.AgentSetStatusMsg) (*messages.AgentSetStatusResp, error) {
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
	return &messages.AgentSetStatusResp{OK: true}, nil
}

// UnregisterAllForKit removes all agents registered by a specific Kit instance.
func (d *AgentsDomain) UnregisterAllForKit(kitID string) int {
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
func (d *AgentsDomain) Get(name string) *AgentInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	info, ok := d.reg[name]
	if !ok {
		return nil
	}
	cp := *info
	return &cp
}
