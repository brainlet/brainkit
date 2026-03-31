package workflow

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// inMemoryRunStore for testing — no SQLite dependency.
type inMemoryRunStore struct {
	mu       sync.Mutex
	runs     map[string]WorkflowRun
	journals map[string][]JournalEntry // key: workflowID+runID
}

func newInMemoryRunStore() *inMemoryRunStore {
	return &inMemoryRunStore{
		runs:     make(map[string]WorkflowRun),
		journals: make(map[string][]JournalEntry),
	}
}

func (s *inMemoryRunStore) SaveRun(run WorkflowRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.RunID] = run
	return nil
}

func (s *inMemoryRunStore) LoadRun(runID string) (*WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok {
		return nil, nil
	}
	return &run, nil
}

func (s *inMemoryRunStore) LoadRunsByWorkflow(workflowID string) ([]WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []WorkflowRun
	for _, r := range s.runs {
		if r.WorkflowID == workflowID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (s *inMemoryRunStore) LoadActiveRuns() ([]WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []WorkflowRun
	for _, r := range s.runs {
		if r.Status == RunRunning || r.Status == RunSuspended {
			result = append(result, r)
		}
	}
	return result, nil
}

func (s *inMemoryRunStore) DeleteRun(runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, runID)
	return nil
}

func (s *inMemoryRunStore) SaveJournalEntry(workflowID, runID string, entry JournalEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := workflowID + ":" + runID
	s.journals[key] = append(s.journals[key], entry)
	return nil
}

func (s *inMemoryRunStore) LoadJournalEntries(workflowID, runID string) ([]JournalEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := workflowID + ":" + runID
	return s.journals[key], nil
}

func (s *inMemoryRunStore) DeleteJournalEntries(workflowID, runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.journals, workflowID+":"+runID)
	return nil
}

func TestEngine_RegisterAndListWorkflows(t *testing.T) {
	reg := NewHostFunctionRegistry()
	engine := NewEngine(reg, nil)

	engine.RegisterWorkflow(WorkflowDef{
		ID:        "wf-1",
		Name:      "test-workflow",
		EntryFunc: "run",
		Timeout:   10 * time.Second,
	})

	engine.RegisterWorkflow(WorkflowDef{
		ID:        "wf-2",
		Name:      "another",
		EntryFunc: "process",
	})

	// Both registered
	engine.mu.Lock()
	count := len(engine.workflows)
	engine.mu.Unlock()
	if count != 2 {
		t.Fatalf("expected 2 workflows, got %d", count)
	}

	// Unregister
	engine.UnregisterWorkflow("wf-1")
	engine.mu.Lock()
	count = len(engine.workflows)
	engine.mu.Unlock()
	if count != 1 {
		t.Fatalf("expected 1 after unregister, got %d", count)
	}
}

func TestEngine_RunNotFound(t *testing.T) {
	engine := NewEngine(NewHostFunctionRegistry(), nil)
	_, err := engine.Run(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestEngine_GetRunNotFound(t *testing.T) {
	engine := NewEngine(NewHostFunctionRegistry(), nil)
	_, err := engine.GetRun("nonexistent-run")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
}

func TestEngine_CancelNotActive(t *testing.T) {
	engine := NewEngine(NewHostFunctionRegistry(), nil)
	err := engine.CancelRun("nonexistent-run")
	if err == nil {
		t.Fatal("expected error for nonexistent run")
	}
}

func TestEngine_ListRunsEmpty(t *testing.T) {
	engine := NewEngine(NewHostFunctionRegistry(), nil)
	runs := engine.ListRuns()
	if len(runs) != 0 {
		t.Fatalf("expected 0 runs, got %d", len(runs))
	}
}

func TestEngine_RunStoresPersistence(t *testing.T) {
	store := newInMemoryRunStore()
	engine := NewEngine(NewHostFunctionRegistry(), store)

	// Register a workflow with no binary (will fail at compile but run state is saved)
	engine.RegisterWorkflow(WorkflowDef{
		ID:        "persist-test",
		Name:      "persist",
		Binary:    []byte{}, // empty — will fail
		EntryFunc: "run",
		Timeout:   5 * time.Second,
	})

	runID, err := engine.Run(context.Background(), "persist-test", json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatal("run:", err)
	}
	if runID == "" {
		t.Fatal("expected non-empty runID")
	}

	// Wait for the run to finish (it should fail since binary is empty)
	time.Sleep(500 * time.Millisecond)

	// Verify run was persisted to store
	stored, err := store.LoadRun(runID)
	if err != nil {
		t.Fatal("load run:", err)
	}
	if stored == nil {
		t.Fatal("run not persisted to store")
	}
	if stored.Status != RunFailed {
		t.Fatalf("expected failed status, got %q", stored.Status)
	}
	if stored.WorkflowID != "persist-test" {
		t.Fatalf("expected workflowID 'persist-test', got %q", stored.WorkflowID)
	}
}

func TestEngine_GetJournalNotFound(t *testing.T) {
	engine := NewEngine(NewHostFunctionRegistry(), nil)
	_, err := engine.GetJournal("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}
}
