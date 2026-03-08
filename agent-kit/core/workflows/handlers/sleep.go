// Ported from: packages/core/src/workflows/handlers/sleep.ts
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Execute Sleep Params
// ---------------------------------------------------------------------------

// ExecuteSleepParams holds parameters for executing a sleep step.
// TS equivalent: export interface ExecuteSleepParams extends ObservabilityContext
type ExecuteSleepParams struct {
	WorkflowID        string
	RunID             string
	SerializedStepGraph []wf.SerializedStepFlowEntry
	Entry             SleepEntry
	PrevStep          wf.StepFlowEntry
	PrevOutput        any
	StepResults       map[string]wf.StepResult
	Resume            *ResumeParams
	ExecutionContext  *wf.ExecutionContext
	PubSub            events.PubSub
	AbortCtx          context.Context
	AbortCancel       context.CancelFunc
	RequestContext    *requestcontext.RequestContext
	OutputWriter      wf.OutputWriter
	Observability     *obstypes.ObservabilityContext
}

// SleepEntry holds data for a sleep flow entry.
type SleepEntry struct {
	Type     string              // "sleep"
	ID       string
	Duration *int64              // milliseconds
	Fn       wf.ExecuteFunction  // dynamic sleep function
}

// ---------------------------------------------------------------------------
// ExecuteSleep
// ---------------------------------------------------------------------------

// ExecuteSleep executes a sleep step.
// TS equivalent: export async function executeSleep(engine, params)
func ExecuteSleep(engine DefaultEngine, params ExecuteSleepParams) error {
	duration := params.Entry.Duration
	fn := params.Entry.Fn

	// If dynamic sleep function is provided, evaluate it
	if fn != nil {
		execParams := &wf.ExecuteFunctionParams{
			RunID:          params.RunID,
			WorkflowID:     params.WorkflowID,
			Mastra:         engine.GetMastra(),
			RequestContext: params.RequestContext,
			InputData:      params.PrevOutput,
			State:          params.ExecutionContext.State,
			SetState: func(state any) error {
				if m, ok := state.(map[string]any); ok {
					params.ExecutionContext.State = m
				}
				return nil
			},
			RetryCount: -1,
			GetInitData: func() any {
				if r, ok := params.StepResults["input"]; ok {
					return r.Output
				}
				return nil
			},
			GetStepResult: func(step any) any {
				return wf.GetStepResult(params.StepResults, step)
			},
			Suspend:       func(_ any, _ *wf.SuspendOptions) error { return nil },
			Bail:          func(_ any) {},
			Abort:         func() {},
			PubSub:        params.PubSub,
			StreamFormat:  params.ExecutionContext.Format,
			Engine:        engine.GetEngineContext(),
			AbortCtx:      params.AbortCtx,
			Observability: params.Observability,
		}

		result, err := fn(execParams)
		if err != nil {
			return fmt.Errorf("dynamic sleep function failed: %w", err)
		}

		// Convert result to duration
		if d, ok := result.(int64); ok {
			duration = &d
		} else if d, ok := result.(int); ok {
			d64 := int64(d)
			duration = &d64
		} else if d, ok := result.(float64); ok {
			d64 := int64(d)
			duration = &d64
		}
	}

	// Execute the sleep
	var d int64
	if duration != nil && *duration > 0 {
		d = *duration
	}
	return engine.ExecuteSleepDuration(d, params.Entry.ID, params.WorkflowID)
}

// ---------------------------------------------------------------------------
// Execute Sleep Until Params
// ---------------------------------------------------------------------------

// ExecuteSleepUntilParams holds parameters for executing a sleepUntil step.
// TS equivalent: export interface ExecuteSleepUntilParams extends ObservabilityContext
type ExecuteSleepUntilParams struct {
	WorkflowID        string
	RunID             string
	SerializedStepGraph []wf.SerializedStepFlowEntry
	Entry             SleepUntilEntry
	PrevStep          wf.StepFlowEntry
	PrevOutput        any
	StepResults       map[string]wf.StepResult
	Resume            *ResumeParams
	ExecutionContext  *wf.ExecutionContext
	PubSub            events.PubSub
	AbortCtx          context.Context
	AbortCancel       context.CancelFunc
	RequestContext    *requestcontext.RequestContext
	OutputWriter      wf.OutputWriter
	Observability     *obstypes.ObservabilityContext
}

// SleepUntilEntry holds data for a sleepUntil flow entry.
type SleepUntilEntry struct {
	Type string              // "sleepUntil"
	ID   string
	Date *time.Time
	Fn   wf.ExecuteFunction  // dynamic date function
}

// ---------------------------------------------------------------------------
// ExecuteSleepUntil
// ---------------------------------------------------------------------------

// ExecuteSleepUntil executes a sleepUntil step.
// TS equivalent: export async function executeSleepUntil(engine, params)
func ExecuteSleepUntil(engine DefaultEngine, params ExecuteSleepUntilParams) error {
	date := params.Entry.Date
	fn := params.Entry.Fn

	// If dynamic date function is provided, evaluate it
	if fn != nil {
		execParams := &wf.ExecuteFunctionParams{
			RunID:          params.RunID,
			WorkflowID:     params.WorkflowID,
			Mastra:         engine.GetMastra(),
			RequestContext: params.RequestContext,
			InputData:      params.PrevOutput,
			State:          params.ExecutionContext.State,
			SetState: func(state any) error {
				if m, ok := state.(map[string]any); ok {
					params.ExecutionContext.State = m
				}
				return nil
			},
			RetryCount: -1,
			GetInitData: func() any {
				if r, ok := params.StepResults["input"]; ok {
					return r.Output
				}
				return nil
			},
			GetStepResult: func(step any) any {
				return wf.GetStepResult(params.StepResults, step)
			},
			Suspend:       func(_ any, _ *wf.SuspendOptions) error { return nil },
			Bail:          func(_ any) {},
			Abort:         func() {},
			PubSub:        params.PubSub,
			StreamFormat:  params.ExecutionContext.Format,
			Engine:        engine.GetEngineContext(),
			AbortCtx:      params.AbortCtx,
			Observability: params.Observability,
		}

		result, err := fn(execParams)
		if err != nil {
			return fmt.Errorf("dynamic sleepUntil function failed: %w", err)
		}

		// Convert result to time.Time
		if t, ok := result.(time.Time); ok {
			date = &t
		} else if t, ok := result.(*time.Time); ok {
			date = t
		} else if s, ok := result.(string); ok {
			t, err := time.Parse(time.RFC3339, s)
			if err == nil {
				date = &t
			}
		}
	}

	if date == nil {
		return nil
	}

	return engine.ExecuteSleepUntilDate(*date, params.Entry.ID, params.WorkflowID)
}
