// Ported from: packages/core/src/workflows/execution-engine.ts
package workflows

import (
	"context"
	"fmt"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/events"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Execution Graph
// ---------------------------------------------------------------------------

// ExecutionGraph represents an execution graph for a workflow.
// TS equivalent: export interface ExecutionGraph<TEngineType = any>
type ExecutionGraph struct {
	ID    string          `json:"id"`
	Steps []StepFlowEntry `json:"steps"`
}

// ---------------------------------------------------------------------------
// Execution Engine Options
// ---------------------------------------------------------------------------

// ExecutionEngineOptions holds options for the execution engine.
// TS equivalent: export interface ExecutionEngineOptions
type ExecutionEngineOptions struct {
	TracingPolicy         *obstypes.TracingPolicy
	ValidateInputs        bool
	ShouldPersistSnapshot func(params ShouldPersistSnapshotParams) bool
	OnFinish              func(result WorkflowFinishCallbackResult) error
	OnError               func(errorInfo WorkflowErrorCallbackInfo) error
}

// ---------------------------------------------------------------------------
// ExecutionEngine Interface
// ---------------------------------------------------------------------------

// ExecutionEngine is the abstract interface for building and executing workflow graphs.
// Providers implement this interface to provide their own execution logic.
// TS equivalent: export abstract class ExecutionEngine extends MastraBase
type ExecutionEngine interface {
	// RegisterMastra registers the mastra instance with this engine.
	RegisterMastra(mastra Mastra)

	// GetLogger returns the engine's logger.
	GetLogger() logger.IMastraLogger

	// SetLogger sets the engine's logger.
	SetLogger(log logger.IMastraLogger)

	// GetMastra returns the mastra instance, if registered.
	GetMastra() Mastra

	// GetOptions returns the engine options.
	GetOptions() *ExecutionEngineOptions

	// InvokeLifecycleCallbacks invokes the onFinish and onError lifecycle callbacks if defined.
	// Errors in callbacks are caught and logged, not propagated.
	InvokeLifecycleCallbacks(result LifecycleCallbackParams) error

	// Execute executes a workflow run with the provided execution graph and input.
	Execute(params ExecuteParams) (any, error)
}

// LifecycleCallbackParams holds the parameters for invoking lifecycle callbacks.
// TS equivalent: the inline type in invokeLifecycleCallbacks
type LifecycleCallbackParams struct {
	Status            WorkflowRunStatus
	Result            any
	Error             any
	Steps             map[string]StepResult
	Tripwire          any
	RunID             string
	WorkflowID        string
	ResourceID        string
	Input             any
	RequestContext    *requestcontext.RequestContext
	State             map[string]any
	StepExecutionPath []string
}

// ExecuteParams holds the parameters for ExecutionEngine.Execute().
// TS equivalent: the params inline type in abstract execute<TState, TInput, TOutput>(...)
type ExecuteParams struct {
	WorkflowID        string
	RunID             string
	ResourceID        string
	DisableScorers    bool
	Graph             ExecutionGraph
	SerializedStepGraph []SerializedStepFlowEntry
	Input             any
	InitialState      any
	TimeTravel        *TimeTravelExecutionParams
	Restart           *RestartExecutionParams
	Resume            *ResumeExecuteParams
	PubSub            events.PubSub
	RequestContext    *requestcontext.RequestContext
	WorkflowSpan      obstypes.AnySpan
	RetryConfig       *RetryConfig
	AbortCtx          context.Context
	AbortCancel       context.CancelFunc
	OutputWriter      OutputWriter
	Format            string // "legacy", "vnext", or ""
	OutputOptions     *OutputOptions
	PerStep           bool
	TracingIDs        *TracingIDs
}

// ResumeExecuteParams holds resume data for workflow execution.
// TS equivalent: the resume inline type in execute params
type ResumeExecuteParams struct {
	Steps             []string
	StepResults       map[string]StepResult
	ResumePayload     any
	ResumePath        []int
	StepExecutionPath []string
	Label             string
	ForEachIndex      *int
}

// ---------------------------------------------------------------------------
// Base Execution Engine
// ---------------------------------------------------------------------------

// BaseExecutionEngine provides a base implementation of ExecutionEngine
// with shared logic for lifecycle callbacks and mastra registration.
// TS equivalent: the non-abstract parts of ExecutionEngine
type BaseExecutionEngine struct {
	mastra  Mastra
	log     logger.IMastraLogger
	Options *ExecutionEngineOptions
}

// NewBaseExecutionEngine creates a new BaseExecutionEngine.
func NewBaseExecutionEngine(mastra Mastra, options *ExecutionEngineOptions) BaseExecutionEngine {
	var log logger.IMastraLogger
	if mastra != nil {
		log = mastra.GetLogger()
	}
	return BaseExecutionEngine{
		mastra:  mastra,
		log:     log,
		Options: options,
	}
}

// RegisterMastra registers the mastra instance with this engine.
// TS equivalent: __registerMastra(mastra: Mastra)
func (e *BaseExecutionEngine) RegisterMastra(mastra Mastra) {
	e.mastra = mastra
	if mastra != nil {
		log := mastra.GetLogger()
		if log != nil {
			e.log = log
		}
	}
}

// GetLogger returns the engine's logger.
func (e *BaseExecutionEngine) GetLogger() logger.IMastraLogger {
	return e.log
}

// SetLogger sets the engine's logger.
func (e *BaseExecutionEngine) SetLogger(log logger.IMastraLogger) {
	e.log = log
}

// GetMastra returns the mastra instance.
func (e *BaseExecutionEngine) GetMastra() Mastra {
	return e.mastra
}

// GetOptions returns the engine options.
func (e *BaseExecutionEngine) GetOptions() *ExecutionEngineOptions {
	return e.Options
}

// InvokeLifecycleCallbacks invokes the onFinish and onError lifecycle callbacks if defined.
// Errors in callbacks are caught and logged, not propagated.
// TS equivalent: public async invokeLifecycleCallbacks(result: {...}): Promise<void>
func (e *BaseExecutionEngine) InvokeLifecycleCallbacks(result LifecycleCallbackParams) error {
	if e.Options == nil {
		return nil
	}

	// Always call onFinish if defined (for any terminal status)
	if e.Options.OnFinish != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if e.log != nil {
						e.log.Error(fmt.Sprintf("Error in onFinish callback: %v", r))
					}
				}
			}()

			err := e.Options.OnFinish(WorkflowFinishCallbackResult{
				Status:            result.Status,
				Result:            result.Result,
				Error:             asSerializedError(result.Error),
				Steps:             result.Steps,
				Tripwire:          asTripwireInfo(result.Tripwire),
				RunID:             result.RunID,
				WorkflowID:        result.WorkflowID,
				ResourceID:        result.ResourceID,
				GetInitData:       func() any { return result.Input },
				Mastra:            e.mastra,
				RequestContext:    result.RequestContext,
				Logger:            e.log,
				State:             result.State,
				StepExecutionPath: result.StepExecutionPath,
			})
			if err != nil && e.log != nil {
				e.log.Error(fmt.Sprintf("Error in onFinish callback: %v", err))
			}
		}()
	}

	// Call onError only for failure states (failed or tripwire)
	if e.Options.OnError != nil &&
		(result.Status == WorkflowRunStatusFailed || result.Status == WorkflowRunStatusTripwire) {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if e.log != nil {
						e.log.Error(fmt.Sprintf("Error in onError callback: %v", r))
					}
				}
			}()

			err := e.Options.OnError(WorkflowErrorCallbackInfo{
				Status:            result.Status,
				Error:             asSerializedError(result.Error),
				Steps:             result.Steps,
				Tripwire:          asTripwireInfo(result.Tripwire),
				RunID:             result.RunID,
				WorkflowID:        result.WorkflowID,
				ResourceID:        result.ResourceID,
				GetInitData:       func() any { return result.Input },
				Mastra:            e.mastra,
				RequestContext:    result.RequestContext,
				Logger:            e.log,
				State:             result.State,
				StepExecutionPath: result.StepExecutionPath,
			})
			if err != nil && e.log != nil {
				e.log.Error(fmt.Sprintf("Error in onError callback: %v", err))
			}
		}()
	}

	return nil
}

// asTripwireInfo attempts to convert a value to *StepTripwireInfo.
func asTripwireInfo(v any) *StepTripwireInfo {
	if v == nil {
		return nil
	}
	if info, ok := v.(*StepTripwireInfo); ok {
		return info
	}
	if info, ok := v.(StepTripwireInfo); ok {
		return &info
	}
	return nil
}

// asSerializedError converts an any value to *mastraerror.SerializedError.
// Returns nil if the input is nil. Handles *SerializedError, SerializedError,
// error, and arbitrary values via mastraerror.Serialize.
func asSerializedError(v any) *mastraerror.SerializedError {
	if v == nil {
		return nil
	}
	if se, ok := v.(*mastraerror.SerializedError); ok {
		return se
	}
	if se, ok := v.(mastraerror.SerializedError); ok {
		return &se
	}
	se := mastraerror.Serialize(v)
	return &se
}
