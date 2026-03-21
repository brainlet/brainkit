package brainkit

import (
	"strings"
	"sync"
)

// AgentInfo describes a registered agent.
type AgentInfo struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Model        string   `json:"model"`
	Status       string   `json:"status"` // "idle" | "busy" | "error"
	Kit          string   `json:"kit"`
}

// agentRegistry stores all registered agents for a Kit.
type agentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*AgentInfo
}

func newAgentRegistry() *agentRegistry {
	return &agentRegistry{
		agents: make(map[string]*AgentInfo),
	}
}

func (r *agentRegistry) register(info AgentInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if info.Status == "" {
		info.Status = "idle"
	}
	r.agents[info.Name] = &info
}

func (r *agentRegistry) unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.agents[name]
	if ok {
		delete(r.agents, name)
	}
	return ok
}

func (r *agentRegistry) unregisterAllForKit(kitID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for name, info := range r.agents {
		if info.Kit == kitID {
			delete(r.agents, name)
			count++
		}
	}
	return count
}

func (r *agentRegistry) list(filter *agentFilter) []AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []AgentInfo
	for _, info := range r.agents {
		if filter != nil && !filter.matches(info) {
			continue
		}
		result = append(result, *info)
	}
	return result
}

func (r *agentRegistry) get(name string) *AgentInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, ok := r.agents[name]
	if !ok {
		return nil
	}
	cp := *info
	return &cp
}

func (r *agentRegistry) setStatus(name, status string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	info, ok := r.agents[name]
	if !ok {
		return false
	}
	info.Status = status
	return true
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
