// Ported from: packages/core/src/mastra/hooks.ts
package mastra

import (
	"context"
	"fmt"
	"sync"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/evals"
	"github.com/brainlet/brainkit/agent-kit/core/hooks"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/scores"
)

// createOnScorerHook creates the hook handler for ON_SCORER_RUN events.
// When a scorer runs, this hook validates and saves the score to storage,
// and optionally exports the score to trace exporters.
//
// Corresponds to TS: export function createOnScorerHook(mastra: Mastra)
func createOnScorerHook(mastra *Mastra) hooks.Handler {
	return func(event any) {
		hookData, ok := event.(*evals.ScoringHookInput)
		if !ok {
			mastra.GetLogger().Warn("createOnScorerHook: received non-ScoringHookInput event, skipping")
			return
		}

		stg := mastra.GetStorage()
		if stg == nil {
			mastra.GetLogger().Warn("Storage not found, skipping score validation and saving")
			return
		}

		entityID, _ := hookData.Entity["id"].(string)
		entityType := hookData.EntityType
		scorerID, _ := hookData.Scorer["id"].(string)

		if scorerID == "" {
			mastra.GetLogger().Warn("Scorer ID not found, skipping score validation and saving")
			return
		}

		scorerToUse := findScorer(mastra, entityID, string(entityType), scorerID)

		if scorerToUse == nil {
			err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "MASTRA_SCORER_NOT_FOUND",
				Domain:   mastraerror.ErrorDomainScorer,
				Category: mastraerror.ErrorCategoryUser,
				Text:     fmt.Sprintf("Scorer with ID %s not found", scorerID),
			})
			mastra.GetLogger().TrackException(err)
			mastra.GetLogger().Error(err.Error())
			return
		}

		// Run the scorer with hookData.
		// In TS: scorerToUse.scorer.run({ ...rest, input, output })
		// The MastraScorer.Run method is not yet fully ported, so we call it
		// if the interface supports it. The scorer result contains score + reason.
		runResult, runErr := runScorer(scorerToUse.Scorer, hookData)
		if runErr != nil {
			hookErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "MASTRA_SCORER_FAILED_TO_RUN_HOOK",
				Domain:   mastraerror.ErrorDomainScorer,
				Category: mastraerror.ErrorCategoryUser,
				Text:     fmt.Sprintf("Scorer %s failed to run: %v", scorerID, runErr),
				Details: map[string]any{
					"scorerId":   scorerID,
					"entityId":   entityID,
					"entityType": entityType,
				},
			})
			mastra.GetLogger().TrackException(hookErr)
			mastra.GetLogger().Error(hookErr.Error())
			return
		}

		// Extract span/trace IDs from the observability context
		var spanID, traceID string
		var currentSpan obstypes.Span
		if hookData.ObservabilityContext != nil {
			currentSpan = hookData.ObservabilityContext.Tracing.CurrentSpan
			if currentSpan != nil && currentSpan.IsValid() {
				spanID = currentSpan.ID()
				traceID = currentSpan.TraceID()
			}
		}

		// Determine if scorer has a judge
		hasJudge := false
		if judgeChecker, ok := scorerToUse.Scorer.(interface{ HasJudge() bool }); ok {
			hasJudge = judgeChecker.HasJudge()
		}

		// Build the payload to save
		// In TS: { ...rest, ...runResult, entityId, scorerId, spanId, traceId, scorer: {...}, metadata: {...} }
		hasStructuredOutput := false
		if hookData.StructuredOutput != nil {
			hasStructuredOutput = *hookData.StructuredOutput
		}

		scorerMap := make(map[string]any)
		for k, v := range hookData.Scorer {
			scorerMap[k] = v
		}
		scorerMap["hasJudge"] = hasJudge

		payload := &evals.SaveScorePayload{
			ScorerID:          scorerID,
			EntityID:          entityID,
			RunID:             hookData.RunID,
			Input:             hookData.Input,
			Output:            hookData.Output,
			AdditionalContext: hookData.AdditionalContext,
			RequestContext:    hookData.RequestContext,
			Source:            hookData.Source,
			Entity:            hookData.Entity,
			EntityType:        hookData.EntityType,
			SpanID:            spanID,
			TraceID:           traceID,
			ResourceID:        hookData.ResourceID,
			ThreadID:          hookData.ThreadID,
			Scorer:            scorerMap,
			Metadata: map[string]any{
				"structuredOutput": hasStructuredOutput,
			},
		}

		// Merge run result fields into the payload
		if runResult != nil {
			payload.Score = runResult.Score
			payload.Reason = runResult.Reason
			if runResult.ExtractStepResult != nil {
				payload.ExtractStepResult = runResult.ExtractStepResult
			}
			if runResult.ExtractPrompt != "" {
				payload.ExtractPrompt = runResult.ExtractPrompt
			}
			if runResult.AnalyzeStepResult != nil {
				payload.AnalyzeStepResult = runResult.AnalyzeStepResult
			}
			if runResult.AnalyzePrompt != "" {
				payload.AnalyzePrompt = runResult.AnalyzePrompt
			}
			if runResult.ReasonPrompt != "" {
				payload.ReasonPrompt = runResult.ReasonPrompt
			}
		}

		if err := validateAndSaveScore(stg, payload); err != nil {
			mastra.GetLogger().Error(fmt.Sprintf("Failed to validate and save score: %v", err))
			// Don't return — continue to export to trace if possible
		}

		// Export score to trace exporters
		if currentSpan != nil && spanID != "" && traceID != "" {
			obsInstance := currentSpan.ObservabilityInstance()
			if obsInstance != nil {
				exporters := obsInstance.GetExporters()
				scorerName := scorerToUse.Scorer.ID()
				var score float64
				var reason string
				if runResult != nil {
					score = runResult.Score
					reason = runResult.Reason
				}

				// Export to each exporter concurrently with bounded concurrency (3)
				sem := make(chan struct{}, 3)
				var wg sync.WaitGroup
				for _, exp := range exporters {
					wg.Add(1)
					sem <- struct{}{} // acquire semaphore
					go func(exporter obstypes.ObservabilityExporter) {
						defer wg.Done()
						defer func() { <-sem }() // release semaphore

						if err := exporter.AddScoreToTrace(obstypes.AddScoreToTraceArgs{
							TraceID:    traceID,
							SpanID:     spanID,
							Score:      score,
							Reason:     reason,
							ScorerName: scorerName,
							Metadata:   currentSpan.Metadata(),
						}); err != nil {
							mastra.GetLogger().Error(fmt.Sprintf("Failed to add score to trace via exporter: %v", err))
						}
					}(exp)
				}
				wg.Wait()
			}
		}
	}
}

// scorerRunResult holds the result of running a scorer.
// This mirrors the fields returned by MastraScorer.Run in TypeScript.
type scorerRunResult struct {
	Score             float64        `json:"score"`
	Reason            string         `json:"reason,omitempty"`
	ExtractStepResult map[string]any `json:"extractStepResult,omitempty"`
	ExtractPrompt     string         `json:"extractPrompt,omitempty"`
	AnalyzeStepResult map[string]any `json:"analyzeStepResult,omitempty"`
	AnalyzePrompt     string         `json:"analyzePrompt,omitempty"`
	ReasonPrompt      string         `json:"reasonPrompt,omitempty"`
}

// RunnableScorer is the interface a MastraScorer must implement to be runnable.
// This is checked at runtime since the base MastraScorer interface in core is a stub.
type RunnableScorer interface {
	Run(input *evals.ScoringHookInput) (*scorerRunResult, error)
}

// runScorer attempts to run the scorer. If the scorer implements RunnableScorer,
// it will be called directly. Otherwise, a nil result is returned with no error,
// indicating the scorer pipeline is not yet fully ported.
func runScorer(scorer MastraScorer, hookData *evals.ScoringHookInput) (*scorerRunResult, error) {
	if runnable, ok := scorer.(RunnableScorer); ok {
		return runnable.Run(hookData)
	}
	// Scorer does not implement Run — return a zero-value result.
	// This path is taken when the scorer pipeline is not yet fully ported.
	return &scorerRunResult{}, nil
}

// validateAndSaveScore validates a score payload and saves it to storage.
//
// Corresponds to TS: export async function validateAndSaveScore(storage: MastraStorage, payload: unknown)
func validateAndSaveScore(stg *storage.MastraCompositeStore, payload *evals.SaveScorePayload) error {
	if stg == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_SCORES_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Scores storage domain is not available",
		})
	}

	// Get the scores store from the composite store.
	// In TS: const scoresStore = await storage.getStore('scores');
	scoresStoreDomain := stg.GetStore(storage.DomainScores)
	if scoresStoreDomain == nil {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_SCORES_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Scores storage domain is not available",
		})
	}

	// Type-assert to ScoresStorage interface
	scoresStore, ok := scoresStoreDomain.(scores.ScoresStorage)
	if !ok {
		return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_SCORES_STORAGE_NOT_AVAILABLE",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Scores storage domain does not implement ScoresStorage interface",
		})
	}

	// Convert evals.SaveScorePayload to scores.SaveScorePayload.
	// The scores domain uses a simplified struct with a Snapshot catch-all
	// for the full payload data. Core fields are mapped directly.
	// scores.SaveScorePayload is a type alias for evals.SaveScorePayload,
	// so we can pass the payload directly.
	scoresPayload := scores.SaveScorePayload(*payload)

	// In TS: const payloadToSave = saveScorePayloadSchema.parse(payload);
	// In Go we skip zod validation and save directly.
	_, err := scoresStore.SaveScore(context.Background(), scoresPayload)
	return err
}

// scorerEntry pairs a scorer interface with its metadata.
// Mirrors the TS pattern: { scorer: MastraScorer }
type scorerEntry struct {
	Scorer MastraScorer
}

// findScorer searches for a scorer by ID across agents, workflows, and the mastra-level registry.
//
// Corresponds to TS: async function findScorer(mastra, entityId, entityType, scorerId)
func findScorer(mastra *Mastra, entityID string, entityType string, scorerID string) *scorerEntry {
	var scorerToUse *scorerEntry

	if entityType == "AGENT" {
		// Try code-defined agents first
		agent, err := mastra.GetAgentByID(entityID)
		if err == nil && agent != nil {
			agentScorers := agent.ListScorers()
			for _, entry := range agentScorers {
				if entry != nil && entry.Scorer != nil && entry.Scorer.ID() == scorerID {
					scorerToUse = &scorerEntry{Scorer: entry.Scorer}
					break
				}
			}
		}

		// If not found in code-defined agents, try stored agents via editor
		if scorerToUse == nil {
			// In TS: mastra.getEditor()?.agent.getById(entityId)
			// The editor interface in Go is minimal; stored agent lookup is not yet ported.
			// This path intentionally falls through to the mastra-registered scorer fallback.
		}
	} else if entityType == "WORKFLOW" {
		wf, err := mastra.GetWorkflowByID(entityID)
		if err == nil && wf != nil {
			wfScorers := wf.ListScorers()
			for _, entry := range wfScorers {
				// The real Workflow.ListScorers() returns map[string]any.
				// Type-assert each entry to MastraScorer for ID comparison.
				if scorer, ok := entry.(MastraScorer); ok && scorer != nil && scorer.ID() == scorerID {
					scorerToUse = &scorerEntry{Scorer: scorer}
					break
				}
			}
		}
	}

	// Fallback to mastra-registered scorer
	if scorerToUse == nil {
		scorer := mastra.GetScorerByID(scorerID)
		if scorer != nil {
			scorerToUse = &scorerEntry{Scorer: scorer}
		}
	}

	return scorerToUse
}
