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

// AgentRegisterMsg and AgentUnregisterMsg removed — agents are created via
// .ts deployment (kit.Deploy), not via bus messages. The internal bridgeRequest
// path in kit_runtime.js still handles agent registration.

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
	ResultMeta
	Text string `json:"text"`
}

func (AgentRequestResp) BusTopic() string { return "agents.request.result" }

type AgentUnregisterResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (AgentUnregisterResp) BusTopic() string { return "agents.unregister.result" }

type AgentGetStatusResp struct {
	ResultMeta
	Name   string `json:"name"`
	Status string `json:"status"`
}

func (AgentGetStatusResp) BusTopic() string { return "agents.get-status.result" }

type AgentSetStatusResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (AgentSetStatusResp) BusTopic() string { return "agents.set-status.result" }

type AgentListResp struct {
	ResultMeta
	Agents []AgentInfo `json:"agents"`
}

func (AgentListResp) BusTopic() string { return "agents.list.result" }

type AgentDiscoverResp struct {
	ResultMeta
	Agents []AgentInfo `json:"agents"`
}

func (AgentDiscoverResp) BusTopic() string { return "agents.discover.result" }

type AgentMessageResp struct {
	ResultMeta
	Delivered bool `json:"delivered"`
}

func (AgentMessageResp) BusTopic() string { return "agents.message.result" }

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
