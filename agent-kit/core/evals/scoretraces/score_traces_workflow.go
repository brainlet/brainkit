// Ported from: packages/core/src/evals/scoreTraces/scoreTracesWorkflow.ts
package scoretraces

import (
	"context"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/evals"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	obstorage "github.com/brainlet/brainkit/agent-kit/core/storage/domains/observability"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/scores"
)

// ============================================================================
// Storage type aliases — imported from real packages (no circular dependency)
//
// Previous stubs claimed circular dependencies that do not actually exist:
//   - scoretraces -> storage/domains/observability: safe (observability does not import evals or scoretraces)
//   - scoretraces -> storage/domains/scores: safe (scores imports evals, but evals does not import scoretraces)
//   - scoretraces -> storage (parent): safe (storage does not import scoretraces)
// ============================================================================

// ObservabilityStore is the real observability storage interface.
// Replaces the previous stub that used map[string]any arguments.
type ObservabilityStore = obstorage.ObservabilityStorage

// ScoresStore is the real scores storage interface.
// Replaces the previous stub that used map[string]any arguments.
type ScoresStore = scores.ScoresStorage

// ScoreRowData is the canonical score record type from evals.
// Replaces the previous simplified local version that only had 4 fields.
type ScoreRowData = evals.ScoreRowData

// MastraStorage is a narrow interface for accessing domain-specific storage.
// This is NOT a false circular dependency stub — it is intentionally narrower
// than storage.MastraCompositeStore to decouple scoretraces from the full
// storage package. The real MastraCompositeStore.GetStore returns
// domains.StorageDomain (not (any, error)), so callers must adapt. Importing
// the storage package directly is safe (no cycle), but we keep a narrow
// interface here for minimal coupling.
type MastraStorage interface {
	// GetObservabilityStore returns the observability storage domain.
	GetObservabilityStore() (ObservabilityStore, error)
	// GetScoresStore returns the scores storage domain.
	GetScoresStore() (ScoresStore, error)
}

// WorkflowStepMastra is a narrow interface for the Mastra orchestrator
// available in workflow steps. Importing core/mastra directly is safe (no
// cycle), but we keep a narrow interface for minimal coupling.
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
	observabilityStore, err := params.Storage.GetObservabilityStore()
	if err != nil || observabilityStore == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_OBSERVABILITY_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Observability storage domain is not available",
		})
	}

	// Get the trace.
	traceResp, err := observabilityStore.GetTrace(ctx, obstorage.GetTraceArgs{
		TraceID: params.Target.TraceID,
	})
	if err != nil || traceResp == nil {
		return fmt.Errorf("trace not found for scoring, traceId: %s", params.Target.TraceID)
	}
	trace := traceResp

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

	// Convert score from any to float64.
	scoreVal := float64(0)
	switch s := result.Score.(type) {
	case float64:
		scoreVal = s
	case float32:
		scoreVal = float64(s)
	case int:
		scoreVal = float64(s)
	case int64:
		scoreVal = float64(s)
	}

	// Convert reason from any to string.
	reasonStr := ""
	if r, ok := result.Reason.(string); ok {
		reasonStr = r
	}

	// Convert step results from any to map[string]any.
	toMapAny := func(v any) map[string]any {
		if m, ok := v.(map[string]any); ok {
			return m
		}
		return nil
	}

	scorerResult := evals.SaveScorePayload{
		ScorerID: params.Scorer.ID(),
		EntityID: entityID,
		RunID:    result.RunID,
		Input:    result.Input,
		Output:   result.Output,
		Score:    scoreVal,
		Reason:   reasonStr,
		Scorer: map[string]any{
			"id":          params.Scorer.ID(),
			"name":        params.Scorer.Name(),
			"description": params.Scorer.Description(),
			"hasJudge":    params.Scorer.Judge() != nil,
		},
		Source:     evals.ScoringSourceTest,
		EntityType: evals.ScoringEntityType(targetSpan.SpanTyp),
		Entity: map[string]any{
			"traceId": targetSpan.TraceID,
			"spanId":  targetSpan.SpanID,
		},
		TraceID: params.Target.TraceID,
		SpanID:  params.Target.SpanID,

		// Include prompt data if available.
		PreprocessStepResult: toMapAny(result.PreprocessStepResult),
		PreprocessPrompt:     result.PreprocessPrompt,
		AnalyzeStepResult:    toMapAny(result.AnalyzeStepResult),
		AnalyzePrompt:        result.AnalyzePrompt,
		GenerateScorePrompt:  result.GenerateScorePrompt,
		GenerateReasonPrompt: result.GenerateReasonPrompt,
	}

	// Validate and save the score.
	savedScoreRecord, err := validateAndSaveScore(ctx, params.Storage, scorerResult)
	if err != nil {
		return err
	}

	// Attach score to span.
	return attachScoreToSpan(ctx, params.Storage, targetSpan, savedScoreRecord)
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
func validateAndSaveScore(ctx context.Context, storage MastraStorage, scorerResult evals.SaveScorePayload) (*ScoreRowData, error) {
	scoresStore, err := storage.GetScoresStore()
	if err != nil || scoresStore == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_SCORES_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Scores storage domain is not available",
		})
	}

	// TODO: Apply saveScorePayloadSchema validation once zod-equivalent is available.
	result, err := scoresStore.SaveScore(ctx, scorerResult)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// attachScoreToSpan attaches a score record to a span via its links field.
// Corresponds to TS: async function attachScoreToSpan({...})
func attachScoreToSpan(ctx context.Context, storage MastraStorage, span *SpanRecord, scoreRecord *ScoreRowData) error {
	observabilityStore, err := storage.GetObservabilityStore()
	if err != nil || observabilityStore == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_OBSERVABILITY_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Observability storage domain is not available",
		})
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

	return observabilityStore.UpdateSpan(ctx, obstorage.UpdateSpanArgs{
		SpanID:  span.SpanID,
		TraceID: span.TraceID,
		Updates: obstorage.UpdateSpanRecord{
			Links: updatedLinks,
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
