// Ported from: packages/core/src/workflows/evented/step-executor.test.ts
package evented

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Mock types for StepExecutor tests
// ---------------------------------------------------------------------------

type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockLogger) Debug(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, "DEBUG: "+message)
}
func (m *mockLogger) Info(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, "INFO: "+message)
}
func (m *mockLogger) Warn(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, "WARN: "+message)
}
func (m *mockLogger) Error(message string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, "ERROR: "+message)
}
func (m *mockLogger) TrackException(err *mastraerror.MastraBaseError) {}
func (m *mockLogger) GetTransports() map[string]logger.LoggerTransport {
	return nil
}
func (m *mockLogger) ListLogs(transportID string, params *logger.ListLogsParams) (logger.LogResult, error) {
	return logger.LogResult{}, nil
}
func (m *mockLogger) ListLogsByRunID(args *logger.ListLogsByRunIDFullArgs) (logger.LogResult, error) {
	return logger.LogResult{}, nil
}

type mockPubSub struct {
	mu         sync.Mutex
	published  []mockPublishCall
	subscribed map[string][]func(event any, ack func() error) error
}

type mockPublishCall struct {
	Topic string
	Event any
}

func newMockPubSub() *mockPubSub {
	return &mockPubSub{
		subscribed: make(map[string][]func(event any, ack func() error) error),
	}
}

func (m *mockPubSub) Publish(topic string, event any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, mockPublishCall{Topic: topic, Event: event})
	return nil
}

func (m *mockPubSub) Subscribe(topic string, handler func(event any, ack func() error) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscribed[topic] = append(m.subscribed[topic], handler)
	return nil
}

func (m *mockPubSub) Unsubscribe(topic string, handler func(event any, ack func() error) error) error {
	return nil
}

type mockStorage struct{}

func (m *mockStorage) GetStore(name string) (WorkflowsStore, error) {
	return nil, nil
}

type mockMastra struct {
	log    logger.IMastraLogger
	pubsub PubSub
}

func (m *mockMastra) GetLogger() logger.IMastraLogger {
	return m.log
}

func (m *mockMastra) PubSub() PubSub {
	return m.pubsub
}

func (m *mockMastra) GetStorage() Storage {
	return &mockStorage{}
}

// ---------------------------------------------------------------------------
// NewStepExecutor tests
// ---------------------------------------------------------------------------

func TestNewStepExecutor(t *testing.T) {
	t.Run("should create executor with mastra", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		se := NewStepExecutor(mastra)
		if se == nil {
			t.Fatal("NewStepExecutor returned nil")
		}
	})

	t.Run("should create executor without mastra", func(t *testing.T) {
		se := NewStepExecutor(nil)
		if se == nil {
			t.Fatal("NewStepExecutor returned nil")
		}
	})
}

// ---------------------------------------------------------------------------
// RegisterMastra tests
// ---------------------------------------------------------------------------

func TestStepExecutor_RegisterMastra(t *testing.T) {
	t.Run("should register mastra and update logger", func(t *testing.T) {
		se := NewStepExecutor(nil)
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		se.RegisterMastra(mastra)
		if se.mastra == nil {
			t.Error("mastra not registered")
		}
		if se.log == nil {
			t.Error("logger not updated")
		}
	})

	t.Run("should handle nil mastra", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		se := NewStepExecutor(mastra)
		se.RegisterMastra(nil)
		if se.mastra != nil {
			t.Error("mastra should be nil after registering nil")
		}
	})
}

// ---------------------------------------------------------------------------
// Execute tests
// ---------------------------------------------------------------------------

func TestStepExecutor_Execute(t *testing.T) {
	t.Run("should execute step successfully", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID: "test-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return map[string]any{"result": "success"}, nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{"data": "test"},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "success" {
			t.Errorf("result.Status = %q, want %q", result.Status, "success")
		}
		output, ok := result.Output.(map[string]any)
		if !ok {
			t.Fatalf("result.Output is not map[string]any, got %T", result.Output)
		}
		if output["result"] != "success" {
			t.Errorf("result.Output[result] = %v, want success", output["result"])
		}
	})

	t.Run("should return failed status on execution error", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID: "failing-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return nil, errors.New("step failed")
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "failed" {
			t.Errorf("result.Status = %q, want %q", result.Status, "failed")
		}
		if result.Error == nil {
			t.Error("result.Error is nil, want error")
		}
	})

	t.Run("should handle TripWire error", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		retry := true
		step := &wf.Step{
			ID: "tripwire-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return nil, &TripWire{
					Message:     "rate limit",
					ProcessorID: "proc-1",
					Options:     &TripWireOptions{Retry: &retry},
				}
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "failed" {
			t.Errorf("result.Status = %q, want %q", result.Status, "failed")
		}
		if result.Tripwire == nil {
			t.Fatal("result.Tripwire is nil")
		}
		if result.Tripwire.Reason != "rate limit" {
			t.Errorf("result.Tripwire.Reason = %q, want %q", result.Tripwire.Reason, "rate limit")
		}
		if result.Tripwire.ProcessorID != "proc-1" {
			t.Errorf("result.Tripwire.ProcessorID = %q, want %q", result.Tripwire.ProcessorID, "proc-1")
		}
		if result.Tripwire.Retry == nil || *result.Tripwire.Retry != true {
			t.Error("result.Tripwire.Retry should be true")
		}
	})

	t.Run("should handle suspend", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID: "suspend-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				err := params.Suspend(map[string]any{"reason": "approval needed"}, nil)
				if err != nil {
					return nil, err
				}
				return "pre-suspend output", nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "suspended" {
			t.Errorf("result.Status = %q, want %q", result.Status, "suspended")
		}
		if result.SuspendPayload == nil {
			t.Fatal("result.SuspendPayload is nil")
		}
		spMap, ok := result.SuspendPayload.(map[string]any)
		if !ok {
			t.Fatalf("SuspendPayload is not map[string]any, got %T", result.SuspendPayload)
		}
		if _, hasMeta := spMap["__workflow_meta"]; !hasMeta {
			t.Error("SuspendPayload should contain __workflow_meta")
		}
		if result.SuspendOutput != "pre-suspend output" {
			t.Errorf("SuspendOutput = %v, want 'pre-suspend output'", result.SuspendOutput)
		}
	})

	t.Run("should handle bail", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID: "bail-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				params.Bail(map[string]any{"reason": "something wrong"})
				return nil, nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "bailed" {
			t.Errorf("result.Status = %q, want %q", result.Status, "bailed")
		}
		output, ok := result.Output.(map[string]any)
		if !ok {
			t.Fatalf("result.Output is not map[string]any, got %T", result.Output)
		}
		if output["reason"] != "something wrong" {
			t.Errorf("result.Output[reason] = %v, want 'something wrong'", output["reason"])
		}
	})

	t.Run("should pass input data from foreach index", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		var receivedInput any
		step := &wf.Step{
			ID: "foreach-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				receivedInput = params.InputData
				return params.InputData, nil
			},
		}

		idx := 1
		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       []any{"a", "b", "c"},
			ForeachIdx:  &idx,
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "success" {
			t.Errorf("result.Status = %q, want %q", result.Status, "success")
		}
		if receivedInput != "b" {
			t.Errorf("receivedInput = %v, want 'b'", receivedInput)
		}
	})

	t.Run("should handle step with nil Execute function", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID:      "nil-execute-step",
			Execute: nil,
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "success" {
			t.Errorf("result.Status = %q, want %q", result.Status, "success")
		}
	})

	t.Run("should set timestamps on result", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		before := time.Now().UnixMilli()

		step := &wf.Step{
			ID: "timed-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return "ok", nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		after := time.Now().UnixMilli()

		if result.StartedAt < before || result.StartedAt > after {
			t.Errorf("result.StartedAt = %d, should be between %d and %d", result.StartedAt, before, after)
		}
		if result.EndedAt < before || result.EndedAt > after {
			t.Errorf("result.EndedAt = %d, should be between %d and %d", result.EndedAt, before, after)
		}
	})

	t.Run("should pass state to execute context and support setState", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID: "state-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				stateMap, ok := params.State.(map[string]any)
				if !ok {
					return nil, errors.New("state is not a map")
				}
				_ = params.SetState(map[string]any{"counter": 1, "existing": stateMap["existing"]})
				return "ok", nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{"existing": "value"},
		})

		if result.Status != "success" {
			t.Errorf("result.Status = %q, want %q", result.Status, "success")
		}

		// State update should be reflected in metadata
		meta := result.Metadata
		if meta == nil {
			t.Fatal("result.Metadata is nil")
		}
		state, ok := meta["__state"].(map[string]any)
		if !ok {
			t.Fatal("result.Metadata[__state] is not map[string]any")
		}
		if state["counter"] != 1 {
			t.Errorf("state[counter] = %v, want 1", state["counter"])
		}
	})

	t.Run("should handle resume data", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		var receivedResumeData any
		step := &wf.Step{
			ID: "resume-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				receivedResumeData = params.ResumeData
				return "resumed", nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			ResumeData:  map[string]any{"approved": true},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "success" {
			t.Errorf("result.Status = %q, want %q", result.Status, "success")
		}
		rd, ok := receivedResumeData.(map[string]any)
		if !ok {
			t.Fatalf("receivedResumeData is not map[string]any, got %T", receivedResumeData)
		}
		if rd["approved"] != true {
			t.Errorf("receivedResumeData[approved] = %v, want true", rd["approved"])
		}
	})

	t.Run("should handle nested workflow step with perStep", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID:        "nested-wf-step",
			Component: "WORKFLOW",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return "nested result", nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
			PerStep:     true,
		})

		if result.Status != "paused" {
			t.Errorf("result.Status = %q, want %q", result.Status, "paused")
		}
	})

	t.Run("should provide abort context", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var receivedCtx context.Context
		step := &wf.Step{
			ID: "ctx-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				receivedCtx = params.AbortCtx
				return "ok", nil
			},
		}

		se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
			AbortCtx:    ctx,
			AbortCancel: cancel,
		})

		if receivedCtx == nil {
			t.Error("step did not receive abort context")
		}
	})

	t.Run("should create abort context when not provided", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		var receivedCtx context.Context
		step := &wf.Step{
			ID: "auto-ctx-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				receivedCtx = params.AbortCtx
				return "ok", nil
			},
		}

		se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if receivedCtx == nil {
			t.Error("step did not receive auto-created abort context")
		}
	})

	t.Run("should strip __workflow_meta from suspendPayload on resume", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		var receivedSuspendData any
		step := &wf.Step{
			ID: "strip-meta-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				receivedSuspendData = params.SuspendData
				return "ok", nil
			},
		}

		// Existing step result has suspended status with __workflow_meta
		existingResults := map[string]wf.StepResult{
			"strip-meta-step": {
				Status: "suspended",
				SuspendPayload: map[string]any{
					"__workflow_meta": map[string]any{"runId": "run-1"},
					"userField":      "value",
				},
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			ResumeData:  map[string]any{"approved": true},
			StepResults: existingResults,
			State:       map[string]any{},
		})

		if result.Status != "success" {
			t.Errorf("result.Status = %q, want %q", result.Status, "success")
		}

		// The suspendData passed to execute should NOT have __workflow_meta
		if sd, ok := receivedSuspendData.(map[string]any); ok {
			if _, hasMeta := sd["__workflow_meta"]; hasMeta {
				t.Error("suspendData should not contain __workflow_meta")
			}
			if sd["userField"] != "value" {
				t.Errorf("suspendData[userField] = %v, want 'value'", sd["userField"])
			}
		}
	})

	t.Run("should handle suspend with resume labels", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log, pubsub: newMockPubSub()}
		se := NewStepExecutor(mastra)

		step := &wf.Step{
			ID: "label-suspend-step",
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				err := params.Suspend(
					map[string]any{"reason": "approval needed"},
					&wf.SuspendOptions{ResumeLabel: []string{"manager-approved", "vp-approved"}},
				)
				if err != nil {
					return nil, err
				}
				return nil, nil
			},
		}

		result := se.Execute(ExecuteParams{
			WorkflowID:  "wf-1",
			Step:        step,
			RunID:       "run-1",
			Input:       map[string]any{},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if result.Status != "suspended" {
			t.Errorf("result.Status = %q, want %q", result.Status, "suspended")
		}

		sp, ok := result.SuspendPayload.(map[string]any)
		if !ok {
			t.Fatalf("SuspendPayload is not map[string]any, got %T", result.SuspendPayload)
		}

		meta, ok := sp["__workflow_meta"].(map[string]any)
		if !ok {
			t.Fatal("SuspendPayload[__workflow_meta] is not map[string]any")
		}

		labels, ok := meta["resumeLabels"].(map[string]any)
		if !ok {
			t.Fatal("meta[resumeLabels] is not map[string]any")
		}

		if _, hasLabel := labels["manager-approved"]; !hasLabel {
			t.Error("resumeLabels should contain 'manager-approved'")
		}
		if _, hasLabel := labels["vp-approved"]; !hasLabel {
			t.Error("resumeLabels should contain 'vp-approved'")
		}
	})
}

// ---------------------------------------------------------------------------
// EvaluateConditions tests
// ---------------------------------------------------------------------------

func TestStepExecutor_EvaluateConditions(t *testing.T) {
	t.Run("should return nil for nil step", func(t *testing.T) {
		se := NewStepExecutor(nil)
		results, err := se.EvaluateConditions(EvaluateConditionsParams{
			Step: nil,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results != nil {
			t.Errorf("results = %v, want nil", results)
		}
	})

	t.Run("should return nil for step without conditions", func(t *testing.T) {
		se := NewStepExecutor(nil)
		results, err := se.EvaluateConditions(EvaluateConditionsParams{
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeConditional,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if results != nil {
			t.Errorf("results = %v, want nil", results)
		}
	})

	t.Run("should return indices of truthy conditions", func(t *testing.T) {
		se := NewStepExecutor(nil)
		results, err := se.EvaluateConditions(EvaluateConditionsParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeConditional,
				Conditions: []wf.ConditionFunction{
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return true, nil
					},
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return false, nil
					},
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return true, nil
					},
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("len(results) = %d, want 2", len(results))
		}
		if results[0] != 0 {
			t.Errorf("results[0] = %d, want 0", results[0])
		}
		if results[1] != 2 {
			t.Errorf("results[1] = %d, want 2", results[1])
		}
	})

	t.Run("should skip conditions that return errors", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		se := NewStepExecutor(mastra)

		results, err := se.EvaluateConditions(EvaluateConditionsParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeConditional,
				Conditions: []wf.ConditionFunction{
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return false, errors.New("condition error")
					},
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return true, nil
					},
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}
		if results[0] != 1 {
			t.Errorf("results[0] = %d, want 1", results[0])
		}
	})

	t.Run("should return empty slice when no conditions match", func(t *testing.T) {
		se := NewStepExecutor(nil)
		results, err := se.EvaluateConditions(EvaluateConditionsParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeConditional,
				Conditions: []wf.ConditionFunction{
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return false, nil
					},
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						return false, nil
					},
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("len(results) = %d, want 0", len(results))
		}
	})

	t.Run("should pass input data and state to condition function", func(t *testing.T) {
		se := NewStepExecutor(nil)

		var capturedInput any
		var capturedState any
		results, err := se.EvaluateConditions(EvaluateConditionsParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Input:      map[string]any{"value": 42},
			State:      map[string]any{"count": 5},
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeConditional,
				Conditions: []wf.ConditionFunction{
					func(params *wf.ExecuteFunctionParams) (bool, error) {
						capturedInput = params.InputData
						capturedState = params.State
						return true, nil
					},
				},
			},
			StepResults: make(map[string]wf.StepResult),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("len(results) = %d, want 1", len(results))
		}

		inputMap, ok := capturedInput.(map[string]any)
		if !ok {
			t.Fatalf("capturedInput is not map[string]any, got %T", capturedInput)
		}
		if inputMap["value"] != 42 {
			t.Errorf("capturedInput[value] = %v, want 42", inputMap["value"])
		}

		stateMap, ok := capturedState.(map[string]any)
		if !ok {
			t.Fatalf("capturedState is not map[string]any, got %T", capturedState)
		}
		if stateMap["count"] != 5 {
			t.Errorf("capturedState[count] = %v, want 5", stateMap["count"])
		}
	})
}

// ---------------------------------------------------------------------------
// ResolveSleep tests
// ---------------------------------------------------------------------------

func TestStepExecutor_ResolveSleep(t *testing.T) {
	t.Run("should return 0 for nil step", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleep(ResolveSleepParams{Step: nil})
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("should return static duration when set", func(t *testing.T) {
		se := NewStepExecutor(nil)
		dur := int64(5000)
		duration := se.ResolveSleep(ResolveSleepParams{
			Step: &wf.StepFlowEntry{
				Type:     wf.StepFlowEntryTypeSleep,
				Duration: &dur,
			},
		})
		if duration != 5000 {
			t.Errorf("duration = %d, want 5000", duration)
		}
	})

	t.Run("should return 0 when no function and no static duration", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleep(ResolveSleepParams{
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleep,
			},
		})
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("should resolve dynamic duration from function returning int64", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleep(ResolveSleepParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleep,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return int64(3000), nil
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration != 3000 {
			t.Errorf("duration = %d, want 3000", duration)
		}
	})

	t.Run("should resolve dynamic duration from function returning float64", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleep(ResolveSleepParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleep,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return float64(2500), nil
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration != 2500 {
			t.Errorf("duration = %d, want 2500", duration)
		}
	})

	t.Run("should resolve dynamic duration from function returning int", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleep(ResolveSleepParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleep,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return 1000, nil
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration != 1000 {
			t.Errorf("duration = %d, want 1000", duration)
		}
	})

	t.Run("should return 0 when function returns error", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		se := NewStepExecutor(mastra)

		duration := se.ResolveSleep(ResolveSleepParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleep,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return nil, errors.New("sleep error")
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("should prefer static duration over function", func(t *testing.T) {
		se := NewStepExecutor(nil)
		dur := int64(5000)
		duration := se.ResolveSleep(ResolveSleepParams{
			Step: &wf.StepFlowEntry{
				Type:     wf.StepFlowEntryTypeSleep,
				Duration: &dur,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return int64(3000), nil
				},
			},
		})
		if duration != 5000 {
			t.Errorf("duration = %d, want 5000 (static takes priority)", duration)
		}
	})
}

// ---------------------------------------------------------------------------
// ResolveSleepUntil tests
// ---------------------------------------------------------------------------

func TestStepExecutor_ResolveSleepUntil(t *testing.T) {
	t.Run("should return 0 for nil step", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{Step: nil})
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("should compute duration from static date", func(t *testing.T) {
		se := NewStepExecutor(nil)
		futureDate := time.Now().Add(10 * time.Second)
		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
				Date: &futureDate,
			},
		})
		// Should be approximately 10000ms (give or take for execution time)
		if duration < 9000 || duration > 11000 {
			t.Errorf("duration = %d, expected ~10000", duration)
		}
	})

	t.Run("should return negative duration for past date", func(t *testing.T) {
		se := NewStepExecutor(nil)
		pastDate := time.Now().Add(-10 * time.Second)
		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
				Date: &pastDate,
			},
		})
		if duration >= 0 {
			t.Errorf("duration = %d, expected negative value for past date", duration)
		}
	})

	t.Run("should return 0 when no function and no static date", func(t *testing.T) {
		se := NewStepExecutor(nil)
		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
			},
		})
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("should resolve duration from function returning time.Time", func(t *testing.T) {
		se := NewStepExecutor(nil)
		futureDate := time.Now().Add(5 * time.Second)
		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return futureDate, nil
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration < 4000 || duration > 6000 {
			t.Errorf("duration = %d, expected ~5000", duration)
		}
	})

	t.Run("should resolve duration from function returning *time.Time", func(t *testing.T) {
		se := NewStepExecutor(nil)
		futureDate := time.Now().Add(5 * time.Second)
		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return &futureDate, nil
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration < 4000 || duration > 6000 {
			t.Errorf("duration = %d, expected ~5000", duration)
		}
	})

	t.Run("should return 0 when function returns error", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		se := NewStepExecutor(mastra)

		duration := se.ResolveSleepUntil(ResolveSleepUntilParams{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					return nil, errors.New("sleepUntil error")
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})
		if duration != 0 {
			t.Errorf("duration = %d, want 0", duration)
		}
	})

	t.Run("should pass request context to function", func(t *testing.T) {
		se := NewStepExecutor(nil)
		rc := requestcontext.NewRequestContext()
		rc.Set("userId", "user-123")

		var capturedRC *requestcontext.RequestContext
		futureDate := time.Now().Add(1 * time.Second)
		se.ResolveSleepUntil(ResolveSleepUntilParams{
			WorkflowID:     "wf-1",
			RunID:          "run-1",
			RequestContext: rc,
			Step: &wf.StepFlowEntry{
				Type: wf.StepFlowEntryTypeSleepUntil,
				Fn: func(params *wf.ExecuteFunctionParams) (any, error) {
					capturedRC = params.RequestContext
					return futureDate, nil
				},
			},
			StepResults: make(map[string]wf.StepResult),
			State:       map[string]any{},
		})

		if capturedRC == nil {
			t.Error("request context was not passed to function")
		}
	})
}
