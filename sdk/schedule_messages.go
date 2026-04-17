package sdk

import "encoding/json"

// ── Scheduling ──

type ScheduleCreateMsg struct {
	Expression string          `json:"expression"` // "every 5m" or "in 30s"
	Topic      string          `json:"topic"`
	Payload    json.RawMessage `json:"payload"`
}

func (ScheduleCreateMsg) BusTopic() string { return "schedules.create" }

type ScheduleCreateResp struct {
	ID string `json:"id"`
}

type ScheduleCancelMsg struct {
	ID string `json:"id"`
}

func (ScheduleCancelMsg) BusTopic() string { return "schedules.cancel" }

type ScheduleCancelResp struct {
	Cancelled bool `json:"cancelled"`
}

type ScheduleListMsg struct{}

func (ScheduleListMsg) BusTopic() string { return "schedules.list" }

type ScheduleListResp struct {
	Schedules []ScheduleInfo `json:"schedules"`
}

type ScheduleInfo struct {
	ID         string `json:"id"`
	Expression string `json:"expression"`
	Topic      string `json:"topic"`
	NextFire   string `json:"nextFire"`
	OneTime    bool   `json:"oneTime"`
	Source     string `json:"source,omitempty"`
}
