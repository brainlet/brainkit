package sdk

import "encoding/json"

// PluginManifest declares a plugin's capabilities.
type PluginManifest struct {
	Owner         string                   `json:"owner"`
	Name          string                   `json:"name"`
	Version       string                   `json:"version"`
	Description   string                   `json:"description"`
	Tools         []ToolDefinition         `json:"tools,omitempty"`
	Interceptors  []InterceptorDefinition  `json:"interceptors,omitempty"`
	Events        []EventDefinition        `json:"events,omitempty"`
	Subscriptions []SubscriptionDefinition `json:"subscriptions,omitempty"`
	Agents        []AgentDefinition        `json:"agents,omitempty"`
	Files         []FileDefinition         `json:"files,omitempty"`
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
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema,omitempty"`
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
	Type    string `json:"type"`    // "agent" | "tool" | "module"
	Content string `json:"content"`
}

// InterceptMessage is delivered to intercept handlers.
type InterceptMessage struct {
	Topic    string            `json:"topic"`
	CallerID string            `json:"callerId"`
	Payload  json.RawMessage   `json:"payload"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
