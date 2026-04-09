package sdk

import (
	"encoding/json"
	"time"
)

// ── Audit query messages ──

// AuditQueryMsg queries the centralized audit event log.
type AuditQueryMsg struct {
	Category  string    `json:"category,omitempty"`  // "plugin", "tools", "bus", "security", "secrets", "deploy", "health"
	Type      string    `json:"type,omitempty"`      // exact event type, e.g. "plugin.started"
	Source    string    `json:"source,omitempty"`     // plugin name, deployment source, etc.
	Since     time.Time `json:"since,omitempty"`      // events after this time
	Until     time.Time `json:"until,omitempty"`      // events before this time
	Limit     int       `json:"limit,omitempty"`      // max results (default 100)
	RuntimeID string    `json:"runtimeId,omitempty"`  // filter by runtime
}

func (AuditQueryMsg) BusTopic() string { return "audit.query" }

// AuditQueryResp contains the query results.
type AuditQueryResp struct {
	Events []AuditEvent `json:"events"`
	Total  int64        `json:"total"` // total events in store (not just this page)
}

// AuditEvent is a single audit log entry.
type AuditEvent struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	Category  string          `json:"category"`
	Type      string          `json:"type"`
	Source    string          `json:"source"`
	RuntimeID string          `json:"runtimeId"`
	Namespace string          `json:"namespace"`
	Data      json.RawMessage `json:"data"`
	Duration  time.Duration   `json:"duration"`
	Error     string          `json:"error,omitempty"`
}

// AuditStatsMsg requests audit store statistics.
type AuditStatsMsg struct{}

func (AuditStatsMsg) BusTopic() string { return "audit.stats" }

// AuditStatsResp returns audit store statistics.
type AuditStatsResp struct {
	TotalEvents    int64            `json:"totalEvents"`
	EventsByCategory map[string]int64 `json:"eventsByCategory"`
}

// AuditPruneMsg prunes old audit events.
type AuditPruneMsg struct {
	OlderThanHours int `json:"olderThanHours"` // delete events older than N hours
}

func (AuditPruneMsg) BusTopic() string { return "audit.prune" }

// AuditPruneResp confirms the prune operation.
type AuditPruneResp struct {
	Pruned bool `json:"pruned"`
}
