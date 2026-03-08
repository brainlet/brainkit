// Ported from: packages/core/src/datasets/experiment/executor.ts
package experiment

import (
	"context"
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// Agent is a stub for agent.Agent.
// STUB REASON: The real agent.Agent.Generate has entirely different params
// (uses agent-specific GenerateOptions, returns agent-specific results).
// Replacing requires a significant refactor of executeAgentTask.
type Agent interface {
	Generate(input any, opts map[string]any) (map[string]any, error)
}

// Workflow is a stub for workflows.Workflow.
// STUB REASON: The real workflows.Workflow.CreateRun has different param/return types.
// Replacing requires updating executeWorkflowTask to use real workflow types.
type Workflow interface {
	CreateRun(opts map[string]any) (WorkflowRun, error)
}

// WorkflowRun is a stub for a workflow run.
// STUB REASON: Same as Workflow — different Start signature in real type.
type WorkflowRun interface {
	Start(input map[string]any) (WorkflowRunResult, error)
}

// WorkflowRunResult is a stub for workflow run result.
// STUB REASON: Simplified version of the real workflow run result type.
type WorkflowRunResult struct {
	Status  string         `json:"status"`
	Result  any            `json:"result,omitempty"`
	Error   *WorkflowError `json:"error,omitempty"`
	TraceID string         `json:"traceId,omitempty"`
	Tripwire *TripwireInfo `json:"tripwire,omitempty"`
}

// WorkflowError is a stub for workflow error info.
type WorkflowError struct {
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// TripwireInfo is a stub for tripwire info.
type TripwireInfo struct {
	Reason string `json:"reason,omitempty"`
}

// Target is the union of target types supported for dataset execution.
// Agent and Workflow are Phase 2; scorer is Phase 4.
type Target interface{}

// ============================================================================
// Execution Result
// ============================================================================

// ExecutionResult is the result from executing a target against a dataset item.
type ExecutionResult struct {
	// Output is the output from the target (nil if failed).
	Output any `json:"output"`
	// Error is the structured error if execution failed.
	Error *ExecutionError `json:"error"`
	// TraceID is the trace ID from agent/workflow execution (empty for scorers or errors).
	TraceID string `json:"traceId"`
	// ScorerInput is the structured input for scorers (extracted from agent scoring data).
	ScorerInput ScorerRunInputForAgent `json:"scorerInput,omitempty"`
	// ScorerOutput is the structured output for scorers (extracted from agent scoring data).
	ScorerOutput ScorerRunOutputForAgent `json:"scorerOutput,omitempty"`
}

// ============================================================================
// executeScorer
// ============================================================================

// executeScorer executes a dataset item against a scorer (LLM-as-judge calibration).
// item.Input should contain exactly what the scorer expects — direct passthrough.
// For calibration: item.Input = { input, output, groundTruth } (user structures it).
func executeScorer(
	scorer MastraScorer,
	item DatasetItemInput,
) ExecutionResult {
	result, err := scorer.Run(item.Input)
	if err != nil {
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: err.Error()},
			TraceID: "",
		}
	}

	// Validate score is a number
	var score *float64
	if result.Score != nil {
		score = result.Score
	}

	var reason *string
	if result.Reason != nil {
		reason = result.Reason
	}

	output := map[string]any{
		"score":  score,
		"reason": reason,
	}

	return ExecutionResult{
		Output:  output,
		Error:   nil,
		TraceID: "", // Scorers don't produce traces
	}
}

// DatasetItemInput is the input shape for executing a dataset item.
type DatasetItemInput struct {
	Input       any `json:"input"`
	GroundTruth any `json:"groundTruth,omitempty"`
}

// ============================================================================
// ExecuteTarget
// ============================================================================

// ExecuteTarget executes a dataset item against a target (agent, workflow, scorer, processor).
// Phase 2: agent/workflow. Phase 4: scorer. Processor deferred.
func ExecuteTarget(
	ctx context.Context,
	target Target,
	targetType TargetType,
	item DatasetItemInput,
) ExecutionResult {
	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: ctx.Err().Error()},
			TraceID: "",
		}
	default:
	}

	switch targetType {
	case TargetTypeAgent:
		agent, ok := target.(Agent)
		if !ok {
			return ExecutionResult{
				Output:  nil,
				Error:   &ExecutionError{Message: "target is not an Agent"},
				TraceID: "",
			}
		}
		return executeAgent(ctx, agent, item)

	case TargetTypeWorkflow:
		wf, ok := target.(Workflow)
		if !ok {
			return ExecutionResult{
				Output:  nil,
				Error:   &ExecutionError{Message: "target is not a Workflow"},
				TraceID: "",
			}
		}
		return executeWorkflow(wf, item)

	case TargetTypeScorer:
		scorer, ok := target.(MastraScorer)
		if !ok {
			return ExecutionResult{
				Output:  nil,
				Error:   &ExecutionError{Message: "target is not a MastraScorer"},
				TraceID: "",
			}
		}
		return executeScorer(scorer, item)

	case TargetTypeProcessor:
		// Processor targets dropped from roadmap — not a core use case
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: fmt.Sprintf("Target type '%s' not yet supported.", targetType)},
			TraceID: "",
		}

	default:
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: fmt.Sprintf("Unknown target type: %s", targetType)},
			TraceID: "",
		}
	}
}

// ============================================================================
// executeAgent
// ============================================================================

// executeAgent executes a dataset item against an agent.
// Uses Generate() for both v1 and v2 models.
func executeAgent(
	_ context.Context,
	agent Agent,
	item DatasetItemInput,
) ExecutionResult {
	rawResult, err := agent.Generate(item.Input, map[string]any{
		"scorers":          map[string]any{},
		"returnScorerData": true,
	})
	if err != nil {
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: err.Error()},
			TraceID: "",
		}
	}

	traceID, _ := rawResult["traceId"].(string)

	// Extract scoring data if present
	var scorerInput ScorerRunInputForAgent
	var scorerOutput ScorerRunOutputForAgent
	if sd, ok := rawResult["scoringData"].(map[string]any); ok {
		scorerInput = sd["input"]
		scorerOutput = sd["output"]
	}

	// Only persist fields relevant to experiment evaluation — drop provider metadata
	trimmedOutput := map[string]any{
		"text":          rawResult["text"],
		"object":        rawResult["object"],
		"toolCalls":     rawResult["toolCalls"],
		"toolResults":   rawResult["toolResults"],
		"sources":       rawResult["sources"],
		"files":         rawResult["files"],
		"usage":         rawResult["usage"],
		"reasoningText": rawResult["reasoningText"],
		"traceId":       traceID,
		"error":         rawResult["error"],
	}

	return ExecutionResult{
		Output:       trimmedOutput,
		Error:        nil,
		TraceID:      traceID,
		ScorerInput:  scorerInput,
		ScorerOutput: scorerOutput,
	}
}

// ============================================================================
// executeWorkflow
// ============================================================================

// executeWorkflow executes a dataset item against a workflow.
// Creates a run with scorers disabled to avoid double-scoring.
func executeWorkflow(
	workflow Workflow,
	item DatasetItemInput,
) ExecutionResult {
	run, err := workflow.CreateRun(map[string]any{"disableScorers": true})
	if err != nil {
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: err.Error()},
			TraceID: "",
		}
	}

	result, err := run.Start(map[string]any{"inputData": item.Input})
	if err != nil {
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: err.Error()},
			TraceID: "",
		}
	}

	traceID := result.TraceID

	switch result.Status {
	case "success":
		return ExecutionResult{
			Output:  result.Result,
			Error:   nil,
			TraceID: traceID,
		}

	case "failed":
		msg := "Workflow failed"
		if result.Error != nil {
			msg = result.Error.Message
		}
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: msg, Stack: safeStack(result.Error)},
			TraceID: traceID,
		}

	case "tripwire":
		reason := "Unknown reason"
		if result.Tripwire != nil && result.Tripwire.Reason != "" {
			reason = result.Tripwire.Reason
		}
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: fmt.Sprintf("Workflow tripwire: %s", reason)},
			TraceID: traceID,
		}

	case "suspended":
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: "Workflow suspended - not yet supported in dataset experiments"},
			TraceID: traceID,
		}

	case "paused":
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: "Workflow paused - not yet supported in dataset experiments"},
			TraceID: traceID,
		}

	default:
		return ExecutionResult{
			Output:  nil,
			Error:   &ExecutionError{Message: fmt.Sprintf("Workflow ended with unexpected status: %s", result.Status)},
			TraceID: traceID,
		}
	}
}

// safeStack extracts the stack from a WorkflowError, returning "" if nil.
func safeStack(werr *WorkflowError) string {
	if werr == nil {
		return ""
	}
	return werr.Stack
}

// ErrAborted is returned when an operation is aborted.
var ErrAborted = errors.New("the operation was aborted")
