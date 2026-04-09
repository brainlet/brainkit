// Package pluginws defines the WebSocket protocol between brainkit host and plugins.
// Plugins connect to the host's WS endpoint, send a manifest, receive tool calls,
// and send results back. No Watermill dependency — just WebSocket + JSON.
package pluginws

import "encoding/json"

// Message is the envelope for all WS messages between host and plugin.
type Message struct {
	Type string          `json:"type"`
	ID   string          `json:"id,omitempty"` // correlation ID for tool calls
	Data json.RawMessage `json:"data"`
}

// Message types
const (
	// Plugin → Host
	TypeManifest   = "manifest"
	TypeToolResult = "tool.result"
	TypePublish    = "publish" // plugin publishes to bus topic

	// Host → Plugin
	TypeManifestAck = "manifest.ack"
	TypeToolCall    = "tool.call"
	TypeShutdown    = "shutdown"
)

// Manifest is sent by the plugin after connecting.
type Manifest struct {
	Owner       string    `json:"owner"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description,omitempty"`
	Tools       []ToolDef `json:"tools"`
}

// ToolDef describes a tool the plugin provides.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"inputSchema,omitempty"`
}

// ManifestAck is sent by the host after processing the manifest.
type ManifestAck struct {
	Registered bool   `json:"registered"`
	Error      string `json:"error,omitempty"`
}

// ToolCall is sent by the host when a tool is invoked.
type ToolCall struct {
	Tool     string          `json:"tool"`
	Input    json.RawMessage `json:"input"`
	CallerID string          `json:"callerID,omitempty"`
}

// ToolResult is sent by the plugin after executing a tool.
type ToolResult struct {
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// PublishMsg is sent by the plugin to publish a message to a bus topic.
type PublishMsg struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}
