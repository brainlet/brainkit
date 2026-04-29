package health

import "encoding/json"

// KitHealthMsg requests the runtime health snapshot over the bus.
type KitHealthMsg struct{}

func (KitHealthMsg) BusTopic() string { return "kit.health" }

// KitHealthResp wraps the JSON-encoded brainkit.HealthStatus payload.
type KitHealthResp struct {
	Health json.RawMessage `json:"health"`
}
