package workflow

import (
	"context"
	"log"
)

// RestoreActiveRuns loads workflow runs that were running or suspended
// when the previous process exited, and resumes them.
// Running workflows are replayed from their journal.
// Suspended workflows re-register their event listener.
func (e *Engine) RestoreActiveRuns(ctx context.Context) int {
	if e.store == nil {
		return 0
	}

	runs, err := e.store.LoadActiveRuns()
	if err != nil {
		log.Printf("[workflow] warning: failed to load active runs: %v", err)
		return 0
	}
	if len(runs) == 0 {
		return 0
	}

	restored := 0
	for _, run := range runs {
		// Load the workflow definition
		e.mu.Lock()
		def, ok := e.workflows[run.WorkflowID]
		if !ok {
			e.mu.Unlock()
			log.Printf("[workflow] skipping run %s: workflow %q not registered", run.RunID, run.WorkflowID)
			continue
		}
		binary := make([]byte, len(def.Binary))
		copy(binary, def.Binary)
		entryFunc := def.EntryFunc
		timeout := def.Timeout
		e.mu.Unlock()

		// Load journal entries for replay
		entries, err := e.store.LoadJournalEntries(run.WorkflowID, run.RunID)
		if err != nil {
			log.Printf("[workflow] skipping run %s: failed to load journal: %v", run.RunID, err)
			continue
		}

		// Create journal pre-loaded with recorded entries for replay
		journal := NewJournalFromEntries(run.WorkflowID, run.RunID, entries)

		runCtx, runCancel := context.WithTimeout(ctx, timeout)
		ar := &activeRun{
			run:     run,
			journal: journal,
			cancel:  runCancel,
		}
		// Reset status to running for replay
		ar.run.Status = RunReplaying

		e.mu.Lock()
		e.runs[run.RunID] = ar
		e.mu.Unlock()

		if e.store != nil {
			ar.run.Status = RunReplaying
			e.store.SaveRun(ar.run)
		}

		// Replay in background
		go func(ar *activeRun, binary []byte, entryFunc string) {
			defer runCancel()
			e.executeWorkflow(runCtx, ar, binary, entryFunc, ar.run.Input)
		}(ar, binary, entryFunc)

		restored++
		log.Printf("[workflow] restoring run %s (workflow: %s, step: %d, journal: %d entries)",
			run.RunID, run.WorkflowID, run.CurrentStep, len(entries))
	}

	if restored > 0 {
		log.Printf("[workflow] restored %d active workflow runs", restored)
	}
	return restored
}
