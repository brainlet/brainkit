package messaging

import (
	"encoding/json"
)

// KitSendMsg requests a nested request/reply call to another bus topic.
type KitSendMsg struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func (KitSendMsg) BusTopic() string { return "kit.send" }

// KitSendResp carries the terminal reply payload from the nested call.
type KitSendResp struct {
	Payload json.RawMessage `json:"payload"`
}
