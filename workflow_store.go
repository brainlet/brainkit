package brainkit

import (
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/workflow"
)

// workflowStoreAdapter bridges SQLiteStore to workflow.RunStore.
type workflowStoreAdapter struct {
	store *SQLiteStore
}

func (a *workflowStoreAdapter) SaveRun(run workflow.WorkflowRun) error {
	completedAt := ""
	if run.CompletedAt != nil {
		completedAt = run.CompletedAt.Format(time.RFC3339)
	}
	inputStr := "{}"
	if len(run.Input) > 0 {
		inputStr = string(run.Input)
	}
	_, err := a.store.db.Exec(
		`INSERT OR REPLACE INTO workflow_runs
		 (workflow_id, run_id, status, input, output, current_step, started_at, completed_at, suspended_event, suspended_timeout, error, retry_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.WorkflowID, run.RunID, string(run.Status), inputStr, run.Output,
		run.CurrentStep, run.StartedAt.Format(time.RFC3339), completedAt,
		run.SuspendedEvent, run.SuspendedTimeout, run.Error, run.RetryCount,
	)
	return err
}

func (a *workflowStoreAdapter) LoadRun(runID string) (*workflow.WorkflowRun, error) {
	var run workflow.WorkflowRun
	var status, inputStr, startedAtStr, completedAtStr string
	err := a.store.db.QueryRow(
		`SELECT workflow_id, run_id, status, input, output, current_step, started_at, completed_at, suspended_event, suspended_timeout, error, retry_count
		 FROM workflow_runs WHERE run_id = ?`, runID,
	).Scan(&run.WorkflowID, &run.RunID, &status, &inputStr, &run.Output,
		&run.CurrentStep, &startedAtStr, &completedAtStr,
		&run.SuspendedEvent, &run.SuspendedTimeout, &run.Error, &run.RetryCount)
	if err != nil {
		return nil, err
	}
	run.Status = workflow.RunStatus(status)
	run.Input = json.RawMessage(inputStr)
	run.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
	if completedAtStr != "" {
		t, _ := time.Parse(time.RFC3339, completedAtStr)
		run.CompletedAt = &t
	}
	return &run, nil
}

func (a *workflowStoreAdapter) LoadRunsByWorkflow(workflowID string) ([]workflow.WorkflowRun, error) {
	rows, err := a.store.db.Query(
		`SELECT workflow_id, run_id, status, input, output, current_step, started_at, completed_at, error, retry_count
		 FROM workflow_runs WHERE workflow_id = ? ORDER BY started_at`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []workflow.WorkflowRun
	for rows.Next() {
		var run workflow.WorkflowRun
		var status, inputStr, startedAtStr, completedAtStr string
		if err := rows.Scan(&run.WorkflowID, &run.RunID, &status, &inputStr, &run.Output,
			&run.CurrentStep, &startedAtStr, &completedAtStr, &run.Error, &run.RetryCount); err != nil {
			return nil, err
		}
		run.Status = workflow.RunStatus(status)
		run.Input = json.RawMessage(inputStr)
		run.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		if completedAtStr != "" {
			t, _ := time.Parse(time.RFC3339, completedAtStr)
			run.CompletedAt = &t
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (a *workflowStoreAdapter) LoadActiveRuns() ([]workflow.WorkflowRun, error) {
	rows, err := a.store.db.Query(
		`SELECT workflow_id, run_id, status, input, output, current_step, started_at, completed_at, suspended_event, suspended_timeout, error, retry_count
		 FROM workflow_runs WHERE status IN ('running', 'suspended') ORDER BY started_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []workflow.WorkflowRun
	for rows.Next() {
		var run workflow.WorkflowRun
		var status, inputStr, startedAtStr, completedAtStr string
		if err := rows.Scan(&run.WorkflowID, &run.RunID, &status, &inputStr, &run.Output,
			&run.CurrentStep, &startedAtStr, &completedAtStr,
			&run.SuspendedEvent, &run.SuspendedTimeout, &run.Error, &run.RetryCount); err != nil {
			return nil, err
		}
		run.Status = workflow.RunStatus(status)
		run.Input = json.RawMessage(inputStr)
		run.StartedAt, _ = time.Parse(time.RFC3339, startedAtStr)
		if completedAtStr != "" {
			t, _ := time.Parse(time.RFC3339, completedAtStr)
			run.CompletedAt = &t
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}

func (a *workflowStoreAdapter) DeleteRun(runID string) error {
	_, err := a.store.db.Exec("DELETE FROM workflow_runs WHERE run_id = ?", runID)
	return err
}

func (a *workflowStoreAdapter) SaveJournalEntry(workflowID, runID string, entry workflow.JournalEntry) error {
	callsJSON, _ := json.Marshal(entry.Calls)
	completedAt := ""
	if entry.CompletedAt != nil {
		completedAt = entry.CompletedAt.Format(time.RFC3339)
	}
	_, err := a.store.db.Exec(
		`INSERT OR REPLACE INTO workflow_journal
		 (workflow_id, run_id, step_index, step_name, status, calls, started_at, completed_at, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		workflowID, runID, entry.StepIndex, entry.StepName, entry.Status,
		string(callsJSON), entry.StartedAt.Format(time.RFC3339), completedAt, entry.Error,
	)
	return err
}

func (a *workflowStoreAdapter) LoadJournalEntries(workflowID, runID string) ([]workflow.JournalEntry, error) {
	rows, err := a.store.db.Query(
		`SELECT step_index, step_name, status, calls, started_at, completed_at, error
		 FROM workflow_journal WHERE workflow_id = ? AND run_id = ? ORDER BY step_index`,
		workflowID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []workflow.JournalEntry
	for rows.Next() {
		var e workflow.JournalEntry
		var callsStr, startedStr, completedStr string
		if err := rows.Scan(&e.StepIndex, &e.StepName, &e.Status, &callsStr, &startedStr, &completedStr, &e.Error); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(callsStr), &e.Calls)
		e.StartedAt, _ = time.Parse(time.RFC3339, startedStr)
		if completedStr != "" {
			t, _ := time.Parse(time.RFC3339, completedStr)
			e.CompletedAt = &t
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (a *workflowStoreAdapter) DeleteJournalEntries(workflowID, runID string) error {
	_, err := a.store.db.Exec("DELETE FROM workflow_journal WHERE workflow_id = ? AND run_id = ?", workflowID, runID)
	return err
}
