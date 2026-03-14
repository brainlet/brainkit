package aiembed

import "encoding/json"

// Tool defines a function that AI models can call.
type Tool struct {
	Description string                                          `json:"description"`
	Parameters  json.RawMessage                                 `json:"parameters"`
	Execute     func(args json.RawMessage) (interface{}, error) `json:"-"`
}

// ToolChoice controls how the model selects tools.
type ToolChoice struct {
	Mode     string `json:"mode"`
	ToolName string `json:"toolName,omitempty"`
}

// ToolCall represents a tool invocation by the model.
type ToolCall struct {
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Args       json.RawMessage `json:"args"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string      `json:"toolCallId"`
	ToolName   string      `json:"toolName"`
	Args       interface{} `json:"args"`
	Result     interface{} `json:"result"`
}
