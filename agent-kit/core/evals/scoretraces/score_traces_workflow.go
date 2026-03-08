// Ported from: packages/core/src/evals/scoreTraces/scoreTracesWorkflow.ts
package scoretraces

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/evals"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Stub types — kept local to avoid circular import dependencies
// ============================================================================

// MastraStorage is a stub for the storage.MastraCompositeStore interface.
// CIRCULAR DEP: Cannot import storage — storage/types.go imports evals.
// The real MastraCompositeStore.GetStore takes storage.DomainName and returns
// domains.StorageDomain (not (any, error)). Signature differs.
type MastraStorage interface {
	// GetStore returns a domain-specific store by name.
	GetStore(domain string) (any, error)
}

// ObservabilityStore is a stub for the observability storage domain.
// CIRCULAR DEP: Cannot import storage/domains/scores (which imports evals),
// and the real observability.ObservabilityStorage has different method signatures
// (GetTrace takes GetTraceArgs, returns *GetTraceResponse; UpdateSpan takes
// UpdateSpanArgs). Kept as simplified interface matching local call patterns.
type ObservabilityStore interface {
	// GetTrace retrieves a trace by traceId.
	GetTrace(args map[string]any) (*TraceRecord, error)
	// UpdateSpan updates a span.
	UpdateSpan(args map[string]any) error
}

// ScoresStore is a stub for the scores storage domain.
// CIRCULAR DEP: Cannot import storage/domains/scores — it imports evals.
// The real scores.ScoresStorage.SaveScore takes (ctx, scores.SaveScorePayload)
// and returns (*ScoreRowData, error), not (map[string]any) → (*SaveScoreResult, error).
type ScoresStore interface {
	// SaveScore saves a score record.
	SaveScore(payload map[string]any) (*SaveScoreResult, error)
}

// SaveScoreResult is the result of saving a score.
// CIRCULAR DEP: Cannot use evals.ScoreRowData directly here because this local
// ScoreRowData is a simplified version used only for span linking (fewer fields).
type SaveScoreResult struct {
	Score ScoreRowData `json:"score"`
}

// ScoreRowData is a simplified local version of evals.ScoreRowData.
// Only includes the fields needed for span linking (ID, Score, Scorer, CreatedAt).
// The real evals.ScoreRowData has 30+ fields. Using the full type would work
// but this simplified version keeps the span-linking code focused.
type ScoreRowData struct {
	ID        string         `json:"id"`
	Score     float64        `json:"score"`
	Scorer    map[string]any `json:"scorer"`
	CreatedAt any            `json:"createdAt"`
}

// WorkflowStepMastra is a stub for the Mastra interface available in workflow steps.
// CIRCULAR DEP: Cannot import core — core/hooks.go imports evals.
// This interface defines the minimal subset of core.Mastra methods needed by
// the score traces workflow steps.
type WorkflowStepMastra interface {
	GetLogger() logger.IMastraLogger
	GetStorage() MastraStorage
	GetScorerByID(id string) *evals.MastraScorer
}

// ============================================================================
// ScoreTracesWorkflowConfig
// ============================================================================

// ScoreTracesWorkflowConfig is the configuration for the score traces workflow.
// Corresponds to the Zod input schema in TS: z.object({ targets, scorerId })
type ScoreTracesWorkflowConfig struct {
	Targets  []ScoreTracesTarget `json:"targets"`
	ScorerID string              `json:"scorerId"`
}

// ============================================================================
// RunScorerOnTarget
// ============================================================================

// RunScorerOnTargetParams holds parameters for RunScorerOnTarget.
type RunScorerOnTargetParams struct {
	Storage              MastraStorage
	Scorer               *evals.MastraScorer
	Target               ScoreTracesTarget
	ObservabilityContext *obstypes.ObservabilityContext
}

// RunScorerOnTarget runs a scorer against a specific trace/span target.
//
// Corresponds to TS: export async function runScorerOnTarget({...})
func RunScorerOnTarget(ctx context.Context, params RunScorerOnTargetParams) error {
	// Get observability store.
	observabilityStoreRaw, err := params.Storage.GetStore("observability")
	if err != nil || observabilityStoreRaw == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_OBSERVABILITY_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Observability storage domain is not available",
		})
	}

	observabilityStore, ok := observabilityStoreRaw.(ObservabilityStore)
	if !ok {
		return errors.New("observability store does not implement ObservabilityStore interface")
	}

	// Get the trace.
	trace, err := observabilityStore.GetTrace(map[string]any{
		"traceId": params.Target.TraceID,
	})
	if err != nil || trace == nil {
		return fmt.Errorf("trace not found for scoring, traceId: %s", params.Target.TraceID)
	}

	// Find the target span.
	var targetSpan *SpanRecord
	if params.Target.SpanID != "" {
		for i := range trace.Spans {
			if trace.Spans[i].SpanID == params.Target.SpanID {
				targetSpan = &trace.Spans[i]
				break
			}
		}
	} else {
		// Find root span (parentSpanId == nil).
		for i := range trace.Spans {
			if trace.Spans[i].ParentSpanID == nil {
				targetSpan = &trace.Spans[i]
				break
			}
		}
	}

	if targetSpan == nil {
		spanIDStr := "Not provided"
		if params.Target.SpanID != "" {
			spanIDStr = params.Target.SpanID
		}
		return fmt.Errorf("span not found for scoring, traceId: %s, spanId: %s",
			params.Target.TraceID, spanIDStr)
	}

	// Build scorer run input.
	scorerRun := buildScorerRun(params.Scorer, params.ObservabilityContext, trace, targetSpan)

	// Execute the scorer.
	result, err := params.Scorer.Run(ctx, scorerRun)
	if err != nil {
		return err
	}

	// Build the scorer result for storage.
	entityID := "unknown"
	if targetSpan.EntityID != nil {
		entityID = *targetSpan.EntityID
	} else if targetSpan.EntityName != nil {
		entityID = *targetSpan.EntityName
	}

	scorerResult := map[string]any{
		"score":  result.Score,
		"reason": result.Reason,
		"input":  result.Input,
		"output": result.Output,
		"scorer": map[string]any{
			"id":          params.Scorer.ID(),
			"name":        params.Scorer.Name(),
			"description": params.Scorer.Description(),
			"hasJudge":    params.Scorer.Judge() != nil,
		},
		"traceId":    params.Target.TraceID,
		"spanId":     params.Target.SpanID,
		"entityId":   entityID,
		"entityType": string(targetSpan.SpanTyp),
		"entity": map[string]any{
			"traceId": targetSpan.TraceID,
			"spanId":  targetSpan.SpanID,
		},
		"source":   "TEST",
		"scorerId": params.Scorer.ID(),
		"runId":    result.RunID,

		// Include prompt data if available.
		"preprocessStepResult":  result.PreprocessStepResult,
		"preprocessPrompt":      result.PreprocessPrompt,
		"analyzeStepResult":     result.AnalyzeStepResult,
		"analyzePrompt":         result.AnalyzePrompt,
		"generateScorePrompt":   result.GenerateScorePrompt,
		"generateReasonPrompt":  result.GenerateReasonPrompt,
	}

	// Validate and save the score.
	savedScoreRecord, err := validateAndSaveScore(params.Storage, scorerResult)
	if err != nil {
		return err
	}

	// Attach score to span.
	return attachScoreToSpan(params.Storage, targetSpan, savedScoreRecord)
}

// buildScorerRun builds the ScorerRun input from the trace data.
// Corresponds to TS: function buildScorerRun({...}): ScorerRun
func buildScorerRun(
	scorer *evals.MastraScorer,
	obsCtx *obstypes.ObservabilityContext,
	trace *TraceRecord,
	targetSpan *SpanRecord,
) *evals.ScorerRun {
	scorerType := scorer.GetType()
	if scorerType == string(evals.ScorerTypeAgent) {
		input, output, err := TransformTraceToScorerInputAndOutput(trace)
		if err == nil {
			return &evals.ScorerRun{
				Input:                input,
				Output:               output,
				ObservabilityContext: obsCtx,
			}
		}
		// Fall through to default if transformation fails.
	}

	return &evals.ScorerRun{
		Input:                targetSpan.Input,
		Output:               targetSpan.Output,
		ObservabilityContext: obsCtx,
	}
}

// validateAndSaveScore validates and saves a scorer result to storage.
// Corresponds to TS: async function validateAndSaveScore({...})
func validateAndSaveScore(storage MastraStorage, scorerResult map[string]any) (*ScoreRowData, error) {
	scoresStoreRaw, err := storage.GetStore("scores")
	if err != nil || scoresStoreRaw == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_SCORES_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Scores storage domain is not available",
		})
	}

	scoresStore, ok := scoresStoreRaw.(ScoresStore)
	if !ok {
		return nil, errors.New("scores store does not implement ScoresStore interface")
	}

	// TODO: Apply saveScorePayloadSchema validation once zod-equivalent is available.
	result, err := scoresStore.SaveScore(scorerResult)
	if err != nil {
		return nil, err
	}

	return &result.Score, nil
}

// attachScoreToSpan attaches a score record to a span via its links field.
// Corresponds to TS: async function attachScoreToSpan({...})
func attachScoreToSpan(storage MastraStorage, span *SpanRecord, scoreRecord *ScoreRowData) error {
	observabilityStoreRaw, err := storage.GetStore("observability")
	if err != nil || observabilityStoreRaw == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_OBSERVABILITY_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Observability storage domain is not available",
		})
	}

	observabilityStore, ok := observabilityStoreRaw.(ObservabilityStore)
	if !ok {
		return errors.New("observability store does not implement ObservabilityStore interface")
	}

	// Build the link.
	var existingLinks []any
	if span.Links != nil {
		existingLinks = span.Links
	}

	scorerID := ""
	if scorer, ok := scoreRecord.Scorer["id"].(string); ok {
		scorerID = scorer
	}

	link := map[string]any{
		"type":       "score",
		"scoreId":    scoreRecord.ID,
		"scorerName": scorerID,
		"score":      scoreRecord.Score,
		"createdAt":  scoreRecord.CreatedAt,
	}

	updatedLinks := append(existingLinks, link)

	return observabilityStore.UpdateSpan(map[string]any{
		"spanId":  span.SpanID,
		"traceId": span.TraceID,
		"updates": map[string]any{
			"links": updatedLinks,
		},
	})
}

// ============================================================================
// ExecuteGetTraceStep — the workflow step logic
// ============================================================================

// ExecuteGetTraceStep implements the logic of the __process-trace-scoring
// workflow step. It retrieves the scorer, iterates over targets, and runs
// the scorer against each trace/span.
//
// Corresponds to TS: const getTraceStep = createStep({...}).execute
func ExecuteGetTraceStep(ctx context.Context, inputData ScoreTracesWorkflowConfig, mastra WorkflowStepMastra) error {
	logger := mastra.GetLogger()

	storage := mastra.GetStorage()
	if storage == nil {
		mastraErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_STORAGE_NOT_FOUND_FOR_TRACE_SCORING",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Storage not found for trace scoring",
			Details: map[string]any{
				"scorerId": inputData.ScorerID,
			},
		})
		if logger != nil {
			logger.Error(mastraErr.Error())
			logger.TrackException(mastraErr)
		}
		return nil
	}

	scorer := mastra.GetScorerByID(inputData.ScorerID)
	if scorer == nil {
		mastraErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_SCORER_NOT_FOUND_FOR_TRACE_SCORING",
			Domain:   mastraerror.ErrorDomainScorer,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Scorer not found for trace scoring",
			Details: map[string]any{
				"scorerId": inputData.ScorerID,
			},
		})
		if logger != nil {
			logger.Error(mastraErr.Error())
			logger.TrackException(mastraErr)
		}
		return nil
	}

	// Process targets with concurrency limit of 3 (matching TS pMap concurrency).
	const concurrency = 3
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, target := range inputData.Targets {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(t ScoreTracesTarget) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			err := RunScorerOnTarget(ctx, RunScorerOnTargetParams{
				Storage: storage,
				Scorer:  scorer,
				Target:  t,
			})
			if err != nil {
				mastraErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "MASTRA_SCORER_FAILED_TO_RUN_SCORER_ON_TRACE",
					Domain:   mastraerror.ErrorDomainScorer,
					Category: mastraerror.ErrorCategorySystem,
					Details: map[string]any{
						"scorerId": scorer.ID(),
						"spanId":   t.SpanID,
						"traceId":  t.TraceID,
					},
				})
				if logger != nil {
					logger.Error(mastraErr.Error())
					logger.TrackException(mastraErr)
				}
			}
		}(target)
	}

	wg.Wait()
	return nil
}

// ============================================================================
// ScoreTracesWorkflow — workflow definition
// ============================================================================

// ScoreTracesWorkflowID is the ID of the internal batch scoring traces workflow.
// Corresponds to TS: scoreTracesWorkflow with id: '__batch-scoring-traces'
const ScoreTracesWorkflowID = "__batch-scoring-traces"

// ScoreTracesWorkflow represents the workflow definition for batch scoring traces.
// In TS this is created via createWorkflow() and uses an evented workflow system.
// In Go, we define a struct that captures the same configuration.
//
// TODO: Wire this into the actual workflow system once the evented workflow
// package is ported. For now, the ExecuteGetTraceStep function can be called
// directly to execute the workflow logic.
type ScoreTracesWorkflow struct {
	ID string
}

// NewScoreTracesWorkflow creates the workflow definition.
func NewScoreTracesWorkflow() *ScoreTracesWorkflow {
	return &ScoreTracesWorkflow{
		ID: ScoreTracesWorkflowID,
	}
}

// Execute runs the workflow synchronously.
// This is a simplified version that directly executes the step.
func (w *ScoreTracesWorkflow) Execute(ctx context.Context, config ScoreTracesWorkflowConfig, mastra WorkflowStepMastra) error {
	return ExecuteGetTraceStep(ctx, config, mastra)
}
