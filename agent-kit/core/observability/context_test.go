// Ported from: packages/core/src/observability/context.test.ts
package observability

import (
	"testing"

	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Mock types for context tests
// ============================================================================

// mockMastraObj is a mock Mastra-like object that has all required getter methods.
type mockMastraObj struct {
	getAgentCalls       []string
	getAgentByIdCalls   []string
	getWorkflowCalls    []string
	getWorkflowByIdCalls []string
	otherMethodCalls    int
	agentToReturn       any
	workflowToReturn    any
}

func newMockMastraObj() *mockMastraObj {
	return &mockMastraObj{}
}

func (m *mockMastraObj) GetAgent(name string) any {
	m.getAgentCalls = append(m.getAgentCalls, name)
	return m.agentToReturn
}

func (m *mockMastraObj) GetAgentById(id string) any {
	m.getAgentByIdCalls = append(m.getAgentByIdCalls, id)
	return m.agentToReturn
}

func (m *mockMastraObj) GetWorkflow(name string) any {
	m.getWorkflowCalls = append(m.getWorkflowCalls, name)
	return m.workflowToReturn
}

func (m *mockMastraObj) GetWorkflowById(id string) any {
	m.getWorkflowByIdCalls = append(m.getWorkflowByIdCalls, id)
	return m.workflowToReturn
}

func (m *mockMastraObj) OtherMethod() string {
	m.otherMethodCalls++
	return "other-result"
}

// mockAgentObj is a mock agent.
type mockAgentObj struct {
	generateCalls int
	streamCalls   int
}

func (a *mockAgentObj) Generate(input string) string {
	a.generateCalls++
	return "generated"
}

func (a *mockAgentObj) Stream(input string) string {
	a.streamCalls++
	return "streamed"
}

func (a *mockAgentObj) OtherMethod() string {
	return "agent-other-result"
}

// mockWorkflowObj is a mock workflow.
type mockWorkflowObj struct {
	executeCalls   int
	createRunCalls int
}

func (w *mockWorkflowObj) Execute(input any) string {
	w.executeCalls++
	return "executed"
}

func (w *mockWorkflowObj) CreateRun() any {
	w.createRunCalls++
	return &mockRunObj{}
}

func (w *mockWorkflowObj) OtherMethod() string {
	return "workflow-other-result"
}

// mockRunObj is a mock workflow run.
type mockRunObj struct {
	startCalls int
}

func (r *mockRunObj) Start(opts any) {
	r.startCalls++
}

func (r *mockRunObj) OtherMethod() string {
	return "run-other-result"
}

// validMockSpan creates a span that is valid (not NoOp) for testing wrapMastra.
type validMockSpan struct {
	mockSpanForFactory
}

func newValidMockSpan() *validMockSpan {
	return &validMockSpan{
		mockSpanForFactory: mockSpanForFactory{
			spanID:   "valid-span",
			instance: &mockObsInstance{},
		},
	}
}

// noOpMockSpan creates a span that is invalid (NoOp) for testing wrapMastra.
type noOpMockSpan struct {
	mockSpanForFactory
}

func newNoOpMockSpan() *noOpMockSpan {
	return &noOpMockSpan{
		mockSpanForFactory: mockSpanForFactory{
			spanID:   "noop-span",
			instance: nil,
		},
	}
}

// Override IsValid to return false, simulating a NoOp span.
func (s *noOpMockSpan) IsValid() bool { return false }

// Ensure noOpMockSpan satisfies Span.
var _ obstypes.Span = (*noOpMockSpan)(nil)

// ============================================================================
// wrapMastra
// ============================================================================

func TestWrapMastra(t *testing.T) {
	t.Run("should return wrapped Mastra with tracing context", func(t *testing.T) {
		mockMastra := newMockMastraObj()
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)

		// Should not be the same object
		if wrapped == mockMastra {
			t.Error("expected wrapped to be different from original")
		}

		// Should be a *WrappedMastra
		wm, ok := wrapped.(*WrappedMastra)
		if !ok {
			t.Fatal("expected wrapped to be *WrappedMastra")
		}
		if wm.Inner() != mockMastra {
			t.Error("expected Inner() to return original mastra")
		}
	})

	t.Run("should return original Mastra when no current span", func(t *testing.T) {
		mockMastra := newMockMastraObj()
		emptyContext := obstypes.TracingContext{CurrentSpan: nil}

		wrapped := WrapMastra(mockMastra, emptyContext)

		if wrapped != mockMastra {
			t.Error("expected wrapped to be the original mastra when no span")
		}
	})

	t.Run("should return original Mastra when using NoOp span", func(t *testing.T) {
		mockMastra := newMockMastraObj()
		noOpSpan := newNoOpMockSpan()
		noOpCtx := obstypes.TracingContext{CurrentSpan: noOpSpan}

		wrapped := WrapMastra(mockMastra, noOpCtx)

		if wrapped != mockMastra {
			t.Error("expected wrapped to be the original mastra when NoOp span")
		}
	})

	t.Run("should wrap agent getters to return tracing-aware agents", func(t *testing.T) {
		mockAgent := &mockAgentObj{}
		mockMastra := newMockMastraObj()
		mockMastra.agentToReturn = mockAgent
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)
		wm := wrapped.(*WrappedMastra)

		agent := wm.CallAgentGetter("GetAgent", "test-agent")

		if len(mockMastra.getAgentCalls) != 1 || mockMastra.getAgentCalls[0] != "test-agent" {
			t.Errorf("expected GetAgent to be called with 'test-agent', got calls: %v", mockMastra.getAgentCalls)
		}

		// Agent should be wrapped (different instance)
		if agent == mockAgent {
			t.Error("expected agent to be wrapped (different from original)")
		}

		_, ok := agent.(*WrappedAgent)
		if !ok {
			t.Error("expected agent to be *WrappedAgent")
		}
	})

	t.Run("should wrap workflow getters to return tracing-aware workflows", func(t *testing.T) {
		mockWorkflow := &mockWorkflowObj{}
		mockMastra := newMockMastraObj()
		mockMastra.workflowToReturn = mockWorkflow
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)
		wm := wrapped.(*WrappedMastra)

		workflow := wm.CallWorkflowGetter("GetWorkflow", "test-workflow")

		if len(mockMastra.getWorkflowCalls) != 1 || mockMastra.getWorkflowCalls[0] != "test-workflow" {
			t.Errorf("expected GetWorkflow to be called with 'test-workflow', got calls: %v", mockMastra.getWorkflowCalls)
		}

		// Workflow should be wrapped (different instance)
		if workflow == mockWorkflow {
			t.Error("expected workflow to be wrapped (different from original)")
		}

		_, ok := workflow.(*WrappedWorkflow)
		if !ok {
			t.Error("expected workflow to be *WrappedWorkflow")
		}
	})

	t.Run("should handle proxy creation errors gracefully", func(t *testing.T) {
		// In Go: test that WrapMastra returns original on nil span (invalid context).
		mockMastra := newMockMastraObj()
		invalidContext := obstypes.TracingContext{CurrentSpan: nil}

		wrapped := WrapMastra(mockMastra, invalidContext)

		if wrapped != mockMastra {
			t.Error("expected wrapped to be original on nil span")
		}
	})
}

// ============================================================================
// Workflow run creation and tracing
// ============================================================================

func TestWorkflowRunWrapping(t *testing.T) {
	t.Run("should wrap createRun to return run proxy", func(t *testing.T) {
		mockWorkflow := &mockWorkflowObj{}
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		// Wrap the workflow directly.
		wrappedWorkflow := wrapWorkflow(mockWorkflow, tc)

		ww, ok := wrappedWorkflow.(*WrappedWorkflow)
		if !ok {
			t.Fatal("expected WrappedWorkflow")
		}

		_ = ww // The wrapper itself proves wrapping occurred.
		if ww.Inner() == nil {
			t.Error("expected Inner() to return the original workflow")
		}
	})

	t.Run("should produce observability context from wrapper", func(t *testing.T) {
		mockWorkflow := &mockWorkflowObj{}
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrappedWorkflow := wrapWorkflow(mockWorkflow, tc)
		ww := wrappedWorkflow.(*WrappedWorkflow)

		obsCtx := ww.ObservabilityContext()

		// The observability context should contain our tracing context.
		if obsCtx.Tracing.CurrentSpan != span {
			t.Error("expected ObservabilityContext to contain our span")
		}
	})

	t.Run("should wrap run with tracing context", func(t *testing.T) {
		mockRun := &mockRunObj{}
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrappedRun := WrapRun(mockRun, tc)

		wr, ok := wrappedRun.(*WrappedRun)
		if !ok {
			t.Fatal("expected WrappedRun")
		}

		if wr.Inner() != mockRun {
			t.Error("expected Inner() to return the original run")
		}

		obsCtx := wr.ObservabilityContext()
		if obsCtx.Tracing.CurrentSpan != span {
			t.Error("expected ObservabilityContext to contain our span")
		}
	})

	t.Run("should not wrap run with NoOp span", func(t *testing.T) {
		mockRun := &mockRunObj{}
		noOpSpan := newNoOpMockSpan()
		noOpCtx := obstypes.TracingContext{CurrentSpan: noOpSpan}

		wrappedRun := WrapRun(mockRun, noOpCtx)

		if wrappedRun != mockRun {
			t.Error("expected run to be unwrapped when using NoOp span")
		}
	})
}

// ============================================================================
// Integration scenarios
// ============================================================================

func TestIntegrationScenarios(t *testing.T) {
	t.Run("should work in nested workflow step scenario", func(t *testing.T) {
		mockAgent := &mockAgentObj{}
		mockMastra := newMockMastraObj()
		mockMastra.agentToReturn = mockAgent
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)
		wm := wrapped.(*WrappedMastra)

		agent := wm.CallAgentGetter("GetAgent", "test-agent")

		// Agent should be wrapped and ready to inject context.
		if agent == mockAgent {
			t.Error("expected agent to be wrapped")
		}

		wa, ok := agent.(*WrappedAgent)
		if !ok {
			t.Fatal("expected *WrappedAgent")
		}

		// Verify the wrapped agent carries our tracing context.
		obsCtx := wa.ObservabilityContext()
		if obsCtx.Tracing.CurrentSpan != span {
			t.Error("expected agent's ObservabilityContext to contain our span")
		}
	})

	t.Run("should work with workflow calling another workflow", func(t *testing.T) {
		mockWorkflow := &mockWorkflowObj{}
		mockMastra := newMockMastraObj()
		mockMastra.workflowToReturn = mockWorkflow
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)
		wm := wrapped.(*WrappedMastra)

		workflow := wm.CallWorkflowGetter("GetWorkflow", "child-workflow")

		if workflow == mockWorkflow {
			t.Error("expected workflow to be wrapped")
		}

		ww, ok := workflow.(*WrappedWorkflow)
		if !ok {
			t.Fatal("expected *WrappedWorkflow")
		}

		obsCtx := ww.ObservabilityContext()
		if obsCtx.Tracing.CurrentSpan != span {
			t.Error("expected workflow's ObservabilityContext to contain our span")
		}
	})

	t.Run("should preserve type safety (all getters work)", func(t *testing.T) {
		mockAgent := &mockAgentObj{}
		mockWorkflow := &mockWorkflowObj{}
		mockMastra := newMockMastraObj()
		mockMastra.agentToReturn = mockAgent
		mockMastra.workflowToReturn = mockWorkflow
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)
		wm := wrapped.(*WrappedMastra)

		agent := wm.CallAgentGetter("GetAgent", "test")
		agentById := wm.CallAgentGetter("GetAgentById", "test-id")
		workflow := wm.CallWorkflowGetter("GetWorkflow", "test")
		workflowById := wm.CallWorkflowGetter("GetWorkflowById", "test-id")

		if agent == nil {
			t.Error("expected agent to be non-nil")
		}
		if agentById == nil {
			t.Error("expected agentById to be non-nil")
		}
		if workflow == nil {
			t.Error("expected workflow to be non-nil")
		}
		if workflowById == nil {
			t.Error("expected workflowById to be non-nil")
		}
	})

	t.Run("should handle mixed wrapped and unwrapped usage", func(t *testing.T) {
		mockAgent := &mockAgentObj{}
		mockMastra := newMockMastraObj()
		mockMastra.agentToReturn = mockAgent
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrappedMastra := WrapMastra(mockMastra, tc)
		unwrappedMastra := WrapMastra(mockMastra, obstypes.TracingContext{CurrentSpan: nil})

		// Wrapped mastra returns a wrapper.
		wm := wrappedMastra.(*WrappedMastra)
		wrappedAgent := wm.CallAgentGetter("GetAgent", "test")

		wa, ok := wrappedAgent.(*WrappedAgent)
		if !ok {
			t.Fatal("expected wrapped agent to be *WrappedAgent")
		}
		if wa.ObservabilityContext().Tracing.CurrentSpan != span {
			t.Error("expected wrapped agent's context to contain our span")
		}

		// Unwrapped mastra IS the original.
		if unwrappedMastra != mockMastra {
			t.Error("expected unwrapped mastra to be the original")
		}
	})
}

// ============================================================================
// Error handling and edge cases
// ============================================================================

func TestErrorHandlingEdgeCases(t *testing.T) {
	t.Run("should handle undefined tracingContext gracefully", func(t *testing.T) {
		mockMastra := newMockMastraObj()

		wrapped := WrapMastra(mockMastra, obstypes.TracingContext{CurrentSpan: nil})

		if wrapped != mockMastra {
			t.Error("expected wrapped to be original when no span")
		}
	})

	t.Run("should handle NoOp spans correctly", func(t *testing.T) {
		mockMastra := newMockMastraObj()

		// Test with various NoOp-like spans.
		noOpSpan1 := newNoOpMockSpan()
		wrapped1 := WrapMastra(mockMastra, obstypes.TracingContext{CurrentSpan: noOpSpan1})
		if wrapped1 != mockMastra {
			t.Error("expected wrapped1 to be original for NoOp span")
		}

		// Nil span.
		wrapped2 := WrapMastra(mockMastra, obstypes.TracingContext{CurrentSpan: nil})
		if wrapped2 != mockMastra {
			t.Error("expected wrapped2 to be original for nil span")
		}
	})

	t.Run("should handle CallAgentGetter with nonexistent method gracefully", func(t *testing.T) {
		mockMastra := newMockMastraObj()
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)
		wm := wrapped.(*WrappedMastra)

		// Calling a method that doesn't exist should return nil.
		result := wm.CallAgentGetter("NonExistentMethod", "test")
		if result != nil {
			t.Errorf("expected nil for nonexistent method, got %v", result)
		}
	})
}

// ============================================================================
// Mastra interface compatibility
// ============================================================================

func TestMastraInterfaceCompatibility(t *testing.T) {
	t.Run("should verify that IsMastra works with a full mock", func(t *testing.T) {
		mockMastra := newMockMastraObj()

		if !IsMastra(mockMastra) {
			t.Error("expected IsMastra to return true for a mock with all getters")
		}
	})

	t.Run("should detect if wrapMastra would skip wrapping due to missing methods", func(t *testing.T) {
		// Test object with no agent or workflow getters.
		type primitivesMastra struct{}
		pm := &primitivesMastra{}
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(pm, tc)

		// Should return the original object since it has no methods to wrap.
		if wrapped != pm {
			t.Error("expected wrapped to be original for object without expected methods")
		}
	})

	t.Run("should wrap objects that have all agent and workflow getters", func(t *testing.T) {
		mockMastra := newMockMastraObj()
		span := newValidMockSpan()
		tc := obstypes.TracingContext{CurrentSpan: span}

		wrapped := WrapMastra(mockMastra, tc)

		// Should return a *WrappedMastra.
		if wrapped == mockMastra {
			t.Error("expected wrapped to be different from original")
		}
		_, ok := wrapped.(*WrappedMastra)
		if !ok {
			t.Error("expected wrapped to be *WrappedMastra")
		}
	})

	t.Run("IsMastra returns false for nil", func(t *testing.T) {
		if IsMastra(nil) {
			t.Error("expected IsMastra(nil) to return false")
		}
	})

	t.Run("IsMastra returns false for object missing some getters", func(t *testing.T) {
		// Object with only some methods.
		type partialMastra struct{}
		pm := &partialMastra{}

		if IsMastra(pm) {
			t.Error("expected IsMastra to return false for object with missing getters")
		}
	})
}

// ============================================================================
// IsTracedAgentMethod / IsTracedWorkflowMethod
// ============================================================================

func TestIsTracedMethods(t *testing.T) {
	t.Run("IsTracedAgentMethod identifies correct methods", func(t *testing.T) {
		for _, method := range []string{"Generate", "Stream", "GenerateLegacy", "StreamLegacy"} {
			if !IsTracedAgentMethod(method) {
				t.Errorf("expected IsTracedAgentMethod(%q) to be true", method)
			}
		}
		if IsTracedAgentMethod("OtherMethod") {
			t.Error("expected IsTracedAgentMethod('OtherMethod') to be false")
		}
	})

	t.Run("IsTracedWorkflowMethod identifies correct methods", func(t *testing.T) {
		for _, method := range []string{"Execute", "CreateRun"} {
			if !IsTracedWorkflowMethod(method) {
				t.Errorf("expected IsTracedWorkflowMethod(%q) to be true", method)
			}
		}
		if IsTracedWorkflowMethod("OtherMethod") {
			t.Error("expected IsTracedWorkflowMethod('OtherMethod') to be false")
		}
	})
}
