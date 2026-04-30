package agenthost

import (
	"context"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/sdk"
)

// AgentInfo is the host-side representation of a registered agent.
type AgentInfo struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Model        string   `json:"model"`
	Status       string   `json:"status"`
	Kit          string   `json:"kit"`
}

// AgentFilter filters agent list results.
type AgentFilter struct {
	Capability string `json:"capability,omitempty"`
	Model      string `json:"model,omitempty"`
	Status     string `json:"status,omitempty"`
}

type ListRequest struct {
	Filter *AgentFilter `json:"filter,omitempty"`
}

type ListResponse struct {
	Agents []AgentInfo `json:"agents"`
}

type DiscoverRequest struct {
	Capability string `json:"capability,omitempty"`
	Model      string `json:"model,omitempty"`
	Status     string `json:"status,omitempty"`
}

type DiscoverResponse struct {
	Agents []AgentInfo `json:"agents"`
}

type StatusRequest struct {
	Name string `json:"name"`
}

type StatusResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type SetStatusRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type SetStatusResponse struct {
	OK bool `json:"ok"`
}

// Registry is the capability consumed by modules/agents.
type Registry interface {
	ListAgents(context.Context, ListRequest) (*ListResponse, error)
	DiscoverAgents(context.Context, DiscoverRequest) (*DiscoverResponse, error)
	GetAgentStatus(context.Context, StatusRequest) (*StatusResponse, error)
	SetAgentStatus(context.Context, SetStatusRequest) (*SetStatusResponse, error)
}

// Domain owns the in-memory host-side agent registry. The modules/agents
// package is only the bus command adapter for this capability.
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
func (d *Domain) ListAgents(_ context.Context, req ListRequest) (*ListResponse, error) {
	d.mu.RLock()
	var result []AgentInfo
	for _, info := range d.reg {
		if req.Filter != nil && !agentMatches(req.Filter, info) {
			continue
		}
		result = append(result, *info)
	}
	d.mu.RUnlock()
	if result == nil {
		result = []AgentInfo{}
	}
	return &ListResponse{Agents: result}, nil
}

// DiscoverAgents finds agents matching criteria.
func (d *Domain) DiscoverAgents(ctx context.Context, req DiscoverRequest) (*DiscoverResponse, error) {
	listResp, _ := d.ListAgents(ctx, ListRequest{
		Filter: &AgentFilter{
			Capability: req.Capability,
			Model:      req.Model,
			Status:     req.Status,
		},
	})
	return &DiscoverResponse{Agents: listResp.Agents}, nil
}

// GetAgentStatus returns the status of a named agent.
func (d *Domain) GetAgentStatus(_ context.Context, req StatusRequest) (*StatusResponse, error) {
	if req.Name == "" {
		return nil, &sdk.ValidationError{Field: "name", Message: "is required"}
	}
	d.mu.RLock()
	info, ok := d.reg[req.Name]
	d.mu.RUnlock()
	if !ok {
		return nil, &sdk.NotFoundError{Resource: "agent", Name: req.Name}
	}
	return &StatusResponse{Name: info.Name, Status: info.Status}, nil
}

// SetAgentStatus updates the status of a named agent.
func (d *Domain) SetAgentStatus(_ context.Context, req SetStatusRequest) (*SetStatusResponse, error) {
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
	return &SetStatusResponse{OK: true}, nil
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

func agentMatches(filter *AgentFilter, info *AgentInfo) bool {
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
