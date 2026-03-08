// Ported from: packages/core/src/run/types.ts
package run

// RunStatus represents the status of a run.
type RunStatus string

const (
	RunStatusCreated   RunStatus = "created"
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
)

// Run represents a run instance.
type Run struct {
	RunID     *string    `json:"runId,omitempty"`
	RunStatus *RunStatus `json:"runStatus,omitempty"`
}
