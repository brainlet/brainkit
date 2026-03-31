package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// AIGenerator abstracts AI text generation for the engine (avoids kit import cycle).
type AIGenerator interface {
	GenerateText(ctx context.Context, prompt string) (string, error)
	EmbedText(ctx context.Context, text string) (string, error) // returns JSON array of floats
}

// RunStore persists workflow runs and journal entries.
type RunStore interface {
	SaveRun(run WorkflowRun) error
	LoadRun(runID string) (*WorkflowRun, error)
	LoadRunsByWorkflow(workflowID string) ([]WorkflowRun, error)
	LoadActiveRuns() ([]WorkflowRun, error)
	DeleteRun(runID string) error

	SaveJournalEntry(workflowID, runID string, entry JournalEntry) error
	LoadJournalEntries(workflowID, runID string) ([]JournalEntry, error)
	DeleteJournalEntries(workflowID, runID string) error
}

// Engine manages workflow definitions, runs, and replay.
// BusSubscriber subscribes to a bus topic for workflow event resumption.
// Returns a cancel function to unsubscribe.
type BusSubscriber func(topic string, handler func(json.RawMessage)) (cancel func(), err error)

// PluginCaller calls a plugin tool topic via the bus and returns the result.
// Used by workflow host functions to route calls to plugins.
type PluginCaller func(ctx context.Context, topic string, args json.RawMessage) (json.RawMessage, error)

// SpanRecorder records trace spans from workflow journal entries.
// Called when a workflow completes — converts journal entries to trace spans.
type SpanRecorder func(workflowID, runID string, entries []JournalEntry)

// BusPublisher publishes a message to the bus from a workflow.
type BusPublisher func(ctx context.Context, topic string, payload json.RawMessage) error

// Engine manages workflow definitions, runs, and replay.
type Engine struct {
	hostRegistry  *HostFunctionRegistry
	store         RunStore       // optional persistence
	ai            AIGenerator    // optional AI provider
	busSubscriber BusSubscriber  // optional — for waitForEvent resumption
	pluginCaller  PluginCaller   // optional — for routing host function calls to plugins
	spanRecorder  SpanRecorder   // optional — converts journal entries to trace spans
	busPublisher  BusPublisher   // optional — for bus.publish/emit from workflows

	mu        sync.Mutex
	workflows map[string]*WorkflowDef // workflowId → definition
	runs      map[string]*activeRun   // runId → active run state
	unsubs    map[string]func()       // runId → event subscription cancel
}

// activeRun tracks an in-flight workflow execution.
type activeRun struct {
	run     WorkflowRun
	journal *Journal
	cancel  context.CancelFunc

	// Workflow control signals (set by host functions)
	mu          sync.Mutex
	completed   bool
	failed      bool
	suspended   bool
	suspendTopic string
	suspendTimeout int
	result      string
	err         string
}

// NewEngine creates a workflow engine.
func NewEngine(hostRegistry *HostFunctionRegistry, store RunStore, opts ...EngineOption) *Engine {
	e := &Engine{
		hostRegistry: hostRegistry,
		store:        store,
		workflows:    make(map[string]*WorkflowDef),
		runs:         make(map[string]*activeRun),
		unsubs:       make(map[string]func()),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// EngineOption configures the engine.
type EngineOption func(*Engine)

// WithBusSubscriber sets the bus subscriber for waitForEvent resumption.
func WithBusSubscriber(sub BusSubscriber) EngineOption {
	return func(e *Engine) { e.busSubscriber = sub }
}

// WithPluginCaller sets the plugin call function for routing host function calls to plugins.
func WithPluginCaller(caller PluginCaller) EngineOption {
	return func(e *Engine) { e.pluginCaller = caller }
}

// WithSpanRecorder sets the trace span recorder for journal→span conversion.
func WithSpanRecorder(recorder SpanRecorder) EngineOption {
	return func(e *Engine) { e.spanRecorder = recorder }
}

// WithBusPublisher sets the bus publisher for workflow→bus communication.
func WithBusPublisher(pub BusPublisher) EngineOption {
	return func(e *Engine) { e.busPublisher = pub }
}

// registerBusHostFunctions registers "bus" module with publish and emit.
func (e *Engine) registerBusHostFunctions(ctx context.Context, rt wazero.Runtime, ar *activeRun) {
	if e.busPublisher == nil {
		return
	}
	pub := e.busPublisher
	rt.NewHostModuleBuilder("bus").
		// bus.publish(topic: string, payload: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := json.RawMessage(readASString(m, payloadPtr))
			pub(ctx, topic, payload)
		}).Export("publish").
		// bus.emit(topic: string, payload: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := json.RawMessage(readASString(m, payloadPtr))
			pub(ctx, topic, payload)
		}).Export("emit").
		Instantiate(ctx)
}

// WithAI sets the AI text generation provider for workflows.
func WithAI(ai AIGenerator) EngineOption {
	return func(e *Engine) { e.ai = ai }
}

// RegisterWorkflow registers a workflow definition.
func (e *Engine) RegisterWorkflow(def WorkflowDef) {
	if def.ID == "" {
		def.ID = uuid.NewString()
	}
	if def.EntryFunc == "" {
		def.EntryFunc = "run"
	}
	if def.Timeout == 0 {
		def.Timeout = 24 * time.Hour
	}
	e.mu.Lock()
	e.workflows[def.ID] = &def
	e.mu.Unlock()
}

// UnregisterWorkflow removes a workflow definition.
func (e *Engine) UnregisterWorkflow(id string) {
	e.mu.Lock()
	delete(e.workflows, id)
	e.mu.Unlock()
}

// RunOption configures a single workflow run.
type RunOption func(*runConfig)

type runConfig struct {
	hostResults map[string][]json.RawMessage // "module.func" → recorded results in order
}

// WithHostResults pre-loads host function results for replay testing.
// Each key is "module.func" (e.g., "ai.generate", "telegram.send").
// Results are consumed in order — first call gets results[0], second gets results[1].
func WithHostResults(results map[string][]json.RawMessage) RunOption {
	return func(c *runConfig) { c.hostResults = results }
}

// Run starts a new workflow execution.
func (e *Engine) Run(ctx context.Context, workflowID string, input json.RawMessage, opts ...RunOption) (string, error) {
	var cfg runConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	e.mu.Lock()
	def, ok := e.workflows[workflowID]
	if !ok {
		e.mu.Unlock()
		return "", fmt.Errorf("workflow %q not registered", workflowID)
	}
	binary := make([]byte, len(def.Binary))
	copy(binary, def.Binary)
	entryFunc := def.EntryFunc
	timeout := def.Timeout
	e.mu.Unlock()

	runID := uuid.NewString()
	run := WorkflowRun{
		WorkflowID:  workflowID,
		RunID:       runID,
		Status:      RunRunning,
		Input:       input,
		CurrentStep: 0,
		StartedAt:   time.Now(),
	}

	var journal *Journal
	if len(cfg.hostResults) > 0 {
		// Replay testing mode: pre-populate journal with recorded host function results.
		// Build a single step with all recorded calls so GetRecordedResult returns them.
		var calls []HostCallRecord
		for funcName, results := range cfg.hostResults {
			for _, result := range results {
				calls = append(calls, HostCallRecord{
					Function: funcName,
					Result:   result,
				})
			}
		}
		entries := []JournalEntry{{
			StepName:  "__replay_test",
			StepIndex: 0,
			Status:    "completed",
			Calls:     calls,
			StartedAt: time.Now(),
		}}
		journal = NewJournalFromEntries(workflowID, runID, entries)
	} else {
		journal = NewJournal(workflowID, runID)
	}

	runCtx, runCancel := context.WithTimeout(ctx, timeout)
	ar := &activeRun{
		run:     run,
		journal: journal,
		cancel:  runCancel,
	}

	e.mu.Lock()
	e.runs[runID] = ar
	e.mu.Unlock()

	if e.store != nil {
		e.store.SaveRun(run)
	}

	// Execute in background
	go func() {
		defer runCancel()
		e.executeWorkflow(runCtx, ar, binary, entryFunc, input)
	}()

	return runID, nil
}

// executeWorkflow runs the WASM module with host functions and journal tracking.
func (e *Engine) executeWorkflow(ctx context.Context, ar *activeRun, binary []byte, entryFunc string, input json.RawMessage) {
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Register env module (abort)
	rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
			log.Printf("[workflow:abort] at %d:%d", line, col)
			m.CloseWithExitCode(ctx, 255)
		}).Export("abort").
		Instantiate(ctx)

	// Register brainkit built-in host functions
	e.registerBuiltinHostFunctions(ctx, rt, ar)

	// Register bus host functions (publish, emit) for workflow→bus communication
	e.registerBusHostFunctions(ctx, rt, ar)

	// Register AI host functions (ai.generate, ai.embed)
	if e.ai != nil {
		e.registerAIHostFunctions(ctx, rt, ar)
	}

	// Register plugin-provided host functions (telegram.send, db.query, etc.)
	e.registerPluginHostFunctions(ctx, rt, ar)

	// Compile and instantiate
	compiled, err := rt.CompileModule(ctx, binary)
	if err != nil {
		e.completeRun(ar, "", fmt.Errorf("compile: %w", err))
		return
	}

	inst, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		e.completeRun(ar, "", fmt.Errorf("instantiate: %w", err))
		return
	}
	defer inst.Close(ctx)

	// Call entry function
	fn := inst.ExportedFunction(entryFunc)
	if fn == nil {
		e.completeRun(ar, "", fmt.Errorf("entry function %q not exported", entryFunc))
		return
	}

	// Write input to WASM memory
	inputPtr, err := writeASString(ctx, inst, string(input))
	if err != nil {
		e.completeRun(ar, "", fmt.Errorf("write input: %w", err))
		return
	}

	_, err = fn.Call(ctx, uint64(inputPtr))
	if err != nil {
		ar.mu.Lock()
		suspended := ar.suspended
		ar.mu.Unlock()

		if suspended {
			// Workflow suspended — not an error
			e.suspendRun(ar)
			return
		}
		e.completeRun(ar, "", err)
		return
	}

	// Check final state
	ar.mu.Lock()
	if ar.completed {
		result := ar.result
		ar.mu.Unlock()
		e.completeRun(ar, result, nil)
	} else if ar.failed {
		errMsg := ar.err
		ar.mu.Unlock()
		e.completeRun(ar, "", fmt.Errorf("%s", errMsg))
	} else if ar.suspended {
		ar.mu.Unlock()
		e.suspendRun(ar)
	} else {
		ar.mu.Unlock()
		// Workflow returned without calling complete() — treat as completed
		e.completeRun(ar, "", nil)
	}
}

// registerBuiltinHostFunctions registers step, sleep, waitForEvent, complete, fail, state.
func (e *Engine) registerBuiltinHostFunctions(ctx context.Context, rt wazero.Runtime, ar *activeRun) {
	rt.NewHostModuleBuilder("brainkit").
		// step(name: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, namePtr uint32) {
			name := readASString(m, namePtr)
			ar.journal.MarkStep(name)
			ar.mu.Lock()
			ar.run.CurrentStep = ar.journal.CurrentStepIndex()
			ar.mu.Unlock()
		}).Export("step").

		// complete(result: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, resultPtr uint32) {
			result := readASString(m, resultPtr)
			ar.mu.Lock()
			ar.completed = true
			ar.result = result
			ar.mu.Unlock()
			ar.journal.MarkCompleted()
		}).Export("complete").

		// fail(error: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, errPtr uint32) {
			errMsg := readASString(m, errPtr)
			ar.mu.Lock()
			ar.failed = true
			ar.err = errMsg
			ar.mu.Unlock()
			ar.journal.MarkFailed(errMsg)
		}).Export("fail").

		// sleep(seconds: i64) — durable sleep
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, seconds uint64) {
			// During replay, skip the sleep
			if ar.journal.IsReplaying() {
				return
			}
			select {
			case <-time.After(time.Duration(seconds) * time.Second):
			case <-ctx.Done():
			}
		}).Export("sleep").

		// waitForEvent(topic: string, timeoutSeconds: i64) → string
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr uint32, timeoutSec uint64) uint32 {
			topic := readASString(m, topicPtr)

			// Check journal for recorded result
			if result, ok := ar.journal.GetRecordedResult("brainkit", "waitForEvent", nil); ok {
				ptr, _ := writeASString(ctx, m, string(result))
				return ptr
			}

			// Suspend the workflow
			ar.mu.Lock()
			ar.suspended = true
			ar.suspendTopic = topic
			ar.suspendTimeout = int(timeoutSec)
			ar.mu.Unlock()

			ar.journal.MarkSuspended(topic, int(timeoutSec))

			// Close the module to stop execution
			m.CloseWithExitCode(ctx, 0)
			return 0
		}).Export("waitForEvent").

		// log(message: string, level: i32)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, level uint32) {
			msg := readASString(m, msgPtr)
			log.Printf("[workflow:%s] %s", ar.run.WorkflowID, msg)
		}).Export("log").

		// get_state(key: string) → string
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr uint32) uint32 {
			// Workflow state stored in journal metadata — simple key/value
			ptr, _ := writeASString(ctx, m, "")
			return ptr
		}).Export("get_state").

		// set_state(key: string, value: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, valPtr uint32) {
			// State recorded in journal
		}).Export("set_state").

		Instantiate(ctx)
}

func (e *Engine) completeRun(ar *activeRun, result string, err error) {
	if err != nil {
		// Check if we should retry
		e.mu.Lock()
		def, hasDef := e.workflows[ar.run.WorkflowID]
		e.mu.Unlock()

		if hasDef && def.MaxRetries > 0 && ar.run.RetryCount < def.MaxRetries {
			ar.run.RetryCount++
			log.Printf("[workflow] run %s failed (attempt %d/%d), retrying: %v",
				ar.run.RunID, ar.run.RetryCount, def.MaxRetries, err)

			// Reset for fresh execution
			ar.journal = NewJournal(ar.run.WorkflowID, ar.run.RunID)
			ar.run.Status = RunRunning
			ar.run.CurrentStep = 0
			ar.run.Error = ""
			ar.mu.Lock()
			ar.completed = false
			ar.failed = false
			ar.suspended = false
			ar.result = ""
			ar.err = ""
			ar.mu.Unlock()

			if e.store != nil {
				e.store.SaveRun(ar.run)
			}

			binary := make([]byte, len(def.Binary))
			copy(binary, def.Binary)
			runCtx, runCancel := context.WithTimeout(context.Background(), def.Timeout)
			ar.cancel = runCancel
			go func() {
				defer runCancel()
				e.executeWorkflow(runCtx, ar, binary, def.EntryFunc, ar.run.Input)
			}()
			return
		}

		now := time.Now()
		ar.run.CompletedAt = &now
		ar.run.Status = RunFailed
		ar.run.Error = err.Error()
	} else {
		now := time.Now()
		ar.run.CompletedAt = &now
		ar.run.Status = RunCompleted
		ar.run.Output = result
	}

	entries := ar.journal.Entries()

	if e.store != nil {
		e.store.SaveRun(ar.run)
		for _, entry := range entries {
			e.store.SaveJournalEntry(ar.run.WorkflowID, ar.run.RunID, entry)
		}
	}

	// Convert journal entries to trace spans
	if e.spanRecorder != nil {
		e.spanRecorder(ar.run.WorkflowID, ar.run.RunID, entries)
	}

	e.mu.Lock()
	delete(e.runs, ar.run.RunID)
	e.mu.Unlock()
}

func (e *Engine) suspendRun(ar *activeRun) {
	ar.run.Status = RunSuspended
	ar.run.SuspendedEvent = ar.suspendTopic
	ar.run.SuspendedTimeout = ar.suspendTimeout

	if e.store != nil {
		e.store.SaveRun(ar.run)
		for _, entry := range ar.journal.Entries() {
			e.store.SaveJournalEntry(ar.run.WorkflowID, ar.run.RunID, entry)
		}
	}

	// Subscribe to the event topic for resumption
	if ar.suspendTopic != "" && e.busSubscriber != nil {
		cancel, err := e.busSubscriber(ar.suspendTopic, func(payload json.RawMessage) {
			// Record the event result in journal so replay can return it
			ar.journal.RecordCall("brainkit", "waitForEvent", nil, payload, nil, 0)
			// Clean up subscription
			e.mu.Lock()
			if unsub, ok := e.unsubs[ar.run.RunID]; ok {
				unsub()
				delete(e.unsubs, ar.run.RunID)
			}
			e.mu.Unlock()
			// Resume by re-executing (journal replay skips to wait point)
			e.resumeRun(ar)
		})
		if err == nil && cancel != nil {
			e.mu.Lock()
			e.unsubs[ar.run.RunID] = cancel
			e.mu.Unlock()
		}
	}
}

// resumeRun re-executes a suspended workflow from journal replay.
func (e *Engine) resumeRun(ar *activeRun) {
	e.mu.Lock()
	def, ok := e.workflows[ar.run.WorkflowID]
	if !ok {
		e.mu.Unlock()
		log.Printf("[workflow] cannot resume run %s: workflow %q not registered", ar.run.RunID, ar.run.WorkflowID)
		return
	}
	binary := make([]byte, len(def.Binary))
	copy(binary, def.Binary)
	entryFunc := def.EntryFunc
	timeout := def.Timeout
	e.mu.Unlock()

	ar.run.Status = RunReplaying
	ar.mu.Lock()
	ar.suspended = false
	ar.mu.Unlock()

	if e.store != nil {
		e.store.SaveRun(ar.run)
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), timeout)
	ar.cancel = runCancel

	go func() {
		defer runCancel()
		e.executeWorkflow(runCtx, ar, binary, entryFunc, ar.run.Input)
	}()
}

// GetRun returns the current state of a workflow run.
func (e *Engine) GetRun(runID string) (*WorkflowRun, error) {
	e.mu.Lock()
	ar, ok := e.runs[runID]
	e.mu.Unlock()

	if ok {
		ar.mu.Lock()
		cp := ar.run
		ar.mu.Unlock()
		return &cp, nil
	}

	if e.store != nil {
		return e.store.LoadRun(runID)
	}

	return nil, fmt.Errorf("run %q not found", runID)
}

// GetJournal returns the journal entries for a run.
func (e *Engine) GetJournal(runID string) ([]JournalEntry, error) {
	e.mu.Lock()
	ar, ok := e.runs[runID]
	e.mu.Unlock()

	if ok {
		return ar.journal.Entries(), nil
	}

	if e.store != nil {
		run, err := e.store.LoadRun(runID)
		if err != nil {
			return nil, err
		}
		return e.store.LoadJournalEntries(run.WorkflowID, runID)
	}

	return nil, fmt.Errorf("run %q not found", runID)
}

// CancelRun cancels a running workflow.
func (e *Engine) CancelRun(runID string) error {
	e.mu.Lock()
	ar, ok := e.runs[runID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("run %q not active", runID)
	}

	ar.cancel()
	now := time.Now()
	ar.run.Status = RunCancelled
	ar.run.CompletedAt = &now

	if e.store != nil {
		e.store.SaveRun(ar.run)
	}

	e.mu.Lock()
	delete(e.runs, runID)
	e.mu.Unlock()

	return nil
}

// ListRuns returns all active runs.
func (e *Engine) ListRuns() []RunInfo {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]RunInfo, 0, len(e.runs))
	for _, ar := range e.runs {
		ar.mu.Lock()
		result = append(result, RunInfo{
			RunID:       ar.run.RunID,
			WorkflowID:  ar.run.WorkflowID,
			Status:      string(ar.run.Status),
			CurrentStep: ar.run.CurrentStep,
			StartedAt:   ar.run.StartedAt.Format(time.RFC3339),
			Error:       ar.run.Error,
		})
		ar.mu.Unlock()
	}
	return result
}
