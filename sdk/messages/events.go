package messages

import "encoding/json"

// Events are fire-and-forget — no response type needed.

type KitDeployedEvent struct {
	Source    string         `json:"source"`
	Resources []ResourceInfo `json:"resources,omitempty"`
}

func (KitDeployedEvent) BusTopic() string { return "kit.deployed" }

type KitTeardownedEvent struct {
	Source  string `json:"source"`
	Removed int    `json:"removed"`
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
