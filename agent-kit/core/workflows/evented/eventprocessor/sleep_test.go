// Ported from: packages/core/src/workflows/evented/workflow-event-processor/sleep.test.ts
package eventprocessor

import (
	"sync"
	"testing"
	"time"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// Mock PubSub for eventprocessor tests
// ---------------------------------------------------------------------------

type mockPublishCall struct {
	Topic string
	Event any
}

type mockPubSub struct {
	mu        sync.Mutex
	published []mockPublishCall
}

func newMockPubSub() *mockPubSub {
	return &mockPubSub{}
}

func (m *mockPubSub) Publish(topic string, event any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, mockPublishCall{Topic: topic, Event: event})
	return nil
}

func (m *mockPubSub) Subscribe(topic string, handler func(event any, ack func() error) error) error {
	return nil
}

func (m *mockPubSub) Unsubscribe(topic string, handler func(event any, ack func() error) error) error {
	return nil
}

func (m *mockPubSub) getPublished() []mockPublishCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]mockPublishCall, len(m.published))
	copy(result, m.published)
	return result
}

// ---------------------------------------------------------------------------
// advanceExecutionPath tests
// ---------------------------------------------------------------------------

func TestAdvanceExecutionPath(t *testing.T) {
	t.Run("should return [1] for empty path", func(t *testing.T) {
		result := advanceExecutionPath([]int{})
		if len(result) != 1 || result[0] != 1 {
			t.Errorf("advanceExecutionPath([]) = %v, want [1]", result)
		}
	})

	t.Run("should increment last element of single-element path", func(t *testing.T) {
		result := advanceExecutionPath([]int{0})
		if len(result) != 1 || result[0] != 1 {
			t.Errorf("advanceExecutionPath([0]) = %v, want [1]", result)
		}
	})

	t.Run("should increment last element of multi-element path", func(t *testing.T) {
		result := advanceExecutionPath([]int{0, 2})
		if len(result) != 2 || result[0] != 0 || result[1] != 3 {
			t.Errorf("advanceExecutionPath([0,2]) = %v, want [0,3]", result)
		}
	})

	t.Run("should not modify the original path", func(t *testing.T) {
		original := []int{0, 5}
		result := advanceExecutionPath(original)
		if original[1] != 5 {
			t.Errorf("original path was modified: %v", original)
		}
		if result[1] != 6 {
			t.Errorf("result[1] = %d, want 6", result[1])
		}
	})

	t.Run("should handle path with three elements", func(t *testing.T) {
		result := advanceExecutionPath([]int{1, 2, 3})
		if len(result) != 3 || result[0] != 1 || result[1] != 2 || result[2] != 4 {
			t.Errorf("advanceExecutionPath([1,2,3]) = %v, want [1,2,4]", result)
		}
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowSleep tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowSleep(t *testing.T) {
	t.Run("should return error for nil step", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		err := ProcessWorkflowSleep(&ProcessorArgs{RunID: "run-1"}, pubsub, se, nil)
		if err == nil {
			t.Fatal("expected error for nil step")
		}
	})

	t.Run("should return error for wrong step type", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		step := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeStep}
		err := ProcessWorkflowSleep(&ProcessorArgs{RunID: "run-1"}, pubsub, se, step)
		if err == nil {
			t.Fatal("expected error for wrong step type")
		}
	})

	t.Run("should publish waiting event immediately", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		dur := int64(50) // 50ms for fast test
		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeSleep,
			ID:       "sleep-1",
			Duration: &dur,
		}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			PrevResult:    map[string]any{"status": "success", "output": "prev"},
			ActiveSteps:   map[string]bool{},
		}

		err := ProcessWorkflowSleep(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have at least the waiting event published immediately
		published := pubsub.getPublished()
		if len(published) < 1 {
			t.Fatal("expected at least 1 published event (waiting)")
		}

		// Check the waiting event
		waitingEvt, ok := published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("waiting event is not map[string]any")
		}
		data, ok := waitingEvt["data"].(map[string]any)
		if !ok {
			t.Fatal("waiting event data is not map[string]any")
		}
		if data["type"] != "workflow-step-waiting" {
			t.Errorf("data[type] = %v, want workflow-step-waiting", data["type"])
		}
	})

	t.Run("should advance execution path after sleep", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		dur := int64(10) // very short sleep
		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeSleep,
			ID:       "sleep-1",
			Duration: &dur,
		}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{2},
			PrevResult:    map[string]any{"status": "success", "output": "prev"},
			ActiveSteps:   map[string]bool{},
		}

		err := ProcessWorkflowSleep(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Wait for the goroutine to complete
		time.Sleep(100 * time.Millisecond)

		published := pubsub.getPublished()
		// Should have: waiting event + step result + step finish + advance
		if len(published) < 4 {
			t.Fatalf("expected at least 4 published events, got %d", len(published))
		}

		// Last event should be workflow.step.run with advanced path
		lastEvt, ok := published[len(published)-1].Event.(map[string]any)
		if !ok {
			t.Fatal("last event is not map[string]any")
		}
		if lastEvt["type"] != "workflow.step.run" {
			t.Errorf("last event type = %v, want workflow.step.run", lastEvt["type"])
		}

		data, ok := lastEvt["data"].(map[string]any)
		if !ok {
			t.Fatal("last event data is not map[string]any")
		}
		execPath, ok := data["executionPath"].([]int)
		if !ok {
			t.Fatal("executionPath is not []int")
		}
		if len(execPath) != 1 || execPath[0] != 3 {
			t.Errorf("executionPath = %v, want [3]", execPath)
		}
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowSleepUntil tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowSleepUntil(t *testing.T) {
	t.Run("should return error for nil step", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		err := ProcessWorkflowSleepUntil(&ProcessorArgs{RunID: "run-1"}, pubsub, se, nil)
		if err == nil {
			t.Fatal("expected error for nil step")
		}
	})

	t.Run("should return error for wrong step type", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		step := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeStep}
		err := ProcessWorkflowSleepUntil(&ProcessorArgs{RunID: "run-1"}, pubsub, se, step)
		if err == nil {
			t.Fatal("expected error for wrong step type")
		}
	})

	t.Run("should publish waiting event for sleepUntil", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		// Past date so sleep is effectively 0
		pastDate := time.Now().Add(-1 * time.Second)
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeSleepUntil,
			ID:   "sleep-until-1",
			Date: &pastDate,
		}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			PrevResult:    map[string]any{"status": "success", "output": "prev"},
			ActiveSteps:   map[string]bool{},
		}

		err := ProcessWorkflowSleepUntil(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have at least the waiting event published immediately
		published := pubsub.getPublished()
		if len(published) < 1 {
			t.Fatal("expected at least 1 published event")
		}

		evt, ok := published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("event is not map[string]any")
		}
		data, ok := evt["data"].(map[string]any)
		if !ok {
			t.Fatal("event data is not map[string]any")
		}
		if data["type"] != "workflow-step-waiting" {
			t.Errorf("data[type] = %v, want workflow-step-waiting", data["type"])
		}
	})

	t.Run("should advance after sleepUntil with past date", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		pastDate := time.Now().Add(-1 * time.Second)
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeSleepUntil,
			ID:   "sleep-until-1",
			Date: &pastDate,
		}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{1},
			PrevResult:    map[string]any{"status": "success", "output": "prev"},
			ActiveSteps:   map[string]bool{},
		}

		err := ProcessWorkflowSleepUntil(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Wait for goroutine
		time.Sleep(100 * time.Millisecond)

		published := pubsub.getPublished()
		if len(published) < 4 {
			t.Fatalf("expected at least 4 published events, got %d", len(published))
		}

		lastEvt, ok := published[len(published)-1].Event.(map[string]any)
		if !ok {
			t.Fatal("last event is not map[string]any")
		}
		data, ok := lastEvt["data"].(map[string]any)
		if !ok {
			t.Fatal("event data is not map[string]any")
		}
		execPath, ok := data["executionPath"].([]int)
		if !ok {
			t.Fatal("executionPath is not []int")
		}
		if len(execPath) != 1 || execPath[0] != 2 {
			t.Errorf("executionPath = %v, want [2]", execPath)
		}
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowWaitForEvent tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowWaitForEvent(t *testing.T) {
	t.Run("should return nil when currentState is nil", func(t *testing.T) {
		pubsub := newMockPubSub()
		err := ProcessWorkflowWaitForEvent(
			&ProcessorArgs{RunID: "run-1"},
			pubsub,
			"my-event",
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pubsub.getPublished()) != 0 {
			t.Error("should not publish when currentState is nil")
		}
	})

	t.Run("should return nil when event is not in waitingPaths", func(t *testing.T) {
		pubsub := newMockPubSub()
		state := &evented.WorkflowRunState{
			WaitingPaths: map[string][]int{
				"other-event": {0},
			},
		}
		err := ProcessWorkflowWaitForEvent(
			&ProcessorArgs{
				RunID:      "run-1",
				WorkflowID: "wf-1",
			},
			pubsub,
			"my-event",
			state,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pubsub.getPublished()) != 0 {
			t.Error("should not publish when event is not in waitingPaths")
		}
	})

	t.Run("should publish step.run event when event matches", func(t *testing.T) {
		pubsub := newMockPubSub()
		state := &evented.WorkflowRunState{
			WaitingPaths: map[string][]int{
				"my-event": {0},
			},
			Context: map[string]any{
				"input": map[string]any{
					"payload": "test-data",
				},
			},
		}
		args := &ProcessorArgs{
			RunID:      "run-1",
			WorkflowID: "wf-1",
			Workflow: &mockWorkflow{
				stepGraph: []wf.StepFlowEntry{
					{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
				},
			},
		}

		err := ProcessWorkflowWaitForEvent(args, pubsub, "my-event", state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		if len(published) != 1 {
			t.Fatalf("expected 1 published event, got %d", len(published))
		}
		if published[0].Topic != "workflows" {
			t.Errorf("topic = %q, want %q", published[0].Topic, "workflows")
		}
		evt, ok := published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("event is not map[string]any")
		}
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run", evt["type"])
		}
	})
}
