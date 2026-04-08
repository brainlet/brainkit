package sdk

import "encoding/json"

// Events are fire-and-forget — no response type needed.

type KitDeployedEvent struct {
	Source    string         `json:"source"`
	RuntimeID string         `json:"runtimeId,omitempty"` // who deployed — for propagation skip-self
	Resources []ResourceInfo `json:"resources,omitempty"`
}

func (KitDeployedEvent) BusTopic() string { return "kit.deployed" }

type KitTeardownedEvent struct {
	Source    string `json:"source"`
	RuntimeID string `json:"runtimeId,omitempty"` // who tore down — for propagation skip-self
	Removed   int    `json:"removed"`
}

func (KitTeardownedEvent) BusTopic() string { return "kit.teardown.done" }

type PluginRegisteredEvent struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Tools   int    `json:"tools"`
}

func (PluginRegisteredEvent) BusTopic() string { return "plugin.registered" }

// CustomEvent is a user-defined event with a dynamic topic.
type CustomEvent struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func (e CustomEvent) BusTopic() string { return e.Topic }

// HandlerFailedEvent is emitted when a bus handler throws an exception.
type HandlerFailedEvent struct {
	Topic      string `json:"topic"`
	Source     string `json:"source"`
	Error      string `json:"error"`
	RetryCount int    `json:"retryCount"`
	WillRetry  bool   `json:"willRetry"`
}

func (HandlerFailedEvent) BusTopic() string { return "bus.handler.failed" }

// HandlerExhaustedEvent is emitted when retries are exhausted for a failed handler.
type HandlerExhaustedEvent struct {
	Topic      string `json:"topic"`
	Source     string `json:"source"`
	Error      string `json:"error"`
	RetryCount int    `json:"retryCount"`
}

func (HandlerExhaustedEvent) BusTopic() string { return "bus.handler.exhausted" }

// PluginStartedEvent is emitted when a plugin is started dynamically.
type PluginStartedEvent struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Version string `json:"version,omitempty"`
}

func (PluginStartedEvent) BusTopic() string { return "plugin.started" }

// PluginStoppedEvent is emitted when a plugin is stopped.
type PluginStoppedEvent struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

func (PluginStoppedEvent) BusTopic() string { return "plugin.stopped" }

// PermissionDeniedEvent is emitted when RBAC denies a bus/command/registration operation.
type PermissionDeniedEvent struct {
	Source string `json:"source"` // deployment or plugin name
	Topic  string `json:"topic"`  // topic or command that was denied
	Action string `json:"action"` // "publish", "subscribe", "emit", "command", "register"
	Role   string `json:"role"`   // role that denied it
	Reason string `json:"reason"` // human-readable
}

func (PermissionDeniedEvent) BusTopic() string { return "bus.permission.denied" }

// ReplyDeniedEvent is emitted when a reply is rejected due to invalid/missing token.
type ReplyDeniedEvent struct {
	Source        string `json:"source"`
	Topic         string `json:"topic"`
	CorrelationID string `json:"correlationId"`
	Reason        string `json:"reason"`
}

func (ReplyDeniedEvent) BusTopic() string { return "bus.reply.denied" }
