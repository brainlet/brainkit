// Package toolmsg contains the typed bus API for modules/tools.
package toolmsg

import "encoding/json"

// ToolCallMsg requests execution of a registered tool.
type ToolCallMsg struct {
	Name  string `json:"name"`
	Input any    `json:"input"`
}

func (ToolCallMsg) BusTopic() string { return "tools.call" }

// ToolListMsg requests registered tools, optionally filtered by namespace.
type ToolListMsg struct {
	Namespace string `json:"namespace,omitempty"`
}

func (ToolListMsg) BusTopic() string { return "tools.list" }

// ToolResolveMsg requests metadata for a registered tool.
type ToolResolveMsg struct {
	Name string `json:"name"`
}

func (ToolResolveMsg) BusTopic() string { return "tools.resolve" }

// ToolListResp reports registered tools.
type ToolListResp struct {
	Tools []ToolInfo `json:"tools"`
}

// ToolInfo is the bus wire shape for one registered tool.
type ToolInfo struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Namespace   string `json:"namespace"`
	Description string `json:"description"`
}

// ToolResolveResp reports metadata for one registered tool.
type ToolResolveResp struct {
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema,omitempty"`
}

// ToolCallResp reports the raw JSON result produced by a tool call.
type ToolCallResp struct {
	Result json.RawMessage `json:"result"`
}
