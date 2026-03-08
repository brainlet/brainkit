// Ported from: packages/core/src/run/types.test.ts
package run

import (
	"testing"
)

func TestRunStatus(t *testing.T) {
	t.Run("has correct status values", func(t *testing.T) {
		cases := []struct {
			status RunStatus
			want   string
		}{
			{RunStatusCreated, "created"},
			{RunStatusRunning, "running"},
			{RunStatusCompleted, "completed"},
			{RunStatusFailed, "failed"},
		}
		for _, tc := range cases {
			if string(tc.status) != tc.want {
				t.Errorf("status = %q, want %q", tc.status, tc.want)
			}
		}
	})

	t.Run("statuses are distinct", func(t *testing.T) {
		statuses := []RunStatus{
			RunStatusCreated,
			RunStatusRunning,
			RunStatusCompleted,
			RunStatusFailed,
		}
		seen := make(map[RunStatus]bool)
		for _, s := range statuses {
			if seen[s] {
				t.Errorf("duplicate status: %q", s)
			}
			seen[s] = true
		}
	})
}

func TestRunStruct(t *testing.T) {
	t.Run("zero value has nil fields", func(t *testing.T) {
		var r Run
		if r.RunID != nil {
			t.Error("RunID should be nil by default")
		}
		if r.RunStatus != nil {
			t.Error("RunStatus should be nil by default")
		}
	})

	t.Run("accepts valid status and ID", func(t *testing.T) {
		id := "run-123"
		status := RunStatusRunning
		r := Run{
			RunID:     &id,
			RunStatus: &status,
		}
		if *r.RunID != "run-123" {
			t.Errorf("RunID = %q, want %q", *r.RunID, "run-123")
		}
		if *r.RunStatus != RunStatusRunning {
			t.Errorf("RunStatus = %q, want %q", *r.RunStatus, RunStatusRunning)
		}
	})

	t.Run("RunID is optional (omitempty)", func(t *testing.T) {
		status := RunStatusCompleted
		r := Run{RunStatus: &status}
		if r.RunID != nil {
			t.Error("RunID should be nil when not set")
		}
		if *r.RunStatus != RunStatusCompleted {
			t.Errorf("RunStatus = %q, want %q", *r.RunStatus, RunStatusCompleted)
		}
	})

	t.Run("RunStatus is optional (omitempty)", func(t *testing.T) {
		id := "run-456"
		r := Run{RunID: &id}
		if r.RunStatus != nil {
			t.Error("RunStatus should be nil when not set")
		}
		if *r.RunID != "run-456" {
			t.Errorf("RunID = %q, want %q", *r.RunID, "run-456")
		}
	})
}
