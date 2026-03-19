package messages

// ── Requests ──

type AgentRequestMsg struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

func (AgentRequestMsg) BusTopic() string { return "agents.request" }

type AgentStreamMsg struct {
	Name     string `json:"name"`
	Prompt   string `json:"prompt"`
	StreamTo string `json:"streamTo"`
}

func (AgentStreamMsg) BusTopic() string { return "agents.stream" }

type AgentMessageMsg struct {
	Target  string `json:"target"`
	Payload any    `json:"payload"`
}

func (AgentMessageMsg) BusTopic() string { return "agents.message" }

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

type AgentRegisterMsg struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
	Model        string   `json:"model"`
	Kit          string   `json:"kit"`
}

func (AgentRegisterMsg) BusTopic() string { return "agents.register" }

type AgentUnregisterMsg struct {
	Name string `json:"name"`
}

func (AgentUnregisterMsg) BusTopic() string { return "agents.unregister" }

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

type AgentRequestResp struct {
	Text string `json:"text"`
}

type AgentUnregisterResp struct {
	OK bool `json:"ok"`
}

type AgentGetStatusResp struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AgentSetStatusResp struct {
	OK bool `json:"ok"`
}

type AgentRegisterResp struct {
	Registered string `json:"registered"`
}

type AgentListResp struct {
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
