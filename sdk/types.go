package sdk

import "encoding/json"

// PluginManifest declares a plugin's capabilities.
type PluginManifest struct {
	Name         string                   `json:"name"`
	Version      string                   `json:"version"`
	Description  string                   `json:"description"`
	Tools        []ToolDefinition         `json:"tools,omitempty"`
	Interceptors []InterceptorDefinition  `json:"interceptors,omitempty"`
	Events       []EventDefinition        `json:"events,omitempty"`
	Subscriptions []SubscriptionDefinition `json:"subscriptions,omitempty"`
	Agents       []AgentDefinition        `json:"agents,omitempty"`
	Files        []FileDefinition         `json:"files,omitempty"`
}

// ToolDefinition declares a tool the plugin provides.
type ToolDefinition struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	InputSchema  string `json:"inputSchema"`
	OutputSchema string `json:"outputSchema,omitempty"`
}

// InterceptorDefinition declares an interceptor.
type InterceptorDefinition struct {
	Name        string `json:"name"`
	Priority    int    `json:"priority"`
	TopicFilter string `json:"topicFilter"`
}

// EventDefinition declares an event the plugin can emit.
type EventDefinition struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

// SubscriptionDefinition declares a bus topic the plugin subscribes to.
type SubscriptionDefinition struct {
	Topic string `json:"topic"`
}

// AgentDefinition declares an agent the plugin provides.
type AgentDefinition struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Model        string   `json:"model,omitempty"`
	Tools        []string `json:"tools,omitempty"`
	Instructions string   `json:"instructions"`
}

// FileDefinition declares a .ts file the plugin injects into the Kit.
type FileDefinition struct {
	Path    string `json:"path"`
	Type    string `json:"type"` // "agent" | "tool" | "module"
	Content string `json:"content"`
}

// Event is delivered to HandleEvent.
type Event struct {
	Topic    string          `json:"topic"`
	Payload  json.RawMessage `json:"payload"`
	TraceID  string          `json:"traceId"`
	CallerID string          `json:"callerId"`
}

// InterceptMessage is delivered to HandleIntercept.
type InterceptMessage struct {
	Topic    string            `json:"topic"`
	CallerID string            `json:"callerId"`
	Payload  json.RawMessage   `json:"payload"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// WASMCompileOpts for CompileWASM.
type WASMCompileOpts struct {
	Name string `json:"name,omitempty"`
}

// WASMModule returned by CompileWASM.
type WASMModule struct {
	Name    string   `json:"name"`
	Size    int      `json:"size"`
	Exports []string `json:"exports"`
}

// ShardDescriptor returned by DeployWASM.
type ShardDescriptor struct {
	Module   string            `json:"module"`
	Mode     string            `json:"mode"`
	StateKey string            `json:"stateKey"`
	Handlers map[string]string `json:"handlers"`
}
