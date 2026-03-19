package messages

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

type ToolRegisterMsg struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

func (ToolRegisterMsg) BusTopic() string { return "tools.register" }

// ── Responses ──

type ToolListResp struct {
	Tools []ToolInfo `json:"tools"`
}

type ToolInfo struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
}

type ToolResolveResp struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

type ToolRegisterResp struct {
	Registered string `json:"registered"`
}
