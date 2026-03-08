// Ported from: packages/core/src/evals/scoreTraces/scoreTraces.ts
package scoretraces

import (
	"context"
	"fmt"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	mastracore "github.com/brainlet/brainkit/agent-kit/core/mastra"
)

// ============================================================================
// Interfaces
// ============================================================================

// Mastra is the narrow interface for the Mastra orchestrator used by scoretraces.
// No circular dependency: mastra imports evals (parent), but evals does not
// import evals/scoretraces. They are separate Go packages.
// Ported from: packages/core/src/evals/scoreTraces/scoreTraces.ts
type Mastra interface {
	// GetInternalWorkflow returns an internal workflow by ID.
	GetInternalWorkflow(id string) (mastracore.AnyWorkflow, error)
	// GetLogger returns the logger.
	GetLogger() logger.IMastraLogger
}

// InternalWorkflow defines the minimal contract for score trace execution.
// mastracore.AnyWorkflow is type-asserted to this interface at runtime because
// AnyWorkflow.CreateRun takes WorkflowCreateRunOpts, while scoretraces needs
// CreateRun(ctx context.Context) for its specific calling convention.
type InternalWorkflow interface {
	// CreateRun creates a new workflow run.
	CreateRun(ctx context.Context) (InternalWorkflowRun, error)
}

// InternalWorkflowRun is the minimal interface for a workflow run used by
// scoretraces. Simplified interface for workflow run execution.
type InternalWorkflowRun interface {
	// Start starts the workflow run.
	Start(ctx context.Context, opts map[string]any) error
}

// ============================================================================
// ScoreTracesTarget
// ============================================================================

// ScoreTracesTarget identifies a trace (and optional span) to score.
type ScoreTracesTarget struct {
	TraceID string `json:"traceId"`
	SpanID  string `json:"spanId,omitempty"`
}

// ============================================================================
// ScoreTraces
// ============================================================================

// ScoreTraces runs a scorer against one or more traces using the internal
// batch-scoring-traces workflow.
//
// Corresponds to TS: export async function scoreTraces({...})
func ScoreTraces(ctx context.Context, scorerID string, targets []ScoreTracesTarget, mastra Mastra) {
	wf, err := mastra.GetInternalWorkflow("__batch-scoring-traces")
	if err != nil {
		mastraError := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			Category: mastraerror.ErrorCategorySystem,
			Domain:   mastraerror.ErrorDomainScorer,
			ID:       "MASTRA_SCORER_FAILED_TO_RUN_TRACE_SCORING",
			Details: map[string]any{
				"scorerId": scorerID,
				"targets":  fmt.Sprintf("%v", targets),
			},
		})
		logger := mastra.GetLogger()
		if logger != nil {
			logger.TrackException(mastraError)
			logger.Error(mastraError.Error())
		}
		return
	}

	// Type-assert mastracore.AnyWorkflow to the local InternalWorkflow interface,
	// which provides the CreateRun(ctx) signature needed by scoretraces.
	workflow, ok := wf.(InternalWorkflow)
	if !ok {
		logger := mastra.GetLogger()
		if logger != nil {
			logger.Error("internal workflow does not implement InternalWorkflow interface")
		}
		return
	}

	run, err := workflow.CreateRun(ctx)
	if err != nil {
		mastraError := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			Category: mastraerror.ErrorCategorySystem,
			Domain:   mastraerror.ErrorDomainScorer,
			ID:       "MASTRA_SCORER_FAILED_TO_RUN_TRACE_SCORING",
			Details: map[string]any{
				"scorerId": scorerID,
				"targets":  fmt.Sprintf("%v", targets),
			},
		})
		logger := mastra.GetLogger()
		if logger != nil {
			logger.TrackException(mastraError)
			logger.Error(mastraError.Error())
		}
		return
	}

	err = run.Start(ctx, map[string]any{
		"inputData": map[string]any{
			"targets":  targets,
			"scorerId": scorerID,
		},
	})
	if err != nil {
		mastraError := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			Category: mastraerror.ErrorCategorySystem,
			Domain:   mastraerror.ErrorDomainScorer,
			ID:       "MASTRA_SCORER_FAILED_TO_RUN_TRACE_SCORING",
			Details: map[string]any{
				"scorerId": scorerID,
				"targets":  fmt.Sprintf("%v", targets),
			},
		})
		logger := mastra.GetLogger()
		if logger != nil {
			logger.TrackException(mastraError)
			logger.Error(mastraError.Error())
		}
	}
}
