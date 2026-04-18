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
	TypePublish    = "publish"   // plugin publishes to bus topic
	TypeSubscribe  = "subscribe" // plugin subscribes to bus topic

	// Host → Plugin
	TypeManifestAck = "manifest.ack"
	TypeToolCall    = "tool.call"
	TypeEvent       = "event" // bus event forwarded to plugin
	TypeShutdown    = "shutdown"
	TypeCancel      = "tool.cancel" // cancel an in-flight tool call
)

// Manifest is sent by the plugin after connecting.
type Manifest struct {
	Owner         string    `json:"owner"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Description   string    `json:"description,omitempty"`
	Tools         []ToolDef `json:"tools"`
	Subscriptions []string  `json:"subscriptions,omitempty"` // bus topics to subscribe to
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
	Topic         string            `json:"topic"`
	Payload       json.RawMessage   `json:"payload"`
	Metadata      map[string]string `json:"metadata,omitempty"` // replyTo, correlationId, etc.
}

// SubscribeMsg is sent by the plugin to subscribe to a bus topic.
// The host creates a bus subscription and forwards matching events over WS.
type SubscribeMsg struct {
	Topic string `json:"topic"`
}

// EventMsg is sent by the host when a subscribed bus topic receives a message.
type EventMsg struct {
	Topic    string          `json:"topic"`
	Payload  json.RawMessage `json:"payload"`
	CallerID string          `json:"callerID,omitempty"`
}

// CancelMsg is sent by the host to signal that an in-flight tool call
// should be aborted. Plugins look up the cancel func they recorded
// when the tool call arrived and invoke it. ToolCallID matches the
// Message.ID of the original TypeToolCall frame.
type CancelMsg struct {
	ToolCallID string `json:"toolCallId"`
	Reason     string `json:"reason,omitempty"`
}
