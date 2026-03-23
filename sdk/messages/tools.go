package messages

import "encoding/json"

// ── Requests ──

type ToolCallMsg struct {
	Name  string `json:"name"`
	Input any    `json:"input"`
}

func (ToolCallMsg) BusTopic() string { return "tools.call" }

type ToolListMsg struct {
	Namespace string `json:"namespace,omitempty"`
}

func (ToolListMsg) BusTopic() string { return "tools.list" }

type ToolResolveMsg struct {
	Name string `json:"name"`
}

func (ToolResolveMsg) BusTopic() string { return "tools.resolve" }

// ToolRegisterMsg removed — tools are created via .ts deployment (kit.Deploy),
// not via bus messages. The internal bridgeRequest path still handles tool registration.

// ── Responses ──

type ToolListResp struct {
	ResultMeta
	Tools []ToolInfo `json:"tools"`
}


type ToolInfo struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
}

type ToolResolveResp struct {
	ResultMeta
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema,omitempty"`
}


// ToolRegisterResp removed — see note above.

type ToolCallResp struct {
	ResultMeta
	Result json.RawMessage `json:"result"`
}

