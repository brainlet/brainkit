// Ported from: packages/core/src/observability/context.ts
package observability

import (
	"log"
	"reflect"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Constants
// ============================================================================

// agentGetters lists methods that return agents from a Mastra-like object.
var agentGetters = []string{"GetAgent", "GetAgentById"}

// AgentMethodsToWrap lists agent methods that need tracing context injection.
// Exported so callers can check whether a method name requires wrapping.
var AgentMethodsToWrap = []string{"Generate", "Stream", "GenerateLegacy", "StreamLegacy"}

// workflowGetters lists methods that return workflows from a Mastra-like object.
var workflowGetters = []string{"GetWorkflow", "GetWorkflowById"}

// WorkflowMethodsToWrap lists workflow methods that need tracing context injection.
// Exported so callers can check whether a method name requires wrapping.
var WorkflowMethodsToWrap = []string{"Execute", "CreateRun"}

// ============================================================================
// Stub Types
// ============================================================================

// Agent is a stub for the Agent type from the agent package.
// Cannot import agent: agent imports observability (via observability/types), creating
// a circular dependency. The observability package wraps agents using reflection (see
// wrapAgent, WrappedAgent), never calling methods on the Agent type directly. These
// methods represent the minimum useful contract for identification when agents are
// passed through the observability wrapping layer.
type Agent interface {
	// GetID returns the agent's unique identifier.
	GetID() string
	// GetName returns the agent's display name.
	GetName() string
}

// Workflow is a stub for the Workflow type from the workflows package.
// Cannot import workflows: would risk circular dependency through agent → observability.
// The observability package wraps workflows using reflection (see wrapWorkflow,
// WrappedWorkflow), never calling methods on the Workflow type directly. These methods
// represent the minimum useful contract for identification when workflows are passed
// through the observability wrapping layer.
type Workflow interface {
	// GetID returns the workflow's unique identifier.
	GetID() string
	// GetName returns the workflow's display name.
	GetName() string
}

// MastraPrimitives is a stub for the MastraPrimitives type from the action package.
// Cannot import action: observability is imported by action (indirectly via types),
// creating a circular dependency risk. MastraPrimitives is a dependency injection
// container holding references to core Mastra services (logger, storage, agents, etc.).
// The observability package never directly calls methods on MastraPrimitives; it is
// only referenced as a type parameter in the wrapping logic. These methods represent
// the minimum useful contract for accessing framework services.
type MastraPrimitives interface {
	// GetLogger returns the configured logger instance from the primitives container.
	GetLogger() ObsLogger
}

// Mastra represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports observability (for tracing setup), so observability cannot import core.
// core.Mastra struct satisfies this interface.
// The observability context wrapper uses this to access agents and workflows
// for injecting tracing context into their methods.
type Mastra interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
}

// ObsLogger is a type alias to logger.IMastraLogger so that core.Mastra
// satisfies the observability.Mastra interface at compile time.
//
// Ported from: packages/core/src/observability — uses mastra.getLogger()
type ObsLogger = logger.IMastraLogger

// ============================================================================
// NoOp Span Detection
// ============================================================================

// isNoOpSpan detects NoOp spans to avoid unnecessary wrapping.
func isNoOpSpan(span obstypes.Span) bool {
	if span == nil {
		return true
	}
	// Check if the span reports itself as invalid (NoOp spans return false from IsValid).
	if !span.IsValid() {
		return true
	}
	// Check by type name as a fallback.
	t := reflect.TypeOf(span)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name() == "NoOpSpan"
}

// ============================================================================
// Mastra Detection
// ============================================================================

// IsMastra checks if the given value implements the methods expected of a Mastra instance
// (for the purposes of wrapping it for tracing).
func IsMastra(mastra any) bool {
	if mastra == nil {
		return false
	}
	v := reflect.ValueOf(mastra)
	// Check agent getters.
	for _, method := range agentGetters {
		m := v.MethodByName(method)
		if !m.IsValid() {
			return false
		}
	}
	// Check workflow getters.
	for _, method := range workflowGetters {
		m := v.MethodByName(method)
		if !m.IsValid() {
			return false
		}
	}
	return true
}

// ============================================================================
// Wrapping
// ============================================================================

// WrappedMastra wraps a Mastra-like object to inject tracing context into
// agent and workflow method calls. In Go, we cannot use JS-style Proxy objects,
// so instead we provide a wrapper that holds a reference to the underlying
// mastra and the tracing context.
//
// Callers use CallAgentGetter/CallWorkflowGetter on this wrapper, which return
// tracing-aware wrappers around the retrieved agents/workflows.
type WrappedMastra struct {
	inner          any
	tracingContext obstypes.TracingContext
}

// WrapMastra creates a tracing-aware wrapper around a Mastra-like object.
// If no current span exists or the span is a NoOp, the original object is returned as-is.
//
// Unlike the TypeScript version which uses ES6 Proxy, Go uses an explicit wrapper struct.
// Callers should type-assert to *WrappedMastra to access agent/workflow getters.
func WrapMastra(mastra any, tracingContext obstypes.TracingContext) any {
	// Don't wrap if no current span or if using NoOp span.
	if tracingContext.CurrentSpan == nil || isNoOpSpan(tracingContext.CurrentSpan) {
		return mastra
	}

	// Check if this object has the methods we want to wrap.
	if !IsMastra(mastra) {
		return mastra
	}

	return &WrappedMastra{
		inner:          mastra,
		tracingContext: tracingContext,
	}
}

// Inner returns the underlying mastra object.
func (w *WrappedMastra) Inner() any {
	return w.inner
}

// TracingCtx returns the tracing context associated with this wrapper.
func (w *WrappedMastra) TracingCtx() obstypes.TracingContext {
	return w.tracingContext
}

// CallAgentGetter calls an agent getter method on the underlying mastra and wraps
// the returned agent with tracing context. The method name should be one of
// "GetAgent" or "GetAgentById".
func (w *WrappedMastra) CallAgentGetter(methodName string, args ...any) any {
	v := reflect.ValueOf(w.inner)
	m := v.MethodByName(methodName)
	if !m.IsValid() {
		log.Printf("Tracing: method %s not found on mastra, returning nil", methodName)
		return nil
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	results := m.Call(in)
	if len(results) == 0 {
		return nil
	}

	agent := results[0].Interface()
	return wrapAgent(agent, w.tracingContext)
}

// CallWorkflowGetter calls a workflow getter method on the underlying mastra and wraps
// the returned workflow with tracing context. The method name should be one of
// "GetWorkflow" or "GetWorkflowById".
func (w *WrappedMastra) CallWorkflowGetter(methodName string, args ...any) any {
	v := reflect.ValueOf(w.inner)
	m := v.MethodByName(methodName)
	if !m.IsValid() {
		log.Printf("Tracing: method %s not found on mastra, returning nil", methodName)
		return nil
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	results := m.Call(in)
	if len(results) == 0 {
		return nil
	}

	workflow := results[0].Interface()
	return wrapWorkflow(workflow, w.tracingContext)
}

// ============================================================================
// Agent Wrapping
// ============================================================================

// WrappedAgent wraps an agent to inject tracing context into generation method calls.
type WrappedAgent struct {
	inner          any
	tracingContext obstypes.TracingContext
}

// wrapAgent creates a tracing-aware wrapper around an agent.
func wrapAgent(agent any, tracingContext obstypes.TracingContext) any {
	if tracingContext.CurrentSpan == nil || isNoOpSpan(tracingContext.CurrentSpan) {
		return agent
	}

	return &WrappedAgent{
		inner:          agent,
		tracingContext: tracingContext,
	}
}

// Inner returns the underlying agent object.
func (w *WrappedAgent) Inner() any {
	return w.inner
}

// TracingCtx returns the tracing context associated with this wrapper.
func (w *WrappedAgent) TracingCtx() obstypes.TracingContext {
	return w.tracingContext
}

// ObservabilityContext returns a full ObservabilityContext derived from the
// agent wrapper's tracing context. Callers should merge this into their
// method options to propagate tracing, logging, and metrics.
func (w *WrappedAgent) ObservabilityContext() obstypes.ObservabilityContext {
	return CreateObservabilityContext(&w.tracingContext)
}

// IsTracedAgentMethod reports whether the given method name is one that should
// have tracing context injected (Generate, Stream, GenerateLegacy, StreamLegacy).
func IsTracedAgentMethod(methodName string) bool {
	for _, m := range AgentMethodsToWrap {
		if m == methodName {
			return true
		}
	}
	return false
}

// ============================================================================
// Workflow Wrapping
// ============================================================================

// WrappedWorkflow wraps a workflow to inject tracing context into execution method calls.
type WrappedWorkflow struct {
	inner          any
	tracingContext obstypes.TracingContext
}

// wrapWorkflow creates a tracing-aware wrapper around a workflow.
func wrapWorkflow(workflow any, tracingContext obstypes.TracingContext) any {
	if tracingContext.CurrentSpan == nil || isNoOpSpan(tracingContext.CurrentSpan) {
		return workflow
	}

	return &WrappedWorkflow{
		inner:          workflow,
		tracingContext: tracingContext,
	}
}

// Inner returns the underlying workflow object.
func (w *WrappedWorkflow) Inner() any {
	return w.inner
}

// TracingCtx returns the tracing context associated with this wrapper.
func (w *WrappedWorkflow) TracingCtx() obstypes.TracingContext {
	return w.tracingContext
}

// ObservabilityContext returns a full ObservabilityContext derived from the
// workflow wrapper's tracing context. Callers should merge this into their
// method options to propagate tracing, logging, and metrics.
func (w *WrappedWorkflow) ObservabilityContext() obstypes.ObservabilityContext {
	return CreateObservabilityContext(&w.tracingContext)
}

// IsTracedWorkflowMethod reports whether the given method name is one that
// should have tracing context injected (Execute, CreateRun).
func IsTracedWorkflowMethod(methodName string) bool {
	for _, m := range WorkflowMethodsToWrap {
		if m == methodName {
			return true
		}
	}
	return false
}

// ============================================================================
// Run Wrapping
// ============================================================================

// WrappedRun wraps a workflow run to inject tracing context into start method calls.
type WrappedRun struct {
	inner          any
	tracingContext obstypes.TracingContext
}

// WrapRun creates a tracing-aware wrapper around a workflow run.
// If no current span exists or the span is a NoOp, the original run is returned as-is.
func WrapRun(run any, tracingContext obstypes.TracingContext) any {
	return wrapRun(run, tracingContext)
}

// wrapRun creates a tracing-aware wrapper around a workflow run.
func wrapRun(run any, tracingContext obstypes.TracingContext) any {
	if tracingContext.CurrentSpan == nil || isNoOpSpan(tracingContext.CurrentSpan) {
		return run
	}

	return &WrappedRun{
		inner:          run,
		tracingContext: tracingContext,
	}
}

// Inner returns the underlying run object.
func (w *WrappedRun) Inner() any {
	return w.inner
}

// TracingCtx returns the tracing context associated with this wrapper.
func (w *WrappedRun) TracingCtx() obstypes.TracingContext {
	return w.tracingContext
}

// ObservabilityContext returns a full ObservabilityContext derived from the
// run wrapper's tracing context. Callers should merge this into start options
// to propagate tracing, logging, and metrics.
func (w *WrappedRun) ObservabilityContext() obstypes.ObservabilityContext {
	return CreateObservabilityContext(&w.tracingContext)
}
