// Ported from: packages/core/src/datasets/experiment/scorer.ts
package experiment

import (
	"fmt"
	"log"
	"strings"
	"sync"

	mastracore "github.com/brainlet/brainkit/agent-kit/core/mastra"
	storagemod "github.com/brainlet/brainkit/agent-kit/core/storage"
)

// ============================================================================
// ResolveScorers
// ============================================================================

// ResolveScorers resolves scorers from a mixed slice of instances and string IDs.
// String IDs are looked up from Mastra's scorer registry.
func ResolveScorers(mastra Mastra, scorers []any) []MastraScorer {
	if len(scorers) == 0 {
		return nil
	}

	var result []MastraScorer
	for _, s := range scorers {
		switch v := s.(type) {
		case string:
			resolved := mastra.GetScorerByID(v)
			if resolved == nil {
				log.Printf("WARNING: Scorer not found: %s", v)
				continue
			}
			// Type-assert mastra.MastraScorer → experiment.MastraScorer.
			// The concrete *evals.MastraScorer satisfies both interfaces.
			localScorer, ok := resolved.(MastraScorer)
			if !ok {
				log.Printf("WARNING: Scorer %s does not implement experiment.MastraScorer interface", v)
				continue
			}
			result = append(result, localScorer)
		case mastracore.MastraScorer:
			// Received as mastra.MastraScorer, assert to local interface.
			localScorer, ok := v.(MastraScorer)
			if !ok {
				log.Printf("WARNING: Scorer does not implement experiment.MastraScorer interface: %T", v)
				continue
			}
			result = append(result, localScorer)
		case MastraScorer:
			result = append(result, v)
		default:
			log.Printf("WARNING: Invalid scorer type: %T", s)
		}
	}

	return result
}

// ============================================================================
// ScorerPromptMetadata
// ============================================================================

// ScorerPromptMetadata holds prompt/step metadata returned by scorer.Run() for DB persistence.
type ScorerPromptMetadata struct {
	GenerateScorePrompt  string         `json:"generateScorePrompt,omitempty"`
	GenerateReasonPrompt string         `json:"generateReasonPrompt,omitempty"`
	PreprocessStepResult map[string]any `json:"preprocessStepResult,omitempty"`
	PreprocessPrompt     string         `json:"preprocessPrompt,omitempty"`
	AnalyzeStepResult    map[string]any `json:"analyzeStepResult,omitempty"`
	AnalyzePrompt        string         `json:"analyzePrompt,omitempty"`
}

// ============================================================================
// RunScorersForItem
// ============================================================================

// ScorerItemInput holds the item fields needed for scoring.
type ScorerItemInput struct {
	Input       any            `json:"input"`
	GroundTruth any            `json:"groundTruth,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// RunScorersForItem runs all scorers for a single item result.
// Errors are isolated per scorer — one failing scorer doesn't affect others.
func RunScorersForItem(
	scorers []MastraScorer,
	item ScorerItemInput,
	output any,
	storage *MastraCompositeStore,
	runID string,
	targetType TargetType,
	targetID string,
	itemID string,
	scorerInput ScorerRunInputForAgent,
	scorerOutput ScorerRunOutputForAgent,
	traceID string,
) []ScorerResult {
	if len(scorers) == 0 {
		return nil
	}

	results := make([]ScorerResult, len(scorers))
	var wg sync.WaitGroup
	wg.Add(len(scorers))

	for i, scorer := range scorers {
		go func(idx int, s MastraScorer) {
			defer wg.Done()

			result, promptMeta := runScorerSafe(s, item, output, scorerInput, scorerOutput)

			// Persist score if storage available and score was computed
			if storage != nil && result.Score != nil {
				persistScore(storage, s, result, item, output, targetType, targetID, itemID, runID, traceID, promptMeta)
			}

			results[idx] = result
		}(i, scorer)
	}

	wg.Wait()
	return results
}

// ============================================================================
// runScorerSafe
// ============================================================================

// runScorerSafe runs a single scorer safely, catching any errors.
// Returns both the ScorerResult and prompt metadata for DB persistence.
func runScorerSafe(
	scorer MastraScorer,
	item ScorerItemInput,
	output any,
	scorerInput ScorerRunInputForAgent,
	scorerOutput ScorerRunOutputForAgent,
) (ScorerResult, ScorerPromptMetadata) {
	// Determine effective input/output for scorer
	effectiveInput := scorerInput
	if effectiveInput == nil {
		effectiveInput = item.Input
	}
	effectiveOutput := scorerOutput
	if effectiveOutput == nil {
		effectiveOutput = output
	}

	// Build scorer run input
	scorerRunInput := map[string]any{
		"input":       effectiveInput,
		"output":      effectiveOutput,
		"groundTruth": item.GroundTruth,
	}

	scoreResult, err := scorer.Run(scorerRunInput)
	if err != nil {
		errMsg := err.Error()
		return ScorerResult{
			ScorerID:   scorer.ID(),
			ScorerName: scorer.Name(),
			Score:      nil,
			Reason:     nil,
			Error:      &errMsg,
		}, ScorerPromptMetadata{}
	}

	if scoreResult == nil {
		errMsg := fmt.Sprintf(
			"Scorer %s (%s) returned invalid result: expected object, got nil",
			scorer.Name(), scorer.ID(),
		)
		return ScorerResult{
			ScorerID:   scorer.ID(),
			ScorerName: scorer.Name(),
			Score:      nil,
			Reason:     nil,
			Error:      &errMsg,
		}, ScorerPromptMetadata{}
	}

	return ScorerResult{
		ScorerID:   scorer.ID(),
		ScorerName: scorer.Name(),
		Score:      scoreResult.Score,
		Reason:     scoreResult.Reason,
		Error:      nil,
	}, ScorerPromptMetadata{
		// TODO: Extract prompt metadata from scorer result once scorer.Run()
		// returns these fields. Currently the ScorerRunResult stub doesn't
		// include prompt metadata.
	}
}

// ============================================================================
// persistScore
// ============================================================================

// hasSaveScore is an interface for stores that support saving scores.
type hasSaveScore interface {
	SaveScore(payload map[string]any) error
}

// persistScore persists a score to storage. Best-effort — errors are logged but not propagated.
// Implements the same logic as mastra/hooks.validateAndSaveScore:
// get the "scores" store, build payload, call SaveScore.
func persistScore(
	storage *MastraCompositeStore,
	scorer MastraScorer,
	result ScorerResult,
	item ScorerItemInput,
	output any,
	targetType TargetType,
	targetID string,
	itemID string,
	runID string,
	traceID string,
	promptMeta ScorerPromptMetadata,
) {
	if storage == nil || result.Score == nil {
		return
	}

	scoresStore := storage.GetStore(storagemod.DomainScores)
	if scoresStore == nil {
		log.Printf("WARNING: Scores storage domain not available, skipping score persistence (scorer=%s)", scorer.ID())
		return
	}

	saver, ok := scoresStore.(hasSaveScore)
	if !ok {
		log.Printf("WARNING: Scores store does not implement SaveScore (scorer=%s, store type=%T)", scorer.ID(), scoresStore)
		return
	}

	payload := map[string]any{
		"scorerId": scorer.ID(),
		"score":    *result.Score,
		"input":    item.Input,
		"output":   output,
		"entityType": strings.ToUpper(string(targetType)),
		"entityId":   itemID,
		"source":     "TEST",
		"runId":      runID,
		"scorer": map[string]any{
			"id":          scorer.ID(),
			"name":        scorer.Name(),
			"description": scorer.Description(),
			"hasJudge":    scorer.HasJudge(),
		},
		"entity": map[string]any{
			"id":   targetID,
			"name": targetID,
		},
	}
	if result.Reason != nil {
		payload["reason"] = *result.Reason
	}
	if item.Metadata != nil {
		payload["additionalContext"] = item.Metadata
	}
	if traceID != "" {
		payload["traceId"] = traceID
	}
	// Include prompt metadata if available.
	if promptMeta.GenerateScorePrompt != "" {
		payload["generateScorePrompt"] = promptMeta.GenerateScorePrompt
	}
	if promptMeta.GenerateReasonPrompt != "" {
		payload["generateReasonPrompt"] = promptMeta.GenerateReasonPrompt
	}
	if promptMeta.PreprocessStepResult != nil {
		payload["preprocessStepResult"] = promptMeta.PreprocessStepResult
	}

	if err := saver.SaveScore(payload); err != nil {
		log.Printf("WARNING: Failed to save score for scorer %s: %v", scorer.ID(), err)
	}
}

// ============================================================================
// Helpers
// ============================================================================

// isAbortError checks whether an error message indicates an abort.
func isAbortError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "abort") || strings.Contains(lower, "canceled") || strings.Contains(lower, "cancelled")
}
