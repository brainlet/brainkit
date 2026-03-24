package messages

// ── Registry Requests (discovery + status) ──

type AgentListMsg struct {
	Filter *AgentFilter `json:"filter,omitempty"`
}

func (AgentListMsg) BusTopic() string { return "agents.list" }

type AgentDiscoverMsg struct {
	Capability string `json:"capability,omitempty"`
	Model      string `json:"model,omitempty"`
	Status     string `json:"status,omitempty"`
}

func (AgentDiscoverMsg) BusTopic() string { return "agents.discover" }

type AgentGetStatusMsg struct {
	Name string `json:"name"`
}

func (AgentGetStatusMsg) BusTopic() string { return "agents.get-status" }

type AgentSetStatusMsg struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (AgentSetStatusMsg) BusTopic() string { return "agents.set-status" }

// ── Responses ──

type AgentGetStatusResp struct {
	ResultMeta
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AgentSetStatusResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

type AgentListResp struct {
	ResultMeta
	Agents []AgentInfo `json:"agents"`
}

type AgentDiscoverResp struct {
	ResultMeta
	Agents []AgentInfo `json:"agents"`
}

// ── Shared types ──

type AgentInfo struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Model        string   `json:"model"`
	Status       string   `json:"status"`
	Kit          string   `json:"kit"`
}

type AgentFilter struct {
	Capability string `json:"capability,omitempty"`
	Model      string `json:"model,omitempty"`
	Status     string `json:"status,omitempty"`
}
