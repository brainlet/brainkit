// Package audit provides a centralized event log for operational events.
// All subsystems (plugins, RBAC, secrets, deployments, bus, tools) record
// events here. Events also fire on the bus for real-time subscribers, but
// the audit log persists them for historical queries.
package audit

import (
	"encoding/json"
	"time"
)

// Event is an operational event recorded in the audit log.
type Event struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Category  string          `json:"category"`  // "plugin", "tools", "bus", "security", "secrets", "deploy", "health"
	Type      string          `json:"type"`       // "plugin.started", "tools.call.denied", etc.
	Source    string          `json:"source"`     // plugin name, deployment source, caller
	RuntimeID string          `json:"runtimeId"`  // which Kit instance produced the event
	Namespace string          `json:"namespace"`  // Kit namespace
	Data      json.RawMessage `json:"data"`       // event-specific payload
	Duration  time.Duration   `json:"duration"`   // for call/operation events (0 if not applicable)
	Error     string          `json:"error"`      // non-empty if the operation failed
}

// Query filters for searching the audit log.
type Query struct {
	Category  string    // filter by category (empty = all)
	Type      string    // filter by exact event type (empty = all)
	Source    string    // filter by source (empty = all)
	Since     time.Time // events after this time (zero = no lower bound)
	Until     time.Time // events before this time (zero = no upper bound)
	Limit     int       // max results (0 = default 100)
	RuntimeID string    // filter by runtime (empty = all)
}

// Store persists and queries audit events.
type Store interface {
	// Record persists an event. Non-blocking — errors are logged, not returned.
	Record(event Event)

	// Query retrieves events matching the filter, newest first.
	Query(q Query) ([]Event, error)

	// Prune deletes events older than the given duration.
	Prune(olderThan time.Duration) error

	// Count returns total events in the store.
	Count() (int64, error)

	// CountByCategory returns event counts grouped by category.
	CountByCategory() (map[string]int64, error)

	// Close releases resources.
	Close() error
}
