// Ported from: packages/core/src/events/types.ts
package events

import "time"

// Event represents a domain event with metadata.
type Event struct {
	Type      string      `json:"type"`
	ID        string      `json:"id"`
	Data      any         `json:"data"`
	RunID     string      `json:"runId"`
	CreatedAt time.Time   `json:"createdAt"`
}

// PublishEvent contains the fields required when publishing an event.
// The ID and CreatedAt fields are assigned by the PubSub implementation.
type PublishEvent struct {
	Type  string `json:"type"`
	Data  any    `json:"data"`
	RunID string `json:"runId"`
}
