package sdk

import "encoding/json"

// ── Requests ──

type McpListToolsMsg struct {
	Server string `json:"server,omitempty"`
}

func (McpListToolsMsg) BusTopic() string { return "mcp.listTools" }

type McpCallToolMsg struct {
	Server string `json:"server"`
	Tool   string `json:"tool"`
	Args   any    `json:"args"`
}

func (McpCallToolMsg) BusTopic() string { return "mcp.callTool" }

// ── Responses ──

type McpListToolsResp struct {
	ResultMeta
	Tools []McpToolInfo `json:"tools"`
}


type McpCallToolResp struct {
	ResultMeta
	Result json.RawMessage `json:"result"`
}


// ── Shared types ──

type McpToolInfo struct {
	Name        string `json:"name"`
	Server      string `json:"server"`
	Description string `json:"description"`
}
