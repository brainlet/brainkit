package systemmsg

import "github.com/brainlet/brainkit/sdk"

// Events are fire-and-forget; no response type is expected.

const (
	TopicKitDeployed      = "kit.deployed"
	TopicKitTeardowned    = "kit.teardown.done"
	TopicHandlerFailed    = "bus.handler.failed"
	TopicHandlerExhausted = "bus.handler.exhausted"
)

type KitDeployedEvent struct {
	Source    string             `json:"source"`
	RuntimeID string             `json:"runtimeId,omitempty"` // who deployed; used for propagation skip-self
	Resources []sdk.ResourceInfo `json:"resources,omitempty"`
}

func (KitDeployedEvent) BusTopic() string { return TopicKitDeployed }

type KitTeardownedEvent struct {
	Source    string `json:"source"`
	RuntimeID string `json:"runtimeId,omitempty"` // who tore down; used for propagation skip-self
	Removed   int    `json:"removed"`
}

func (KitTeardownedEvent) BusTopic() string { return TopicKitTeardowned }

// HandlerFailedEvent is emitted when a bus handler returns an error.
type HandlerFailedEvent struct {
	Topic      string `json:"topic"`
	Source     string `json:"source"`
	Error      string `json:"error"`
	RetryCount int    `json:"retryCount"`
	WillRetry  bool   `json:"willRetry"`
}

func (HandlerFailedEvent) BusTopic() string { return TopicHandlerFailed }

// HandlerExhaustedEvent is emitted when retries are exhausted for a failed handler.
type HandlerExhaustedEvent struct {
	Topic      string `json:"topic"`
	Source     string `json:"source"`
	Error      string `json:"error"`
	RetryCount int    `json:"retryCount"`
}

func (HandlerExhaustedEvent) BusTopic() string { return TopicHandlerExhausted }
