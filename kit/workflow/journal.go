package workflow

import (
	"encoding/json"
	"sync"
	"time"
)

// Journal tracks the execution history of a workflow run for durable replay.
// Thread-safe — accessed from host function callbacks during WASM execution.
type Journal struct {
	WorkflowID string
	RunID      string

	mu      sync.Mutex
	entries []JournalEntry
	current *JournalEntry // the step currently being executed

	// Replay state
	replayIndex int  // which step we're replaying
	replaying   bool // true during replay phase
	callIndex   int  // within current step, which call we're replaying
}

// NewJournal creates a fresh journal for a new run.
func NewJournal(workflowID, runID string) *Journal {
	return &Journal{
		WorkflowID: workflowID,
		RunID:      runID,
	}
}

// NewJournalFromEntries creates a journal pre-loaded with entries for replay.
func NewJournalFromEntries(workflowID, runID string, entries []JournalEntry) *Journal {
	return &Journal{
		WorkflowID: workflowID,
		RunID:      runID,
		entries:    entries,
		replaying:  len(entries) > 0,
	}
}

// MarkStep records the start of a new durable step.
// During replay, advances to the next recorded step.
// When replay catches up to unrecorded territory, switches to live execution.
func (j *Journal) MarkStep(name string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// Complete current step if any
	if j.current != nil && j.current.Status == "pending" {
		now := time.Now()
		j.current.CompletedAt = &now
		j.current.Status = "completed"
	}

	stepIndex := len(j.entries)

	// During replay: check if this step was already recorded
	if j.replaying && j.replayIndex < len(j.entries) {
		existing := &j.entries[j.replayIndex]
		if existing.StepName == name {
			j.current = existing
			j.replayIndex++
			j.callIndex = 0
			// Don't set replaying=false yet — the current step may still have
			// recorded calls to replay. We switch to live when GetRecordedResult
			// finds no matching call.
			return
		}
		// Step name mismatch — non-determinism detected. Switch to live.
		j.replaying = false
	}

	// All recorded steps exhausted — switch to live
	if j.replaying && j.replayIndex >= len(j.entries) {
		j.replaying = false
	}

	// Live execution: create new entry
	entry := JournalEntry{
		StepName:  name,
		StepIndex: stepIndex,
		Status:    "pending",
		StartedAt: time.Now(),
	}
	j.entries = append(j.entries, entry)
	j.current = &j.entries[len(j.entries)-1]
	j.callIndex = 0
}

// GetRecordedResult checks if a host function call was already recorded in the journal.
// Returns (result, true) if replaying and the call matches. Returns (nil, false) for live execution.
func (j *Journal) GetRecordedResult(module, name string, args json.RawMessage) (json.RawMessage, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if !j.replaying || j.current == nil {
		return nil, false
	}

	funcName := module + "." + name
	if j.callIndex < len(j.current.Calls) {
		recorded := j.current.Calls[j.callIndex]
		if recorded.Function == funcName {
			j.callIndex++
			if recorded.Error != "" {
				return nil, false // let the caller handle the error path
			}
			return recorded.Result, true
		}
	}

	// No matching record — we've caught up
	j.replaying = false
	return nil, false
}

// RecordCall records a host function call and its result in the current step.
func (j *Journal) RecordCall(module, name string, args, result json.RawMessage, err error, duration time.Duration) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.current == nil {
		return
	}

	record := HostCallRecord{
		Function: module + "." + name,
		Args:     args,
		Result:   result,
		Duration: duration,
	}
	if err != nil {
		record.Error = err.Error()
	}

	j.current.Calls = append(j.current.Calls, record)
}

// MarkSuspended records that the workflow is suspended waiting for an event.
func (j *Journal) MarkSuspended(eventTopic string, timeoutSeconds int) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.current != nil {
		j.current.Status = "suspended"
	}
}

// MarkCompleted marks the current step and overall journal as completed.
func (j *Journal) MarkCompleted() {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.current != nil {
		now := time.Now()
		j.current.CompletedAt = &now
		j.current.Status = "completed"
	}
}

// MarkFailed marks the current step as failed.
func (j *Journal) MarkFailed(err string) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if j.current != nil {
		now := time.Now()
		j.current.CompletedAt = &now
		j.current.Status = "failed"
		j.current.Error = err
	}
}

// Entries returns a copy of all journal entries.
func (j *Journal) Entries() []JournalEntry {
	j.mu.Lock()
	defer j.mu.Unlock()

	cp := make([]JournalEntry, len(j.entries))
	copy(cp, j.entries)
	return cp
}

// IsReplaying returns true if the journal is still in replay mode.
func (j *Journal) IsReplaying() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.replaying
}

// CurrentStepIndex returns the index of the current step.
func (j *Journal) CurrentStepIndex() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.current != nil {
		return j.current.StepIndex
	}
	return len(j.entries)
}
